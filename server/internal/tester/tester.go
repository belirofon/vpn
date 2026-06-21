package tester

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"vpn-server/internal/model"
	"vpn-server/internal/resolver"
)

// Test target for proxy verification — domain-based to also validate
// DNS resolution through the proxy (matching v2rayN's generate_204 approach).
const (
	testTargetDomain = "www.gstatic.com"
	testTargetPort   = 80
)

type pingResult struct {
	IP        string
	LatencyMs int64
	OK        bool
}

// TestConfigs tests connectivity for all configs in parallel.
// The context can be used to cancel the entire test batch early.
func TestConfigs(ctx context.Context, configs []*model.VpnConfig, timeout time.Duration, skipVerifyTLS bool) []*model.VpnConfig {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 500)

	results := make([]*model.VpnConfig, 0, len(configs))

	for _, cfg := range configs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return results
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(c *model.VpnConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			pr := pingServer(ctx, c, timeout, skipVerifyTLS)
			if !pr.OK {
				return
			}

			mu.Lock()
			c.Server = pr.IP
			c.LatencyMs = pr.LatencyMs
			results = append(results, c)
			mu.Unlock()
		}(cfg)
	}

	wg.Wait()
	return results
}

func pingServer(ctx context.Context, cfg *model.VpnConfig, timeout time.Duration, skipVerifyTLS bool) pingResult {
	ip, err := resolver.ResolveIP(ctx, cfg.Server, timeout)
	if err != nil {
		return pingResult{OK: false}
	}

	addr := net.JoinHostPort(ip, strconv.Itoa(cfg.Port))
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return pingResult{OK: false}
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	switch {
	// VLESS: full proxy test through a domain-based target (tests DNS + protocol + forwarding)
	case cfg.Protocol == "vless" && cfg.TLS != "reality":
		if !testVlessProxy(conn, cfg, timeout, skipVerifyTLS) {
			return pingResult{OK: false}
		}

	// Trojan: simple password-authenticated proxy test
	case cfg.Protocol == "trojan":
		if cfg.TLS == "tls" {
			tlsConn := tls.Client(conn, &tls.Config{
				ServerName:         cfg.Server,
				InsecureSkipVerify: skipVerifyTLS,
			})
			if err := tlsConn.Handshake(); err != nil {
				return pingResult{OK: false}
			}
			conn = tlsConn
		}
		if !testTrojanProxy(conn, cfg) {
			return pingResult{OK: false}
		}

	// VMESS / SS / other: TCP + TLS + WS check (protocol-level test not implemented)
	default:
		if cfg.TLS == "tls" {
			tlsConn := tls.Client(conn, &tls.Config{
				ServerName:         cfg.Server,
				InsecureSkipVerify: skipVerifyTLS,
			})
			if err := tlsConn.Handshake(); err != nil {
				return pingResult{OK: false}
			}
			conn = tlsConn

			if cfg.Network == "ws" || cfg.Network == "websocket" {
				if !testWsUpgrade(conn, cfg, timeout) {
					return pingResult{OK: false}
				}
			}
		}
	}

	latency := time.Since(start).Milliseconds()
	conn.SetDeadline(time.Time{})
	return pingResult{
		IP:        ip,
		LatencyMs: latency,
		OK:        true,
	}
}

// testTrojanProxy validates a Trojan proxy by sending the password
// followed by an HTTP request through the tunnel.
// Trojan wire format:  [password]\r\n[request]
func testTrojanProxy(conn net.Conn, cfg *model.VpnConfig) bool {
	req := []byte(cfg.Password + "\r\n" +
		"GET /generate_204 HTTP/1.1\r\n" +
		"Host: " + testTargetDomain + "\r\n" +
		"Connection: close\r\n\r\n")

	if _, err := conn.Write(req); err != nil {
		return false
	}

	resp := make([]byte, 256)
	n, err := conn.Read(resp)
	if err != nil {
		return false
	}
	return n > 8 && bytes.HasPrefix(resp[:n], []byte("HTTP/"))
}

func testWsUpgrade(conn net.Conn, cfg *model.VpnConfig, timeout time.Duration) bool {
	host := cfg.Host
	if host == "" {
		host = cfg.Server
	}

	path := cfg.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	conn.SetDeadline(time.Now().Add(timeout))

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, host)

	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}

	resp := make([]byte, 1024)
	n, err := conn.Read(resp)
	if err != nil {
		return false
	}

	return strings.Contains(string(resp[:n]), "101 Switching Protocols")
}
