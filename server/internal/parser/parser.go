package parser

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"vpn-server/internal/model"
)

// ParseSubscription parses raw subscription data and returns config links.
// Supports: base64-encoded list, plain text (one link per line), JSON array.
func ParseSubscription(data []byte) []string {
	text := string(data)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	// Try base64 decode
	if decoded, err := base64.StdEncoding.DecodeString(text); err == nil {
		lines := strings.Split(string(decoded), "\n")
		var links []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				links = append(links, line)
			}
		}
		if len(links) > 0 {
			return links
		}
	}

	// Try base64 URL-safe decode
	if decoded, err := base64.URLEncoding.DecodeString(text); err == nil {
		lines := strings.Split(string(decoded), "\n")
		var links []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				links = append(links, line)
			}
		}
		if len(links) > 0 {
			return links
		}
	}

	// Try JSON array of strings
	var jsonLinks []string
	if err := json.Unmarshal(data, &jsonLinks); err == nil {
		return jsonLinks
	}

	// Fallback: plain text, one link per line
	lines := strings.Split(text, "\n")
	var links []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			links = append(links, line)
		}
	}
	return links
}

// ParseConfigLink parses a single config link and returns a VpnConfig.
// Supports: vless://, vmess://, trojan://, ss://
func ParseConfigLink(link string) *model.VpnConfig {
	if !strings.Contains(link, "://") {
		return nil
	}

	u, err := url.Parse(link)
	if err != nil {
		return nil
	}

	cfg := &model.VpnConfig{
		RawLink:  link,
		Protocol: u.Scheme,
	}

	switch u.Scheme {
	case "vless":
		parseVless(cfg, u)
	case "vmess":
		parseVmess(cfg, link, u)
	case "trojan":
		parseTrojan(cfg, u)
	case "ss":
		parseShadowsocks(cfg, u)
	default:
		return nil
	}

	if cfg.Server == "" || cfg.Port == 0 {
		return nil
	}

	cfg.ID = cfg.Server + ":" + strconv.Itoa(cfg.Port)
	if cfg.Name == "" {
		cfg.Name = cfg.Server
	}

	return cfg
}

func parseVless(cfg *model.VpnConfig, u *url.URL) {
	cfg.UUID = u.User.String()
	cfg.Server = u.Hostname()
	cfg.Port = parsePort(u.Port(), 443)

	q := u.Query()
	cfg.TLS = q.Get("security")
	cfg.Network = q.Get("type")
	cfg.Host = q.Get("host")
	cfg.Path = q.Get("path")
	cfg.SNI = q.Get("sni")
	cfg.FP = q.Get("fp")
	cfg.ALPN = q.Get("alpn")
	cfg.Pbk = q.Get("pbk")
	cfg.Sid = q.Get("sid")

	// Name priority: remark query > fragment (after #) > host
	if v := q.Get("remark"); v != "" {
		cfg.Name = v
	} else if frag := strings.TrimSpace(u.Fragment); frag != "" {
		cfg.Name = frag
	}
}

func parseVmess(cfg *model.VpnConfig, raw string, u *url.URL) {
	// vmess://base64encodedjson
	encoded := strings.TrimPrefix(raw, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return
		}
	}

	var vmess struct {
		Add  string `json:"add"`
		Port string `json:"port"`
		ID   string `json:"id"`
		Aid  string `json:"aid"`
		Net  string `json:"net"`
		Type string `json:"type"`
		Host string `json:"host"`
		Path string `json:"path"`
		TLS  string `json:"tls"`
		Ps   string `json:"ps"`
		Scy  string `json:"scy"`
	}

	if err := json.Unmarshal(decoded, &vmess); err != nil {
		return
	}

	cfg.Server = vmess.Add
	cfg.Port = parsePort(vmess.Port, 443)
	cfg.UUID = vmess.ID
	cfg.TLS = vmess.TLS
	cfg.Network = vmess.Net
	cfg.Host = vmess.Host
	cfg.Path = vmess.Path
	cfg.Name = vmess.Ps
}

func parseTrojan(cfg *model.VpnConfig, u *url.URL) {
	cfg.Password = u.User.String()
	cfg.Server = u.Hostname()
	cfg.Port = parsePort(u.Port(), 443)

	q := u.Query()
	cfg.TLS = q.Get("security")
	cfg.Network = q.Get("type")
	if v := q.Get("remark"); v != "" {
		cfg.Name = v
	}
}

func parseShadowsocks(cfg *model.VpnConfig, u *url.URL) {
	// ss://base64(method:password)@server:port
	userInfo := u.User.String()
	if decoded, err := base64.StdEncoding.DecodeString(userInfo); err == nil {
		cfg.Password = string(decoded)
	} else if decoded, err := base64.URLEncoding.DecodeString(userInfo); err == nil {
		cfg.Password = string(decoded)
	}

	cfg.Server = u.Hostname()
	cfg.Port = parsePort(u.Port(), 443)

	q := u.Query()
	if v := q.Get("remark"); v != "" {
		cfg.Name = v
	}
}

func parsePort(portStr string, defaultPort int) int {
	if portStr == "" {
		return defaultPort
	}
	p := 0
	for _, c := range portStr {
		if c < '0' || c > '9' {
			return defaultPort
		}
		p = p*10 + int(c-'0')
	}
	if p <= 0 || p > 65535 {
		return defaultPort
	}
	return p
}
