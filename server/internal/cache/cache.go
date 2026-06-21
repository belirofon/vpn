// Package cache provides a thread-safe in-memory store for tested VPN configs.
package cache

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"vpn-server/internal/config"
	"vpn-server/internal/geo"
	"vpn-server/internal/model"
	"vpn-server/internal/pipeline"
)

// ConfigCache is a thread-safe cache of tested and filtered VPN configs.
type ConfigCache struct {
	mu       sync.RWMutex
	status   model.ServerStatus
	statusMsg string
	configs  []model.VpnConfig
	updated  time.Time
	pl       *pipeline.Pipeline
	ticker   *time.Ticker
	stopCh   chan struct{}
	logger   *slog.Logger
}

// NewCache creates a new ConfigCache.
func NewCache(cfg config.Config, g *geo.GeoDB, logger *slog.Logger) *ConfigCache {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConfigCache{
		stopCh: make(chan struct{}),
		status: model.StatusLoading,
		pl:     pipeline.New(cfg, g, logger),
		logger: logger,
	}
}

func (cc *ConfigCache) setStatus(s model.ServerStatus, msg string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.status = s
	cc.statusMsg = msg
}

// Status returns the current server status.
func (cc *ConfigCache) Status() (model.ServerStatus, string) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.status, cc.statusMsg
}

// GetConfigs returns a copy of all cached configs.
func (cc *ConfigCache) GetConfigs() []model.VpnConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := make([]model.VpnConfig, len(cc.configs))
	copy(result, cc.configs)
	return result
}

// GetBestConfig returns the lowest-latency config, or nil if empty.
func (cc *ConfigCache) GetBestConfig() *model.VpnConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if len(cc.configs) == 0 {
		return nil
	}
	best := cc.configs[0]
	return &best
}

// GetUpdated returns the last update timestamp as RFC3339.
func (cc *ConfigCache) GetUpdated() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.updated.Format(time.RFC3339)
}

// Init performs the initial config refresh synchronously.
func (cc *ConfigCache) Init() {
	cc.refresh()
}

// Start begins periodic config refreshes on a ticker.
func (cc *ConfigCache) Start() {
	if cc.ticker != nil {
		return
	}
	if cc.pl == nil || cc.pl.Cfg().RefreshInterval <= 0 {
		return
	}
	cc.ticker = time.NewTicker(cc.pl.Cfg().RefreshInterval)
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

// Stop stops the periodic refresh ticker.
func (cc *ConfigCache) Stop() {
	if cc.tickerCancel != nil {
		cc.tickerCancel()
		cc.tickerCancel = nil
	}
}

func (cc *ConfigCache) GetStartedAt() time.Time {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.startedAt
}

func (cc *ConfigCache) SetSubscriptionURL(url string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cfg.SubscriptionURL = url
}

func (cc *ConfigCache) SetRefreshInterval(d time.Duration) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.tickerCancel != nil {
		cc.tickerCancel()
		cc.tickerCancel = nil
	}

	if d <= 0 {
		d = 30 * time.Minute
		log.Printf("WARN: invalid REFRESH_INTERVAL=%v, falling back to %v", cc.cfg.RefreshInterval, d)
	}

	cc.cfg.RefreshInterval = d

	ctx, cancel := context.WithCancel(context.Background())
	cc.tickerCancel = cancel

	ticker := time.NewTicker(d)
	go func() {
		for {
			select {
			case <-ticker.C:
				cc.refresh()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	log.Printf("INFO: auto-refresh ticker restarted at %v interval", d)
}

// Refresh triggers an async refresh.
func (cc *ConfigCache) Refresh() {
	go cc.refresh()
}

func (cc *ConfigCache) refresh() {
	cc.logger.Info("starting config refresh")
	cc.setStatus(model.StatusLoading, "Refreshing configs...")

	result, err := cc.pl.Run(context.Background())
	if err != nil {
		cc.logger.Error("refresh failed", "error", err)
		cc.setStatus(model.StatusError, err.Error())
		return
	}

	cc.mu.Lock()
	cc.configs = result
	cc.updated = time.Now()
	cc.status = model.StatusReady
	cc.statusMsg = ""
	cc.mu.Unlock()

	cc.logger.Info("refresh complete",
		"configs", len(result),
		"fastest", result[0].Name,
		"latency_ms", result[0].LatencyMs,
	)
}
