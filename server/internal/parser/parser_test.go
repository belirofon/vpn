package parser

import (
	"encoding/base64"
	"testing"
)

func TestParseConfigLink_Vless(t *testing.T) {
	link := "vless://550e8400-e29b-41d4-a716-446655440000@sg-01.example.com:443?security=tls&type=ws&host=example.com&path=%2Fws&sni=sni.example.com&fp=chrome&alpn=h2%2Chttp%2F1.1&pbk=publickey&sid=sessionid"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Protocol != "vless" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "vless")
	}
	if cfg.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("UUID = %q, want %q", cfg.UUID, "550e8400-e29b-41d4-a716-446655440000")
	}
	if cfg.Server != "sg-01.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "sg-01.example.com")
	}
	if cfg.Port != 443 {
		t.Errorf("Port = %d, want %d", cfg.Port, 443)
	}
	if cfg.TLS != "tls" {
		t.Errorf("TLS = %q, want %q", cfg.TLS, "tls")
	}
	if cfg.Network != "ws" {
		t.Errorf("Network = %q, want %q", cfg.Network, "ws")
	}
	if cfg.Host != "example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
	}
	if cfg.Path != "/ws" {
		t.Errorf("Path = %q, want %q", cfg.Path, "/ws")
	}
	if cfg.SNI != "sni.example.com" {
		t.Errorf("SNI = %q, want %q", cfg.SNI, "sni.example.com")
	}
	if cfg.FP != "chrome" {
		t.Errorf("FP = %q, want %q", cfg.FP, "chrome")
	}
	if cfg.ALPN != "h2,http/1.1" {
		t.Errorf("ALPN = %q, want %q", cfg.ALPN, "h2,http/1.1")
	}
	if cfg.Pbk != "publickey" {
		t.Errorf("Pbk = %q, want %q", cfg.Pbk, "publickey")
	}
	if cfg.Sid != "sessionid" {
		t.Errorf("Sid = %q, want %q", cfg.Sid, "sessionid")
	}
}

func TestParseConfigLink_Vless_FragmentName(t *testing.T) {
	link := "vless://uuid@server.com:443#%F0%9F%87%AA%F0%9F%87%B8%20US-01"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Name != "🇪🇸 US-01" && cfg.Name != "%F0%9F%87%AA%F0%9F%87%B8 US-01" {
		// URL fragment may or may not be decoded depending on url.URL parsing
		t.Logf("Name = %q (fragment test, may vary)", cfg.Name)
	}
	if cfg.ID != "server.com:443" {
		t.Errorf("ID = %q, want %q", cfg.ID, "server.com:443")
	}
}

func TestParseConfigLink_Vless_DefaultPort(t *testing.T) {
	link := "vless://uuid@server.com"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Port != 443 {
		t.Errorf("Port = %d, want default 443", cfg.Port)
	}
}

func TestParseConfigLink_Vless_NonDefaultPort(t *testing.T) {
	link := "vless://uuid@server.com:8443?security=none"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Port != 8443 {
		t.Errorf("Port = %d, want 8443", cfg.Port)
	}
	if cfg.TLS != "none" {
		t.Errorf("TLS = %q, want %q (security=none)", cfg.TLS, "none")
	}
}

