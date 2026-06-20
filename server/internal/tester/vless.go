package tester

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"vpn-server/internal/model"
)

const wsKey = "dGhlIHNhbXBsZSBub25jZQ=="

func testVlessProxy(conn net.Conn, cfg *model.VpnConfig, timeout time.Duration, skipVerifyTLS bool) bool {
	conn.SetDeadline(time.Now().Add(timeout))

	var tlsConn net.Conn = conn

	switch cfg.TLS {
	case "tls":
		tc := tls.Client(conn, &tls.Config{
			ServerName:         cfg.Server,
			InsecureSkipVerify: skipVerifyTLS,
		})
		if err := tc.Handshake(); err != nil {
			return false
		}
		tlsConn = tc
	case "reality":
		tc, err := testRealityHandshake(conn, cfg, timeout)
		if err != nil {
			return false
		}
		tlsConn = tc

	}

	// For WebSocket transport: do WS upgrade, then send VLESS via WS frames
	if cfg.Network == "ws" || cfg.Network == "websocket" {
		return testVlessOverWS(tlsConn, cfg, timeout)
	}

	// For TCP/other transports: send VLESS request directly
	return testVlessDirect(tlsConn, cfg)
}

// testVlessOverWS does WS upgrade then sends VLESS request through WebSocket frames.
func testVlessOverWS(conn net.Conn, cfg *model.VpnConfig, timeout time.Duration) bool {
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

	// HTTP WebSocket upgrade
	reqStr := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, host, wsKey)
	if _, err := conn.Write([]byte(reqStr)); err != nil {
		return false
	}

	resp := make([]byte, 1024)
	n, err := conn.Read(resp)
	if err != nil || !strings.Contains(string(resp[:n]), "101 Switching Protocols") {
		return false
	}

	if !sendVlessRequest(conn, cfg) {
		return false
	}

	// Send HTTP GET as WS binary frame
	httpReq := buildHttpTestRequest()
	if err := writeWSFrame(conn, httpReq); err != nil {
		return false
	}

	// Read WS frame response
	data, err := readWSFrame(conn)
	if err != nil {
		return false
	}

	return len(data) > 8 && bytes.HasPrefix(data, []byte("HTTP/"))
}

// testVlessDirect sends VLESS request directly over the TLS/TCP connection.
func testVlessDirect(conn net.Conn, cfg *model.VpnConfig) bool {
	if !sendVlessRequest(conn, cfg) {
		return false
	}

	httpReq := buildHttpTestRequest()
	if _, err := conn.Write(httpReq); err != nil {
		return false
	}

	resp := make([]byte, 256)
	n, err := conn.Read(resp)
	if err != nil {
		return false
	}

	return n > 8 && bytes.HasPrefix(resp[:n], []byte("HTTP/"))
}

// sendVlessRequest builds and sends a VLESS protocol handshake
// targeting the configured test domain (address type 0x03, domain-based).
// This validates both the VLESS protocol and DNS resolution through the proxy.
func sendVlessRequest(conn net.Conn, cfg *model.VpnConfig) bool {
	uuid := parseUUID(cfg.UUID)
	if uuid == nil {
		return false
	}

	domain := testTargetDomain
	req := make([]byte, 0, 1+16+1+1+1+len(domain)+2)
	req = append(req, 0x00)                        // version
	req = append(req, uuid...)                      // UUID (16 bytes)
	req = append(req, 0x01)                         // command: TCP
	req = append(req, 0x03)                         // address type: domain
	req = append(req, byte(len(domain)))            // domain length
	req = append(req, []byte(domain)...)             // domain
	req = append(req, 0x00, byte(testTargetPort))   // port

	if _, err := conn.Write(req); err != nil {
		return false
	}
	return true
}

// buildHttpTestRequest returns an HTTP/1.1 GET request for the test target.
// Uses /generate_204 path matching v2rayN's approach (returns 204 No Content).
func buildHttpTestRequest() []byte {
	return []byte("GET /generate_204 HTTP/1.1\r\n" +
		"Host: " + testTargetDomain + "\r\n" +
		"Connection: close\r\n\r\n")
}

