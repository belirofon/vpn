package cache

import (
	"log"
	"sort"
	"sync"
	"time"

	"vpn-server/internal/config"
	"vpn-server/internal/fetcher"
	"vpn-server/internal/geo"
	"vpn-server/internal/model"
	"vpn-server/internal/parser"
	"vpn-server/internal/tester"
)

type ConfigCache struct {
	mu         sync.RWMutex
	status     model.ServerStatus
	statusMsg  string
	configs    []model.VpnConfig
	updated    time.Time
	cfg        config.Config
	geo        *geo.GeoDB
	ticker     *time.Ticker
	stopCh     chan struct{}
}

func NewCache(cfg config.Config, g *geo.GeoDB) *ConfigCache {
	return &ConfigCache{
		cfg:    cfg,
		geo:    g,
		stopCh: make(chan struct{}),
		status: model.StatusLoading,
	}
}

func (cc *ConfigCache) setStatus(s model.ServerStatus, msg string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.status = s
	cc.statusMsg = msg
}

func (cc *ConfigCache) Status() (model.ServerStatus, string) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.status, cc.statusMsg
}

func (cc *ConfigCache) GetConfigs() []model.VpnConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := make([]model.VpnConfig, len(cc.configs))
	copy(result, cc.configs)
	return result
}

func (cc *ConfigCache) GetBestConfig() *model.VpnConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if len(cc.configs) == 0 {
		return nil
	}
	best := cc.configs[0]
	return &best
}

func (cc *ConfigCache) GetUpdated() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.updated.Format(time.RFC3339)
}

func (cc *ConfigCache) Init() {
	cc.refresh()
}

func (cc *ConfigCache) Start() {
	if cc.cfg.RefreshInterval > 0 {
		cc.ticker = time.NewTicker(cc.cfg.RefreshInterval)
		go func() {
			for {
				select {
				case <-cc.ticker.C:
					cc.refresh()
				case <-cc.stopCh:
					cc.ticker.Stop()
					return
				}
			}
		}()
	}
}

func (cc *ConfigCache) Stop() {
	close(cc.stopCh)
}

func (cc *ConfigCache) Refresh() {
	go cc.refresh()
}

func (cc *ConfigCache) refresh() {
	log.Println("INFO: refreshing configs...")

	if cc.cfg.MockConfigs {
		log.Println("INFO: MOCK_CONFIGS=true, loading mock configs")
		cc.loadMockConfigs()
		cc.setStatus(model.StatusReady, "Mock configs loaded")
		return
	}

	if cc.cfg.SubscriptionURL == "" {
		log.Println("WARN: SUBSCRIPTION_URL not set, skipping refresh")
		cc.setStatus(model.StatusError, "SUBSCRIPTION_URL not configured")
		return
	}

	cc.setStatus(model.StatusLoading, "Fetching subscription...")

	raw, err := fetcher.FetchSubscription(cc.cfg.SubscriptionURL, 30*time.Second)
	if err != nil {
		log.Printf("ERROR: fetch failed: %v", err)
		cc.setStatus(model.StatusError, "Failed to fetch subscription: "+err.Error())
		return
	}

	links := parser.ParseSubscription(raw)
	if len(links) == 0 {
		log.Println("WARN: no configs found in subscription")
		cc.setStatus(model.StatusError, "No configs found in subscription")
		return
	}

	log.Printf("INFO: parsed %d links from subscription", len(links))

	var parsed []*model.VpnConfig
	for _, link := range links {
		if cfg := parser.ParseConfigLink(link); cfg != nil {
			parsed = append(parsed, cfg)
		}
	}

	log.Printf("INFO: parsed %d valid configs", len(parsed))

	if len(parsed) == 0 {
		log.Println("WARN: no valid configs could be parsed")
		cc.setStatus(model.StatusError, "No valid configs could be parsed")
		return
	}

	cc.setStatus(model.StatusTesting, "Testing configs...")

	tested := tester.TestConfigs(parsed, cc.cfg.PingTimeout)
	log.Printf("INFO: %d/%d configs passed connectivity test", len(tested), len(parsed))

	if len(tested) == 0 {
		log.Println("WARN: no configs passed connectivity test")
		cc.setStatus(model.StatusError, "No configs passed connectivity test")
		return
	}

	filtered := geo.FilterNonRussia(tested, cc.geo)
	log.Printf("INFO: %d/%d configs after GeoIP filter", len(filtered), len(tested))

	if len(filtered) == 0 {
		log.Println("WARN: no configs passed GeoIP filter")
		cc.setStatus(model.StatusError, "No configs passed GeoIP filter")
		return
	}

	// Filter out REALITY — Flutter client can't handle them yet
	var noReality []*model.VpnConfig
	for _, c := range filtered {
		if c.TLS != "reality" {
			noReality = append(noReality, c)
		}
	}
	if len(noReality) > 0 {
		filtered = noReality
		log.Printf("INFO: %d configs after REALITY filter", len(filtered))
	} else {
		log.Println("WARN: all configs are REALITY, keeping them as fallback")
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].LatencyMs < filtered[j].LatencyMs
	})

	result := make([]model.VpnConfig, len(filtered))
	for i, c := range filtered {
		result[i] = *c
	}

	cc.mu.Lock()
	cc.configs = result
	cc.updated = time.Now()
	cc.status = model.StatusReady
	cc.statusMsg = ""
	cc.mu.Unlock()

	log.Printf("INFO: cache ready with %d configs (fastest: %s, %dms)",
		len(result), result[0].Name, result[0].LatencyMs)
}

func (cc *ConfigCache) loadMockConfigs() {
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

	cc.mu.Lock()
	cc.configs = filtered
	cc.updated = time.Now()
	cc.mu.Unlock()

	log.Printf("INFO: loaded %d mock configs after filter (fastest: %s, %dms)",
		len(filtered), filtered[0].Name, filtered[0].LatencyMs)
}
