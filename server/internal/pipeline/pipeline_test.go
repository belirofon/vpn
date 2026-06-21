package pipeline

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"vpn-server/internal/config"
	"vpn-server/internal/model"
)

func TestLoadMockConfigs_ReturnsNonRU(t *testing.T) {
	result := loadMockConfigs()

	if len(result) == 0 {
		t.Fatal("expected mock configs, got empty")
	}

	for _, cfg := range result {
		if cfg.Country == "RU" {
			t.Errorf("expected no RU configs, got %+v", cfg)
		}
	}
}

func TestLoadMockConfigs_SortedByLatency(t *testing.T) {
	result := loadMockConfigs()

	for i := 1; i < len(result); i++ {
		if result[i].LatencyMs < result[i-1].LatencyMs {
			t.Errorf("configs not sorted by latency: %dms > %dms at index %d",
				result[i-1].LatencyMs, result[i].LatencyMs, i)
		}
	}
}

func TestLoadMockConfigs_FastestIsDE(t *testing.T) {
	result := loadMockConfigs()

	if len(result) == 0 || result[0].Name != "de-1.example.com" {
		t.Errorf("expected fastest to be DE (45ms), got %+v", result[0])
	}
}

func TestRun_MockConfigs(t *testing.T) {
	cfg := config.Config{
		MockConfigs: true,
	}
	p := New(&cfg, nil, nil)
	result, err := p.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() with MockConfigs=true: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestRun_NoSubscriptionURL(t *testing.T) {
	cfg := config.Config{
		MockConfigs: false,
	}
	p := New(&cfg, nil, nil)
	_, err := p.Run(context.Background())

	if err == nil {
		t.Fatal("expected error when no subscription URL and no mock configs")
	}
}

func TestFilterReality_RemovesReality(t *testing.T) {
	configs := []*model.VpnConfig{
		{TLS: "tls", Server: "a.com", Port: 443, LatencyMs: 10},
		{TLS: "reality", Server: "b.com", Port: 443, LatencyMs: 20},
		{TLS: "none", Server: "c.com", Port: 443, LatencyMs: 30},
	}

	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	result := filterReality(configs, discard)

	if len(result) != 2 {
		t.Fatalf("expected 2 non-reality configs, got %d", len(result))
	}
	for _, c := range result {
		if c.TLS == "reality" {
			t.Errorf("expected no reality configs, got %+v", c)
		}
	}
}

func TestFilterReality_AllReality_KeepsFallback(t *testing.T) {
	configs := []*model.VpnConfig{
		{TLS: "reality", Server: "a.com", Port: 443, LatencyMs: 10},
		{TLS: "reality", Server: "b.com", Port: 443, LatencyMs: 20},
	}

	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	result := filterReality(configs, discard)

	if len(result) != 2 {
		t.Fatalf("expected all configs kept as fallback, got %d", len(result))
	}
}