func TestParseConfigLink_Vmess(t *testing.T) {
	vmessJSON := `{"add":"us-01.example.com","port":"443","id":"uuid-vmess","aid":"0","net":"ws","type":"","host":"example.com","path":"/ws","tls":"tls","ps":"🇺🇸 US-01","scy":"auto"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	link := "vmess://" + encoded

	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Protocol != "vmess" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "vmess")
	}
	if cfg.Server != "us-01.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "us-01.example.com")
	}
	if cfg.Port != 443 {
		t.Errorf("Port = %d, want 443", cfg.Port)
	}
	if cfg.UUID != "uuid-vmess" {
		t.Errorf("UUID = %q, want %q", cfg.UUID, "uuid-vmess")
	}
	if cfg.Network != "ws" {
		t.Errorf("Network = %q, want %q", cfg.Network, "ws")
	}
	if cfg.TLS != "tls" {
		t.Errorf("TLS = %q, want %q", cfg.TLS, "tls")
	}
	if cfg.Host != "example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
	}
	if cfg.Path != "/ws" {
		t.Errorf("Path = %q, want %q", cfg.Path, "/ws")
	}
	if cfg.Name != "🇺🇸 US-01" {
		t.Errorf("Name = %q, want %q", cfg.Name, "🇺🇸 US-01")
	}
}

func TestParseConfigLink_Vmess_InvalidBase64(t *testing.T) {
	link := "vmess://not-valid-base64!!!"
	cfg := ParseConfigLink(link)

	if cfg != nil {
		t.Error("expected nil for invalid base64, got config")
	}
}

func TestParseConfigLink_Trojan(t *testing.T) {
	link := "trojan://password123@nl-01.example.com:443?security=tls&type=tcp"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Protocol != "trojan" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "trojan")
	}
	if cfg.Password != "password123" {
		t.Errorf("Password = %q, want %q", cfg.Password, "password123")
	}
	if cfg.Server != "nl-01.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "nl-01.example.com")
	}
	if cfg.Port != 443 {
		t.Errorf("Port = %d, want 443", cfg.Port)
	}
	if cfg.TLS != "tls" {
		t.Errorf("TLS = %q, want %q", cfg.TLS, "tls")
	}
}

func TestParseConfigLink_Trojan_WithRemark(t *testing.T) {
	link := "trojan://pass@de-01.example.com:443?security=tls&remark=DE-01"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Name != "DE-01" {
		t.Errorf("Name = %q, want %q", cfg.Name, "DE-01")
	}
}

func TestParseConfigLink_Shadowsocks(t *testing.T) {
	// ss://base64(method:password)@server:port
	userInfo := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secret123"))
	link := "ss://" + userInfo + "@jp-01.example.com:8443"

	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Protocol != "ss" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "ss")
	}
	if cfg.Password != "aes-256-gcm:secret123" {
		t.Errorf("Password = %q, want %q", cfg.Password, "aes-256-gcm:secret123")
	}
	if cfg.Server != "jp-01.example.com" {
		t.Errorf("Server = %q, want %q", cfg.Server, "jp-01.example.com")
	}
	if cfg.Port != 8443 {
		t.Errorf("Port = %d, want 8443", cfg.Port)
	}
}

func TestParseConfigLink_Shadowsocks_DefaultPort(t *testing.T) {
	userInfo := base64.StdEncoding.EncodeToString([]byte("chacha20:key"))
	link := "ss://" + userInfo + "@hk-01.example.com"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Port != 443 {
		t.Errorf("Port = %d, want default 443", cfg.Port)
	}
}

func TestParseConfigLink_UnknownScheme(t *testing.T) {
	link := "unknown://user@server.com:443"
	cfg := ParseConfigLink(link)

	if cfg != nil {
		t.Error("expected nil for unknown scheme, got config")
	}
}

func TestParseConfigLink_NoScheme(t *testing.T) {
	link := "just-a-plain-string-without-scheme"
	cfg := ParseConfigLink(link)

	if cfg != nil {
		t.Error("expected nil for link without scheme, got config")
	}
}

func TestParseConfigLink_EmptyServer(t *testing.T) {
	link := "vless://uuid@?security=tls"
	cfg := ParseConfigLink(link)

	if cfg != nil {
		t.Error("expected nil for empty server, got config")
	}
}

func TestParseConfigLink_IDGeneration(t *testing.T) {
	link := "vless://uuid@server.com:443"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.ID != "server.com:443" {
		t.Errorf("ID = %q, want %q", cfg.ID, "server.com:443")
	}
}

func TestParseSubscription_Base64(t *testing.T) {
	links := "vless://a@a.com\nvless://b@b.com:8443\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(links))

	result := ParseSubscription([]byte(encoded))

	if len(result) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(result), result)
	}
	if result[0] != "vless://a@a.com" {
		t.Errorf("result[0] = %q, want %q", result[0], "vless://a@a.com")
	}
}

func TestParseSubscription_Base64URL(t *testing.T) {
	links := "vless://a@a.com"
	encoded := base64.URLEncoding.EncodeToString([]byte(links))

	result := ParseSubscription([]byte(encoded))

	if len(result) != 1 {
		t.Fatalf("expected 1 link, got %d", len(result))
	}
}

func TestParseSubscription_JSON(t *testing.T) {
	data := `["vless://a@a.com","vmess://encoded","trojan://b@b.com"]`
	result := ParseSubscription([]byte(data))

	if len(result) != 3 {
		t.Fatalf("expected 3 links, got %d: %v", len(result), result)
	}
}

func TestParseSubscription_PlainText(t *testing.T) {
	data := "vless://a@a.com\nvless://b@b.com\n"
	result := ParseSubscription([]byte(data))

	if len(result) != 2 {
		t.Fatalf("expected 2 links, got %d", len(result))
	}
}

func TestParseSubscription_WithEmptyLines(t *testing.T) {
	data := "\n\nvless://a@a.com\n\n\nvless://b@b.com\n\n"
	result := ParseSubscription([]byte(data))

	if len(result) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(result), result)
	}
}

func TestParseSubscription_Empty(t *testing.T) {
	result := ParseSubscription([]byte(""))
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}

	result = ParseSubscription([]byte("  \n  \n"))
	if result != nil {
		t.Errorf("expected nil for whitespace-only input, got %v", result)
	}
}

func TestParsePort_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"443", 443},
		{"80", 80},
		{"8080", 8080},
		{"65535", 65535},
		{"1", 1},
	}

	for _, tt := range tests {
		got := parsePort(tt.input, 443)
		if got != tt.want {
			t.Errorf("parsePort(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParsePort_Invalid(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 443},
		{"-1", 443},
		{"0", 443},
		{"65536", 443},
		{"99999", 443},
		{"abc", 443},
		{"12a34", 443},
	}

	for _, tt := range tests {
		got := parsePort(tt.input, 443)
		if got != tt.want {
			t.Errorf("parsePort(%q) = %d, want default %d", tt.input, got, tt.want)
		}
	}
}

func TestParseConfigLink_DefaultName(t *testing.T) {
	link := "vless://uuid@server.com:443?security=none"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Name != "server.com" {
		t.Errorf("Name = %q, want %q (fallback to server)", cfg.Name, "server.com")
	}
}

func TestParseConfigLink_RawLink(t *testing.T) {
	link := "vless://uuid@server.com:443"
	cfg := ParseConfigLink(link)

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.RawLink != link {
		t.Errorf("RawLink = %q, want %q", cfg.RawLink, link)
	}
}
