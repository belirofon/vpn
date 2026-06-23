package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/gin-gonic/gin"
	"github.com/nwaples/rardecode"
	"vpn-server/internal/cache"
	"vpn-server/internal/model"
	"vpn-server/internal/parser"
	"vpn-server/internal/singbox"
)

const (
	maxFileSize    = 50 << 20
	maxArchiveSize = 200 << 20
	maxFileCount   = 500
)

type fileEntry struct {
	name string
	data []byte
}

func addToCache(cc *cache.ConfigCache, cfgs []model.VpnConfig) int {
	added := 0
	for _, cfg := range cfgs {
		cc.AddBestConfig(cfg)
		added++
	}
	return added
}

func processArchive(cc *cache.ConfigCache, data []byte, ext string) (int, []string, error) {
	var entries []fileEntry
	var err error

	switch ext {
	case ".zip":
		entries, err = extractZip(data)
	case ".7z":
		entries, err = extractSevenZip(data)
	case ".rar":
		entries, err = extractRar(data)
	default:
		return 0, nil, fmt.Errorf("unsupported archive type: %s", ext)
	}
	if err != nil {
		return 0, nil, fmt.Errorf("failed to extract %s: %w", ext, err)
	}
	return processFileEntries(cc, entries)
}

func processUnknownExt(cc *cache.ConfigCache, data []byte, ext string) (int, error) {
	links := parser.ParseSubscription(data)
	added := 0
	for _, link := range links {
		if parsed := parser.ParseConfigLink(link); parsed != nil {
			cc.AddBestConfig(*parsed)
			added++
		}
	}
	if added > 0 {
		return added, nil
	}
	return 0, fmt.Errorf("unsupported file type: %s", ext)
}

func processUploadedFile(cc *cache.ConfigCache, data []byte, filename string) (int, []string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".zip", ".7z", ".rar":
		return processArchive(cc, data, ext)

	case ".json":
		cfgs, err := processJSONFile(data)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to parse json: %w", err)
		}
		return addToCache(cc, cfgs), nil, nil

	case ".conf":
		cfgs, err := processConfFile(data)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to parse conf: %w", err)
		}
		return addToCache(cc, cfgs), nil, nil

	default:
		n, err := processUnknownExt(cc, data, ext)
		if err != nil {
			return 0, nil, err
		}
		return n, nil, nil
	}
}

// processFileEntries processes a list of extracted files (from archives).
func processFileEntries(cc *cache.ConfigCache, entries []fileEntry) (int, []string, error) {
	added := 0
	var errors []string

	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry.name))
		var cfgs []model.VpnConfig
		var err error

		switch ext {
		case ".json":
			cfgs, err = processJSONFile(entry.data)
		case ".conf":
			cfgs, err = processConfFile(entry.data)
		default:
			// Try as plain text
			links := parser.ParseSubscription(entry.data)
			for _, link := range links {
				if parsed := parser.ParseConfigLink(link); parsed != nil {
					cfgs = append(cfgs, *parsed)
				}
			}
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entry.name, err))
			continue
		}
		for _, cfg := range cfgs {
			cc.AddBestConfig(cfg)
			added++
		}
	}

	return added, errors, nil
}

// extractZip extracts a ZIP archive into file entries (recursively handles subdirectories).
func extractZip(data []byte) ([]fileEntry, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var entries []fileEntry
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := sanitizePath(f.Name)
		if name == "" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			slog.Warn("failed to open entry in zip", "name", f.Name, "error", err)
			continue
		}

		content, err := io.ReadAll(io.LimitReader(rc, maxFileSize))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name, err)
		}

		entries = append(entries, fileEntry{name: name, data: content})

		if len(entries) > maxFileCount {
			return nil, fmt.Errorf("archive contains too many files (max %d)", maxFileCount)
		}
	}
	return entries, nil
}

