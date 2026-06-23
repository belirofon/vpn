// Package cache provides a thread-safe in-memory store for tested VPN configs.
package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"vpn-server/internal/config"
	"vpn-server/internal/geo"
	"vpn-server/internal/model"
	"vpn-server/internal/pipeline"
)

// ConfigCache is a thread-safe cache of tested and filtered VPN configs.
type ConfigCache struct {
	mu              sync.RWMutex
	cfg             config.Config
	status          model.ServerStatus
	statusMsg       string
	configs         []model.VpnConfig
	bestConfigs     []model.VpnConfig
	warpConfig      *model.WarpConfig
	updated         time.Time
	startedAt       time.Time
	pl              *pipeline.Pipeline
	ticker          *time.Ticker
	stopCh          chan struct{}
	logger          *slog.Logger
	bestConfigsPath string
}

// NewCache creates a new ConfigCache.
func NewCache(cfg config.Config, g *geo.DB, logger *slog.Logger) *ConfigCache {
	if logger == nil {
		logger = slog.Default()
	}
	cc := &ConfigCache{
		cfg:             cfg,
		stopCh:          make(chan struct{}),
		status:          model.StatusLoading,
		startedAt:       time.Now(),
		logger:          logger,
		bestConfigsPath: cfg.BestConfigsPath,
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

// GetBestConfigs returns a copy of all admin-scanned best configs.
func (cc *ConfigCache) GetBestConfigs() []model.VpnConfig {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := make([]model.VpnConfig, len(cc.bestConfigs))
	copy(result, cc.bestConfigs)
	return result
}

// AddBestConfig appends a config to the admin-scanned best configs list and persists to disk.
func (cc *ConfigCache) AddBestConfig(cfg model.VpnConfig) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.bestConfigs = append(cc.bestConfigs, cfg)
	cc.logger.Info("best config added",
		"name", cfg.Name,
		"server", cfg.Server,
	)
	cc.persistBestConfigs()
}

// ClearBestConfigs removes all admin-scanned best configs.
func (cc *ConfigCache) ClearBestConfigs() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.bestConfigs = nil
	cc.logger.Info("best configs cleared")
	cc.persistBestConfigs()
}

// DeleteBestConfig removes a single best config by its index or ID.
// Returns true if a config was removed.
func (cc *ConfigCache) DeleteBestConfig(id string) bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	for i, c := range cc.bestConfigs {
		if c.ID == id {
			cc.bestConfigs = append(cc.bestConfigs[:i], cc.bestConfigs[i+1:]...)
			cc.logger.Info("best config deleted", "id", id)
			cc.persistBestConfigs()
			return true
		}
	}
	return false
}

// UpdateBestConfig updates a single best config by its ID.
// Returns the updated config or nil if not found.
func (cc *ConfigCache) UpdateBestConfig(id string, updated model.VpnConfig) *model.VpnConfig {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	for i, c := range cc.bestConfigs {
		if c.ID == id {
			updated.ID = id
			cc.bestConfigs[i] = updated
			cc.logger.Info("best config updated", "id", id)
			cc.persistBestConfigs()
			result := cc.bestConfigs[i]
			return &result
		}
	}
	return nil
}

// persistBestConfigs writes best configs to the JSON file if a path is configured.
func (cc *ConfigCache) persistBestConfigs() {
	if cc.bestConfigsPath == "" {
		return
	}
	data, err := json.MarshalIndent(cc.bestConfigs, "", "  ")
	if err != nil {
		cc.logger.Error("failed to marshal best configs", "error", err)
		return
	}
	if err := os.WriteFile(cc.bestConfigsPath, data, 0644); err != nil {
		cc.logger.Error("failed to write best configs", "path", cc.bestConfigsPath, "error", err)
	}
}

// loadBestConfigs reads best configs from the JSON file if a path is configured.
func (cc *ConfigCache) loadBestConfigs() {
	if cc.bestConfigsPath == "" {
		return
	}
	data, err := os.ReadFile(cc.bestConfigsPath)
	if err != nil {
		if os.IsNotExist(err) {
			cc.logger.Info("no best configs file found, starting fresh", "path", cc.bestConfigsPath)
		} else {
			cc.logger.Error("failed to read best configs", "path", cc.bestConfigsPath, "error", err)
		}
		return
	}
	var configs []model.VpnConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		cc.logger.Error("failed to unmarshal best configs", "error", err)
		return
	}
	cc.bestConfigs = configs
	cc.logger.Info("loaded best configs from file", "path", cc.bestConfigsPath, "count", len(configs))
}

// GetUpdated returns the last update timestamp as RFC3339.
func (cc *ConfigCache) GetUpdated() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.updated.Format(time.RFC3339)
}

// Init performs the initial config refresh synchronously and loads persisted best configs.
func (cc *ConfigCache) Init() {
	cc.loadBestConfigs()
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

// ForceGenerateWarp generates and caches a WARP config, returns it (may be offline fallback).
func (cc *ConfigCache) ForceGenerateWarp() *model.WarpConfig {
	cc.logger.Info("forced WARP generation from admin")
	cc.generateWarpConfig()
	return cc.GetWarpConfig()
}

// ClearWarpConfig removes the cached WARP config.
func (cc *ConfigCache) ClearWarpConfig() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.warpConfig = nil
	cc.logger.Info("WARP config cleared")
}