// writeWSFrame sends a binary WebSocket frame (masked, client→server).
func writeWSFrame(conn net.Conn, payload []byte) error {
	// FIN + binary opcode
	frame := []byte{0x82}

	// Mask + length
	maskKey := make([]byte, 4)
	if _, err := rand.Read(maskKey); err != nil {
		return err
	}

	length := len(payload)
	if length < 126 {
		frame = append(frame, byte(0x80|length))
	} else if length < 65536 {
		frame = append(frame, 0x80|126, byte(length>>8), byte(length))
	} else {
		frame = append(frame, 0x80|127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(length>>(i*8)))
		}
	}
	frame = append(frame, maskKey...)

	// Masked payload
	masked := make([]byte, length)
	for i := 0; i < length; i++ {
		masked[i] = payload[i] ^ maskKey[i%4]
	}
	frame = append(frame, masked...)

	_, err := conn.Write(frame)
	return err
}

// readWSFrame reads a binary WebSocket frame (unmasked, server→client).
func readWSFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	// opcode & fin
	if header[0]&0x0F != 0x02 && header[0]&0x0F != 0x01 {
		return nil, fmt.Errorf("unexpected WS opcode: %d", header[0]&0x0F)
	}

	length := int64(header[1] & 0x7F)
	offset := 2

	switch {
	case length == 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return nil, err
		}
		length = int64(ext[0])<<8 | int64(ext[1])
		offset += 2
	case length == 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(conn, ext); err != nil {
			return nil, err
		}
		length = 0
		for i := 0; i < 8; i++ {
			length = length<<8 | int64(ext[i])
		}
		offset += 8
	}

	if length < 0 || length > 65536 {
		return nil, fmt.Errorf("unexpected WS frame length: %d", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return data, nil
}

// testRealityHandshake performs a REALITY TLS handshake using uTLS
// to mimic a browser fingerprint. Returns the uTLS connection for further
// VLESS protocol testing.
func testRealityHandshake(conn net.Conn, cfg *model.VpnConfig, timeout time.Duration) (net.Conn, error) {
	sni := cfg.SNI
	if sni == "" {
		sni = cfg.Server
	}

	uconn := utls.UClient(conn, &utls.Config{
		ServerName: sni,
	}, getClientHelloID(cfg.FP))

	if err := uconn.SetDeadline(time.Now().Add(timeout)); err != nil {
		uconn.Close()
		return nil, err
	}

	if err := uconn.Handshake(); err != nil {
		uconn.Close()
		return nil, err
	}

	return uconn, nil
}

// getClientHelloID maps a fingerprint string to a uTLS ClientHelloID.
// Supports common REALITY fingerprint values. Defaults to Chrome if unknown.
func getClientHelloID(fp string) utls.ClientHelloID {
	switch strings.ToLower(fp) {
	case "chrome":
		return utls.HelloChrome_Auto
	case "firefox":
		return utls.HelloFirefox_Auto
	case "safari":
		return utls.HelloSafari_Auto
	case "ios":
		return utls.HelloIOS_Auto
	case "android":
		return utls.HelloAndroid_11_OkHttp
	case "edge":
		return utls.HelloEdge_Auto
	case "random", "randomized":
		return utls.HelloRandomized
	default:
		return utls.HelloChrome_Auto
	}
}

func parseUUID(s string) []byte {
	if len(s) != 36 {
		return nil
	}
	raw := make([]byte, 16)
	for i := 0; i < 4; i++ {
		raw[i] = byte(unhex(s[i*2]))<<4 | byte(unhex(s[i*2+1]))
	}
	raw[4] = byte(unhex(s[9]))<<4 | byte(unhex(s[10]))
	raw[5] = byte(unhex(s[11]))<<4 | byte(unhex(s[12]))
	raw[6] = byte(unhex(s[14]))<<4 | byte(unhex(s[15]))
	raw[7] = byte(unhex(s[16]))<<4 | byte(unhex(s[17]))
	raw[8] = byte(unhex(s[19]))<<4 | byte(unhex(s[20]))
	raw[9] = byte(unhex(s[21]))<<4 | byte(unhex(s[22]))
	for i := 0; i < 6; i++ {
		raw[10+i] = byte(unhex(s[24+i*2]))<<4 | byte(unhex(s[24+i*2+1]))
	}
	return raw
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