// extractSevenZip extracts a 7z archive into file entries.
func extractSevenZip(data []byte) ([]fileEntry, error) {
	reader, err := sevenzip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var entries []fileEntry
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := sanitizePath(f.Name)
		if name == "" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			slog.Warn("failed to open entry in 7z", "name", f.Name, "error", err)
			continue
		}

		content, err := io.ReadAll(io.LimitReader(rc, maxFileSize))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name, err)
		}

		entries = append(entries, fileEntry{name: name, data: content})

		if len(entries) > maxFileCount {
			return nil, fmt.Errorf("archive contains too many files (max %d)", maxFileCount)
		}
	}
	return entries, nil
}

// extractRar extracts a RAR archive into file entries.
func extractRar(data []byte) ([]fileEntry, error) {
	reader, err := rardecode.NewReader(bytes.NewReader(data), "")
	if err != nil {
		return nil, err
	}

	var entries []fileEntry
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("rar read error: %w", err)
		}
		if header.IsDir {
			continue
		}

		name := sanitizePath(header.Name)
		if name == "" {
			continue
		}

		content, err := io.ReadAll(io.LimitReader(reader, maxFileSize))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", header.Name, err)
		}

		entries = append(entries, fileEntry{name: name, data: content})

		if len(entries) > maxFileCount {
			return nil, fmt.Errorf("archive contains too many files (max %d)", maxFileCount)
		}
	}
	return entries, nil
}

// processJSONFile parses a JSON file into VPN configs.
// Supports: array of VpnConfig, single VpnConfig, sing-box config with outbounds.
func processJSONFile(data []byte) ([]model.VpnConfig, error) {
	// Try array of VpnConfig first
	var cfgs []model.VpnConfig
	if err := json.Unmarshal(data, &cfgs); err == nil {
		for i := range cfgs {
			normalizeConfig(&cfgs[i])
		}
		return cfgs, nil
	}

	// Try single VpnConfig
	var single model.VpnConfig
	if err := json.Unmarshal(data, &single); err == nil && single.Server != "" {
		normalizeConfig(&single)
		return []model.VpnConfig{single}, nil
	}

	// Try sing-box config with outbounds
	var sbConfig struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(data, &sbConfig); err == nil && len(sbConfig.Outbounds) > 0 {
		return extractSingboxOutbounds(sbConfig.Outbounds)
	}

	return nil, fmt.Errorf("json does not contain valid VPN configs")
}

// processConfFile parses a .conf file into VPN configs.
// Tries: sing-box JSON, plain text with proxy links.
func processConfFile(data []byte) ([]model.VpnConfig, error) {
	// Try as JSON (sing-box config often uses .conf extension)
	var sbConfig struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(data, &sbConfig); err == nil && len(sbConfig.Outbounds) > 0 {
		return extractSingboxOutbounds(sbConfig.Outbounds)
	}

	if cfg := processWireGuardConf(data); cfg != nil {
		return []model.VpnConfig{*cfg}, nil
	}

	// Fallback: treat as plain text with proxy links
	links := parser.ParseSubscription(data)
	var cfgs []model.VpnConfig
	for _, link := range links {
		if parsed := parser.ParseConfigLink(link); parsed != nil {
			cfgs = append(cfgs, *parsed)
		}
	}
	if len(cfgs) > 0 {
		return cfgs, nil
	}

	return nil, fmt.Errorf("conf file contains no recognizable configs")
}

type wgInterface struct {
	PrivateKey string
	Address    string
	DNS        string
	MTU        string
	I1         string // WARP-specific binary blob (Cloudflare reserved bytes)
}

type wgPeer struct {
	PublicKey           string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive string
}

func parseWGConf(data []byte) (iface wgInterface, peer wgPeer) {
	ifaceKeys := map[string]*string{
		"PrivateKey": &iface.PrivateKey,
		"Address":    &iface.Address,
		"DNS":        &iface.DNS,
		"MTU":        &iface.MTU,
		"I1":         &iface.I1,
	}
	peerKeys := map[string]*string{
		"PublicKey":           &peer.PublicKey,
		"AllowedIPs":          &peer.AllowedIPs,
		"Endpoint":            &peer.Endpoint,
		"PersistentKeepalive": &peer.PersistentKeepalive,
	}

	lines := strings.Split(string(data), "\n")
	var keys map[string]*string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		switch line {
		case "[Interface]":
			keys = ifaceKeys
			continue
		case "[Peer]":
			keys = peerKeys
			continue
		}
		if keys == nil {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if ptr, ok := keys[key]; ok {
			*ptr = val
		}
	}
	return
}

