// Package pipeline orchestrates the config refresh pipeline:
// fetch → parse → test → geo-filter → reality-filter → sort.
//
// It separates the config processing logic from the cache storage,
// giving each step of the pipeline its own responsibility.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"vpn-server/internal/config"
	"vpn-server/internal/fetcher"
	"vpn-server/internal/geo"
	"vpn-server/internal/model"
	"vpn-server/internal/parser"
	"vpn-server/internal/tester"
	"vpn-server/internal/warp"
)

const fetchTimeout = 30 * time.Second

// Pipeline orchestrates the full config refresh lifecycle.
type Pipeline struct {
	cfg    *config.Config
	geoDB  *geo.DB
	logger *slog.Logger
}

// New creates a new Pipeline. If logger is nil, slog.Default() is used.
func New(cfg *config.Config, geoDB *geo.DB, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		cfg:    cfg,
		geoDB:  geoDB,
		logger: logger,
	}
}

// Run executes the full config refresh pipeline.
// Returns the sorted, filtered configs ready for caching.
func (p *Pipeline) Run(ctx context.Context) ([]model.VpnConfig, error) {
	if p.cfg.MockConfigs {
		p.logger.InfoContext(ctx, "MOCK_CONFIGS=true, loading mock configs")
		return loadMockConfigs(), nil
	}

	if p.cfg.SubscriptionURL == "" {
		return nil, fmt.Errorf("subscription URL not configured")
	}

	p.logger.InfoContext(ctx, "fetching subscription")
	raw, err := fetcher.FetchSubscription(ctx, p.cfg.SubscriptionURL, fetchTimeout)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	links := parser.ParseSubscription(raw)
	if len(links) == 0 {
		return nil, fmt.Errorf("no configs found in subscription")
	}
	p.logger.InfoContext(ctx, "parsed links from subscription", "count", len(links))

	var parsed []*model.VpnConfig
	for _, link := range links {
		if cfg := parser.ParseConfigLink(link); cfg != nil {
			parsed = append(parsed, cfg)
		}
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("no valid configs could be parsed")
	}
	p.logger.InfoContext(ctx, "parsed valid configs", "count", len(parsed))

	p.logger.InfoContext(ctx, "testing configs")
	tested := tester.TestConfigs(ctx, parsed, p.cfg.PingTimeout, p.cfg.SkipVerifyTLS)
	if len(tested) == 0 {
		return nil, fmt.Errorf("no configs passed connectivity test")
	}
	p.logger.InfoContext(ctx, "connectivity test results",
		"passed", len(tested), "total", len(parsed),
	)

	filtered := geo.FilterNonRussia(tested, p.geoDB)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no configs passed GeoIP filter")
	}
	p.logger.InfoContext(ctx, "configs after GeoIP filter",
		"count", len(filtered), "total", len(tested),
	)

	filtered = filterReality(filtered, p.logger)

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].LatencyMs < filtered[j].LatencyMs
	})

	result := make([]model.VpnConfig, len(filtered))
	for i, c := range filtered {
		result[i] = *c
	}

	p.logger.InfoContext(ctx, "pipeline complete",
		"configs", len(result),
		"fastest", result[0].Name,
		"latency_ms", result[0].LatencyMs,
	)
	return result, nil
}

// RunWarp generates a WARP Cloudflare config and tests connectivity.
func (p *Pipeline) RunWarp(ctx context.Context) (*model.WarpConfig, error) {
	p.logger.InfoContext(ctx, "generating WARP config")
	return warp.Generate(ctx, p.cfg.PingTimeout)
}

// filterReality removes REALITY configs when non-REALITY exist.
// Flutter client (flutter_v2ray_client) does not support REALITY yet.
func filterReality(configs []*model.VpnConfig, logger *slog.Logger) []*model.VpnConfig {
	var noReality []*model.VpnConfig
	for _, c := range configs {
		if c.TLS != "reality" {
			noReality = append(noReality, c)
		}
	}
	if len(noReality) > 0 {
		logger.Info("configs after REALITY filter", "count", len(noReality))
		return noReality
	}
	logger.Warn("all configs are REALITY, keeping them as fallback")
	return configs
}

// loadMockConfigs returns hardcoded configs for development/testing.
func loadMockConfigs() []model.VpnConfig {
	mockConfigs := []model.VpnConfig{
		{
			ID: "de-1.example.com:443", Name: "de-1.example.com",
			Server: "192.168.1.10", Port: 443, Protocol: "vless",
			UUID: "mock-uuid-1111-1111", LatencyMs: 45, Country: "DE",
			RawLink: "vless://mock-uuid-1111-1111@de-1.example.com:443?security=tls&type=tcp",
		},
		{
			ID: "nl-1.example.com:443", Name: "nl-1.example.com",
			Server: "192.168.1.20", Port: 443, Protocol: "vless",
			UUID: "mock-uuid-2222-2222", LatencyMs: 62, Country: "NL",
			RawLink: "vless://mock-uuid-2222-2222@nl-1.example.com:443?security=tls&type=tcp",
		},
		{
			ID: "us-1.example.com:443", Name: "us-1.example.com",
			Server: "192.168.1.30", Port: 443, Protocol: "vmess",
			UUID: "mock-uuid-3333-3333", LatencyMs: 120, Country: "US",
			RawLink: "vmess://eyJhZGQiOiJ1cy0xLmV4YW1wbGUuY29tIiwicG9ydCI6IjQ0MyIsImlkIjoibW9jay11dWlkLTMzMzMtMzMzMyJ9",
		},
		{
			ID: "ru-1.example.com:443", Name: "ru-1.example.com",
			Server: "192.168.1.40", Port: 443, Protocol: "vless",
			UUID: "mock-uuid-4444-4444", LatencyMs: 5, Country: "RU",
			RawLink: "vless://mock-uuid-4444-4444@ru-1.example.com:443?security=tls&type=tcp",
		},
	}

	// Filter out RU and sort.
	var filtered []model.VpnConfig
	for _, cfg := range mockConfigs {
		if cfg.Country == "RU" {
			continue
		}
		filtered = append(filtered, cfg)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].LatencyMs < filtered[j].LatencyMs
	})
	return filtered
}
