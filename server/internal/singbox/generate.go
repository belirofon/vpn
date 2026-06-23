// Package singbox generates Sing-box compatible outbound JSON configs
// from parsed VpnConfig models. Each generated outbound can be wrapped
// into a full Sing-box configuration by the client.
package singbox

import (
	"encoding/json"
	"strings"

	"vpn-server/internal/model"
)

// GenerateOutbound builds a Sing-box outbound JSON object for the given config.
// Returns nil if the protocol is unsupported or fields are incomplete.
func GenerateOutbound(cfg *model.VpnConfig) *json.RawMessage {
	var outbound map[string]any

	switch cfg.Protocol {
	case "vless":
		outbound = buildVLESS(cfg)
	case "vmess":
		outbound = buildVMess(cfg)
	case "trojan":
		outbound = buildTrojan(cfg)
	case "ss":
		outbound = buildShadowsocks(cfg)
	default:
		return nil
	}

	if outbound == nil {
		return nil
	}

	data, err := json.Marshal(outbound)
	if err != nil {
		return nil
	}
	raw := json.RawMessage(data)
	return &raw
}

// buildTLS builds the TLS object for a Sing-box outbound.
func buildTLS(cfg *model.VpnConfig) map[string]any {
	tls := map[string]any{
		"enabled":  true,
		"insecure": true,
	}

	// SNI priority: sni param > host > server hostname (already in cfg.Server)
	if cfg.SNI != "" {
		tls["server_name"] = cfg.SNI
	} else if cfg.Host != "" {
		tls["server_name"] = cfg.Host
	}

	if cfg.ALPN != "" {
		tls["alpn"] = strings.Split(cfg.ALPN, ",")
	}

	if cfg.FP != "" {
		tls["fingerprint"] = cfg.FP
	}

	return tls
}

// buildRealityTLS builds TLS+REALITY object.
func buildRealityTLS(cfg *model.VpnConfig) map[string]any {
	tls := map[string]any{
		"enabled": true,
	}

	if cfg.SNI != "" {
		tls["server_name"] = cfg.SNI
	} else if cfg.Host != "" {
		tls["server_name"] = cfg.Host
	}

	if cfg.FP != "" {
		tls["fingerprint"] = cfg.FP
	}

	tls["reality"] = map[string]any{
		"enabled":    true,
		"public_key": cfg.Pbk,
		"short_id":   cfg.Sid,
	}

	return tls
}

// buildTransport builds the transport object (e.g., WebSocket).
func buildTransport(cfg *model.VpnConfig) map[string]any {
	// Skip for plain TCP
	if cfg.Network == "" || cfg.Network == "tcp" {
		return nil
	}

	transport := map[string]any{
		"type": cfg.Network,
	}

	if cfg.Path != "" {
		transport["path"] = cfg.Path
	}

	if cfg.Host != "" {
		transport["headers"] = map[string]any{
			"Host": cfg.Host,
		}
	}

	return transport
}

func buildVLESS(cfg *model.VpnConfig) map[string]any {
	outbound := map[string]any{
		"type":        "vless",
		"tag":         "proxy",
		"server":      cfg.Server,
		"server_port": cfg.Port,
		"uuid":        cfg.UUID,
		"flow":        "",
	}

	switch cfg.TLS {
	case "tls":
		outbound["tls"] = buildTLS(cfg)
	case "reality":
		outbound["tls"] = buildRealityTLS(cfg)
	}

	if transport := buildTransport(cfg); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func buildVMess(cfg *model.VpnConfig) map[string]any {
	outbound := map[string]any{
		"type":        "vmess",
		"tag":         "proxy",
		"server":      cfg.Server,
		"server_port": cfg.Port,
		"uuid":        cfg.UUID,
		"security":    "auto",
	}

	if cfg.TLS == "tls" {
		outbound["tls"] = buildTLS(cfg)
	}

	if transport := buildTransport(cfg); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func buildTrojan(cfg *model.VpnConfig) map[string]any {
	outbound := map[string]any{
		"type":        "trojan",
		"tag":         "proxy",
		"server":      cfg.Server,
		"server_port": cfg.Port,
		"password":    cfg.Password,
	}

	if cfg.TLS == "tls" {
		outbound["tls"] = buildTLS(cfg)
	}

	return outbound
}

func buildShadowsocks(cfg *model.VpnConfig) map[string]any {
	method, password := splitSSPassword(cfg.Password)

	// If method:password split failed (no colon) or method is empty,
	// skip generating sing-box config — client will fall back to raw_link.
	if method == "" {
		return nil
	}

	outbound := map[string]any{
		"type":        "shadowsocks",
		"tag":         "proxy",
		"server":      cfg.Server,
		"server_port": cfg.Port,
		"method":      method,
		"password":    password,
	}

	return outbound
}

// splitSSPassword splits "method:password" from the parsed Shadowsocks URI.
// The decoded format is always "method:password" — method never contains ':'.
// Returns empty method if the format is invalid (no colon found).
func splitSSPassword(raw string) (method, password string) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