func parseHostPort(endpoint string) (host string, port int) {
	host = endpoint
	port = 2408
	if idx := strings.LastIndex(endpoint, ":"); idx >= 0 {
		host = endpoint[:idx]
		port = 0
		for _, c := range endpoint[idx+1:] {
			if c >= '0' && c <= '9' {
				port = port*10 + int(c-'0')
			}
		}
		if port == 0 {
			port = 2408
		}
	}
	return
}

func addrsWithCIDR(addrs string) []string {
	var result []string
	for _, a := range strings.Fields(addrs) {
		if !strings.Contains(a, "/") {
			if strings.Contains(a, ":") {
				a += "/128"
			} else {
				a += "/32"
			}
		}
		result = append(result, a)
	}
	return result
}

func parseAllowedIPs(allowed string) []string {
	ips := strings.Fields(allowed)
	if len(ips) == 0 {
		return []string{"0.0.0.0/0", "::/0"}
	}
	return ips
}

func processWireGuardConf(data []byte) *model.VpnConfig {
	iface, peer := parseWGConf(data)
	if iface.PrivateKey == "" || peer.PublicKey == "" || peer.Endpoint == "" {
		return nil
	}

	server, port := parseHostPort(peer.Endpoint)
	cfg := &model.VpnConfig{
		ID:       server + ":" + itoa(port),
		Protocol: "wireguard",
		Server:   server,
		Port:     port,
		Name:     server,
	}

	addrs := addrsWithCIDR(iface.Address)
	allowedIPs := parseAllowedIPs(peer.AllowedIPs)
	peerMap, clientID := buildWGPair(server, port, peer.PublicKey, allowedIPs, iface.I1)

	ep := map[string]any{
		"type":        "wireguard",
		"private_key": iface.PrivateKey,
		"address":     addrs,
		"peers":       []map[string]any{peerMap},
	}
	if clientID != "" {
		ep["client_id"] = clientID
	}
	ep["mtu"] = wgMTU(iface.MTU, server)

	if clientID != "" {
		cfg.Name = "WARP " + clientID
	} else if isWarpEndpoint(server) {
		cfg.Name = "WARP " + server
	}

	rawData, _ := json.Marshal(ep)
	raw := json.RawMessage(rawData)
	cfg.SingboxConfig = &raw
	return cfg
}

func buildWGPair(server string, port int, publicKey string, allowedIPs []string, i1 string) (peerMap map[string]any, clientID string) {
	peerMap = map[string]any{
		"address":     server,
		"port":        port,
		"public_key":  publicKey,
		"allowed_ips": allowedIPs,
	}
	if i1 == "" {
		return peerMap, ""
	}
	reserved, cid := parseWarpI1(i1)
	if reserved != nil {
		peerMap["reserved"] = reserved
	}
	if cid != "" {
		clientID = cid
	} else if isWarpEndpoint(server) {
		peerMap["reserved"] = []int{0, 0, 0}
	}
	return peerMap, clientID
}

func wgMTU(mtu string, server string) any {
	if mtu != "" {
		if v, err := parseInt(mtu); err == nil {
			return v
		}
	}
	if isWarpEndpoint(server) {
		return 1280
	}
	return nil
}

// isWarpEndpoint returns true if the server host is a known WARP endpoint.
func isWarpEndpoint(host string) bool {
	return host == "engage.cloudflareclient.com"
}

