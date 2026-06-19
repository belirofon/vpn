package resolver

import (
	"testing"
	"time"
)

func TestResolveIP_Localhost(t *testing.T) {
	ip, err := ResolveIP("localhost", 5*time.Second)
	if err != nil {
		t.Fatalf("ResolveIP(localhost) error: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Logf("ResolveIP(localhost) = %q (may vary by OS)", ip)
	}
}

func TestResolveIP_IPv4Loopback(t *testing.T) {
	ip, err := ResolveIP("127.0.0.1", 5*time.Second)
	if err != nil {
		t.Fatalf("ResolveIP(127.0.0.1) error: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Errorf("got %q, want %q", ip, "127.0.0.1")
	}
}

func TestResolveIP_InvalidDomain(t *testing.T) {
	ip, err := ResolveIP("this-domain-definitely-does-not-exist-hopefully.example.com", 2*time.Second)
	if err == nil {
		t.Logf("expected error, got IP %q (network may have wildcard DNS)", ip)
	}
}

func TestResolveIP_EmptyHost(t *testing.T) {
	ip, err := ResolveIP("", time.Second)
	if err == nil {
		t.Errorf("expected error for empty host, got IP %q", ip)
	}
}
