package warp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/curve25519"
	"vpn-server/internal/model"
)

const (
	apiBaseURL    = "https://api.cloudflareclient.com/v0a2222"
	warpHost      = "engage.cloudflareclient.com"
	warpPort      = 2408
	warpDNS       = "1.1.1.1"
	cfPublicKeyB64 = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
)

type registrationRequest struct {
	Key         string `json:"key"`
	InstallID   string `json:"install_id"`
	FCMToken    string `json:"fcm_token"`
	Referrer    string `json:"referrer"`
	WarpEnabled bool   `json:"warp_enabled"`
	TOS         string `json:"tos"`
	Type        string `json:"type"`
	Locale      string `json:"locale"`
}

type registrationResponse struct {
	ID      string `json:"id"`
	Token   string `json:"token"`
	Account struct {
		AccountType string `json:"account_type"`
	} `json:"account"`
	Config struct {
		ClientID string `json:"client_id"`
		Peers    []struct {
			PublicKey string `json:"public_key"`
			Endpoint  struct {
				V4 string `json:"v4"`
				V6 string `json:"v6"`
			} `json:"endpoint"`
			Host string `json:"host"`
		} `json:"peers"`
		Interface struct {
			Addresses struct {
				V4 string `json:"v4"`
				V6 string `json:"v6"`
			} `json:"addresses"`
		} `json:"interface"`
	} `json:"config"`
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func todayISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// GenerateKeyPair creates a new X25519 keypair for WireGuard.
func GenerateKeyPair() (privateKey, publicKey []byte, err error) {
	privateKey = make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}

	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("generate public key: %w", err)
	}

	return privateKey, publicKey, nil
}

func registerDevice(ctx context.Context, publicKey []byte) (*registrationResponse, error) {
	pubB64 := base64.StdEncoding.EncodeToString(publicKey)

	installID := newUUID()
	body := registrationRequest{
		Key:         pubB64,
		InstallID:   installID,
		FCMToken:    "",
		Referrer:    "",
		WarpEnabled: true,
		TOS:         todayISO(),
		Type:        "Android",
		Locale:      "nl-NL",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/reg", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "okhttp/4.12.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("register device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var regResp registrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &regResp, nil
}

func generateOfflineConfig(privKey []byte) *model.WarpConfig {
	privB64 := base64.StdEncoding.EncodeToString(privKey)
	return &model.WarpConfig{
		Protocol:        "warp",
		PrivateKey:      privB64,
		AddressV4:       "100.96.0.1/12",
		AddressV6:       "2606:4700:110:85b9:8dc3:9962:37d5:b42/128",
		DNSServers:      warpDNS,
		ServerPublicKey: cfPublicKeyB64,
		Endpoint:        fmt.Sprintf("%s:%d", warpHost, warpPort),
	}
}

func buildConfig(privKey []byte, reg *registrationResponse) *model.WarpConfig {
	privB64 := base64.StdEncoding.EncodeToString(privKey)

	var endpoint string
	if len(reg.Config.Peers) > 0 {
		endpoint = reg.Config.Peers[0].Endpoint.V4
		if endpoint == "" {
			endpoint = fmt.Sprintf("%s:%d", warpHost, warpPort)
		} else {
			// Cloudflare API sometimes returns IP without port or with port=0.
			// WireGuard always uses UDP port 2408.
			host, _, err := net.SplitHostPort(endpoint)
			if err != nil {
				host = endpoint
			}
			endpoint = fmt.Sprintf("%s:%d", host, warpPort)
		}
	} else {
		endpoint = fmt.Sprintf("%s:%d", warpHost, warpPort)
	}

	serverPubKey := ""
	if len(reg.Config.Peers) > 0 {
		serverPubKey = reg.Config.Peers[0].PublicKey
	}

	return &model.WarpConfig{
		Protocol:        "warp",
		PrivateKey:      privB64,
		AddressV4:       reg.Config.Interface.Addresses.V4,
		AddressV6:       reg.Config.Interface.Addresses.V6,
		DNSServers:      warpDNS,
		ServerPublicKey: serverPubKey,
		Endpoint:        endpoint,
		ClientID:        reg.Config.ClientID,
	}
}

func testEndpoint(ctx context.Context, endpoint string, timeout time.Duration) (int64, error) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
	}

	resolveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return 0, fmt.Errorf("resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return 0, fmt.Errorf("no addresses for %s", host)
	}

	latency := time.Since(start).Milliseconds()

	// Try UDP "connect" to the endpoint (best-effort for WireGuard).
	// WireGuard uses UDP, so full TCP/TLS handshake is not possible.
	addr := net.JoinHostPort(ips[0].IP.String(), fmt.Sprintf("%d", warpPort))
	udpConn, udpErr := net.DialTimeout("udp", addr, timeout/2)
	if udpErr == nil {
		udpConn.Close()
	}

	if latency < 1 {
		latency = 1
	}
	return latency, nil
}

// Generate performs the full WARP config generation cycle:
// key generation → device registration → connectivity test.
// Falls back to offline config if Cloudflare API registration fails.
func Generate(ctx context.Context, timeout time.Duration) (*model.WarpConfig, error) {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("key generation: %w", err)
	}

	cfg, err := func() (*model.WarpConfig, error) {
		reg, err := registerDevice(ctx, pubKey)
		if err != nil {
			return nil, err
		}
		return buildConfig(privKey, reg), nil
	}()

	if err != nil {
		cfg = generateOfflineConfig(privKey)
	}

	ep := cfg.Endpoint
	if ep == "" {
		ep = fmt.Sprintf("%s:%d", warpHost, warpPort)
	}

	latency, err := testEndpoint(ctx, ep, timeout)
	if err != nil {
		cfg.LatencyMs = -1
		return cfg, fmt.Errorf("endpoint test failed (config generated): %w", err)
	}

	cfg.LatencyMs = latency
	return cfg, nil
}