// hexDigit decodes a single hex character to nybble value.
func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// parseWarpI1 parses the WARP I1 field value (format: "<b 0xHEX...>") and
// extracts the reserved bytes (first 3 bytes) and a client_id hex string.
// Returns nil reserved if the field cannot be parsed.
func parseWarpI1(val string) (reserved []int, clientID string) {
	// Format: <b 0xHEXDATA>
	s := strings.TrimSpace(val)
	s = strings.TrimPrefix(s, "<b ")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimSpace(s)
	if s == "" || len(s)%2 != 0 {
		return nil, ""
	}

	// Decode hex bytes.
	raw := make([]byte, len(s)/2)
	for i := 0; i < len(raw); i++ {
		hi, ok1 := hexDigit(s[i*2])
		lo, ok2 := hexDigit(s[i*2+1])
		if !ok1 || !ok2 {
			return nil, ""
		}
		raw[i] = (hi << 4) | lo
	}
	if len(raw) < 3 {
		return nil, ""
	}

	// The first 3 bytes are the WireGuard reserved bytes.
	reserved = []int{int(raw[0]), int(raw[1]), int(raw[2])}

	// client_id: first 4 bytes as hex (8 chars).
	if len(raw) >= 4 {
		clientID = fmt.Sprintf("%02x%02x%02x%02x", raw[0], raw[1], raw[2], raw[3])
	}

	return reserved, clientID
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// extractSingboxOutbounds extracts VpnConfig entries from sing-box outbound objects.
// Looks for protocol, server, port, uuid, etc. in each outbound.
func extractSingboxOutbounds(outbounds []json.RawMessage) ([]model.VpnConfig, error) {
	var cfgs []model.VpnConfig

	for _, raw := range outbounds {
		var ob struct {
			Type     string          `json:"type"`
			Server   string          `json:"server"`
			Port     int             `json:"port"`
			UUID     string          `json:"uuid"`
			Password string          `json:"password"`
			Method   string          `json:"method"`
			TLS      json.RawMessage `json:"tls"`
		}
		if err := json.Unmarshal(raw, &ob); err != nil {
			continue
		}
		if ob.Server == "" || ob.Port == 0 {
			continue
		}

		cfg := model.VpnConfig{
			Server:   ob.Server,
			Port:     ob.Port,
			Protocol: ob.Type,
			UUID:     ob.UUID,
			Password: ob.Password,
			ID:       ob.Server + ":" + itoa(ob.Port),
			Name:     ob.Server,
		}

		// Try to extract TLS details
		if ob.TLS != nil {
			var tlsInfo struct {
				Enabled    bool   `json:"enabled"`
				Insecure   bool   `json:"insecure"`
				ServerName string `json:"server_name"`
			}
			if err := json.Unmarshal(ob.TLS, &tlsInfo); err == nil && tlsInfo.Enabled {
				cfg.TLS = "tls"
				cfg.SNI = tlsInfo.ServerName
			}
		}

		// Generate singbox config
		if sc := singbox.GenerateOutbound(&cfg); sc != nil {
			cfg.SingboxConfig = sc
		}

		cfgs = append(cfgs, cfg)
	}

	if len(cfgs) == 0 {
		return nil, fmt.Errorf("no valid outbounds found in sing-box config")
	}
	return cfgs, nil
}

// normalizeConfig fills in missing fields for a VpnConfig.
func normalizeConfig(cfg *model.VpnConfig) {
	if cfg.SingboxConfig == nil && cfg.Server != "" && cfg.Port > 0 {
		if sc := singbox.GenerateOutbound(cfg); sc != nil {
			cfg.SingboxConfig = sc
		}
	}
	if cfg.ID == "" {
		cfg.ID = cfg.Server + ":" + itoa(cfg.Port)
	}
	if cfg.Name == "" {
		cfg.Name = cfg.Server
	}
}

func sanitizePath(path string) string {
	path = filepath.ToSlash(path)
	var clean []string
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." || part == "" {
			continue
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/")
}

// AdminImportFile handles multipart file upload for importing configs.
func AdminImportFile(c *gin.Context, cc *cache.ConfigCache) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_file"})
		return
	}
	defer file.Close()

	if header.Size > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":   "file_too_large",
			"message": fmt.Sprintf("max file size is %d MB", maxFileSize/(1<<20)),
		})
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxFileSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read_failed"})
		return
	}

	added, fileErrors, err := processUploadedFile(cc, data, header.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "import_failed",
			"message": err.Error(),
		})
		return
	}

	resp := gin.H{"status": "imported", "added": added}
	if len(fileErrors) > 0 {
		resp["errors"] = fileErrors
	}
	c.JSON(http.StatusOK, resp)
}
