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
	mu         sync.RWMutex
	cfg        config.Config
	status     model.ServerStatus
	statusMsg  string
	configs    []model.VpnConfig
	warpConfig *model.WarpConfig
	updated    time.Time
	startedAt  time.Time
	pl         *pipeline.Pipeline
	ticker     *time.Ticker
	stopCh     chan struct{}
	logger     *slog.Logger
}

// NewCache creates a new ConfigCache.
func NewCache(cfg config.Config, g *geo.DB, logger *slog.Logger) *ConfigCache {
	if logger == nil {
		logger = slog.Default()
	}
	cc := &ConfigCache{
		cfg:       cfg,
		stopCh:    make(chan struct{}),
		status:    model.StatusLoading,
		startedAt: time.Now(),
		logger:    logger,
	}
	cc.pl = pipeline.New(&cc.cfg, g, logger)
	return cc
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

// GetWarpConfig returns the cached WARP config, or nil if not available.
func (cc *ConfigCache) GetWarpConfig() *model.WarpConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if cc.warpConfig == nil {
		return nil
	}
	wc := *cc.warpConfig
	return &wc
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
	if cc.pl == nil || cc.cfg.RefreshInterval <= 0 {
		return
	}
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

// Stop stops the periodic refresh ticker.
func (cc *ConfigCache) Stop() {
	close(cc.stopCh)
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
		cc.logger.Error("pipeline failed", "error", err)
		cc.setStatus(model.StatusError, err.Error())
	} else {
		cc.mu.Lock()
		cc.configs = result
		cc.mu.Unlock()

		cc.logger.Info("pipeline complete",
			"configs", len(result),
			"fastest", result[0].Name,
			"latency_ms", result[0].LatencyMs,
		)
	}

	if cc.cfg.WarpEnabled {
		cc.generateWarpConfig()
	}

	cc.mu.Lock()
	cc.updated = time.Now()
	if cc.status != model.StatusError {
		cc.status = model.StatusReady
		cc.statusMsg = ""
	}
	cc.mu.Unlock()
}

func (cc *ConfigCache) generateWarpConfig() {
	cc.logger.Info("generating WARP config")
	wc, err := cc.pl.RunWarp(context.Background())
	if err != nil {
		cc.logger.Error("WARP generation failed", "error", err)
		if wc == nil {
			return
		}
	}

	cc.mu.Lock()
	cc.warpConfig = wc
	cc.mu.Unlock()

	cc.logger.Info("WARP config ready",
		"latency_ms", wc.LatencyMs,
		"endpoint", wc.Endpoint,
	)
}

// GetStartedAt returns the time the cache was initialized.
func (cc *ConfigCache) GetStartedAt() time.Time {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.startedAt
}

// Config returns a copy of the current server configuration.
func (cc *ConfigCache) Config() config.Config {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.cfg
}

// SetSubscriptionURL updates the subscription URL at runtime.
func (cc *ConfigCache) SetSubscriptionURL(url string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cfg.SubscriptionURL = url
	cc.logger.Info("subscription URL updated", "url", url)
}

// SetRefreshInterval updates the refresh interval at runtime.
func (cc *ConfigCache) SetRefreshInterval(d time.Duration) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cfg.RefreshInterval = d
	cc.logger.Info("refresh interval updated", "interval", d)
	if cc.ticker != nil {
		cc.ticker.Reset(d)
	}
}
