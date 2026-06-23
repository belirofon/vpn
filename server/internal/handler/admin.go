package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/fetcher"
	"vpn-server/internal/model"
	"vpn-server/internal/parser"
	"vpn-server/internal/singbox"
)

var (
	adminToken   string
	adminTokenMu sync.RWMutex
)

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func getAdminToken() string {
	adminTokenMu.RLock()
	defer adminTokenMu.RUnlock()
	return adminToken
}

func setAdminToken(t string) {
	adminTokenMu.Lock()
	defer adminTokenMu.Unlock()
	adminToken = t
}

func clearAdminToken() {
	adminTokenMu.Lock()
	defer adminTokenMu.Unlock()
	adminToken = ""
}

func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := getAdminToken()
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
			return
		}
		auth := c.GetHeader("Authorization")
		if auth != "Bearer "+token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		c.Next()
	}
}

// AdminLogin authenticates and returns a bearer token.
// @Summary      Admin login
// @Description  Authenticate with email and password to obtain a bearer token for subsequent admin requests
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        credentials body object true "Login credentials" {"email":"admin@example.com","password":"secret"}
// @Success      200 {object} map[string]string "Token"
// @Failure      400 {object} map[string]string "Invalid request"
// @Failure      401 {object} map[string]string "Invalid credentials"
// @Router       /api/admin/login [post]
func AdminLogin(c *gin.Context, cc *cache.ConfigCache) {
	cfg := cc.Config()
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if req.Email != cfg.AdminEmail || req.Password != cfg.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}
	token := generateToken()
	setAdminToken(token)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// AdminHealth returns server health with uptime and configuration details.
// @Summary      Admin server health
// @Description  Returns server health status with uptime and configuration details
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]any
// @Router       /api/admin/health [get]
func AdminHealth(c *gin.Context, cc *cache.ConfigCache) {
	currentCfg := cc.Config()
	status, msg := cc.Status()
	configs := cc.GetConfigs()
	uptime := time.Since(cc.GetStartedAt()).Truncate(time.Second).String()

	c.JSON(http.StatusOK, gin.H{
		"status":           status,
		"message":          msg,
		"configs_tested":   len(configs),
		"uptime":           uptime,
		"subscription_url": currentCfg.SubscriptionURL,
		"refresh_interval": currentCfg.RefreshInterval.String(),
	})
}

// AdminEndpoints lists all registered API routes.
// @Summary      List all endpoints
// @Description  Lists all registered API routes with methods and paths
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]any "Endpoints list"
// @Router       /api/admin/endpoints [get]
func AdminEndpoints(c *gin.Context, r *gin.Engine) {
	routes := r.Routes()
	type endpointInfo struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	endpoints := make([]endpointInfo, 0, len(routes))
	for _, route := range routes {
		endpoints = append(endpoints, endpointInfo{
			Method: route.Method,
			Path:   route.Path,
		})
	}
	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints, "total": len(endpoints)})
}

// AdminGetConfig returns the current runtime configuration values.
// @Summary      Get runtime config
// @Description  Returns the current runtime configuration values
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]any
// @Router       /api/admin/config [get]
func AdminGetConfig(c *gin.Context, cc *cache.ConfigCache) {
	currentCfg := cc.Config()
	c.JSON(http.StatusOK, gin.H{
		"subscription_url": currentCfg.SubscriptionURL,
		"refresh_interval": currentCfg.RefreshInterval.String(),
		"ping_timeout":     currentCfg.PingTimeout.String(),
		"mock_configs":     currentCfg.MockConfigs,
		"skip_verify_tls":  currentCfg.SkipVerifyTLS,
		"cors_origins":     currentCfg.CORSOrigins,
	})
}

// AdminUpdateConfig updates runtime configuration fields.
// @Summary      Update runtime config
// @Description  Update runtime configuration fields (subscription_url, refresh_interval)
// @Tags         Admin
// @Security     bearerAuth
// @Accept       json
// @Produce      json
// @Param        config body object true "Config fields to update"
// @Success      200 {object} map[string]any "Updated"
// @Failure      400 {object} map[string]string "Invalid request"
// @Router       /api/admin/config [put]
func AdminUpdateConfig(c *gin.Context, cc *cache.ConfigCache) {
	var req struct {
		SubscriptionURL *string `json:"subscription_url"`
		RefreshInterval *string `json:"refresh_interval"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	updates := []string{}
	if req.SubscriptionURL != nil {
		cc.SetSubscriptionURL(*req.SubscriptionURL)
		updates = append(updates, "SUBSCRIPTION_URL")
	}
	if req.RefreshInterval != nil {
		d, err := time.ParseDuration(*req.RefreshInterval)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_refresh_interval"})
			return
		}
		cc.SetRefreshInterval(d)
		updates = append(updates, "REFRESH_INTERVAL")
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "updated",
		"updated": updates,
	})
}

// AdminRefreshConfigs triggers an async refresh of the config cache.
// @Summary      Refresh configs (admin)
// @Description  Triggers an asynchronous refresh of the config cache
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]string "Refreshing"
// @Failure      409 {object} model.StatusResponse "Already in progress"
// @Router       /api/admin/refresh-configs [post]
func AdminRefreshConfigs(c *gin.Context, cc *cache.ConfigCache) {
	status, msg := cc.Status()
	if status == model.StatusReady || status == model.StatusError {
		go cc.Refresh()
		c.JSON(http.StatusOK, gin.H{"status": "refreshing"})
	} else {
		c.JSON(http.StatusConflict, model.StatusResponse{
			Status:  status,
			Message: msg,
		})
	}
}

// AdminLogout invalidates the current admin session token.
// @Summary      Admin logout
// @Description  Invalidates the current admin session token
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]string "Logged out"
// @Router       /api/admin/logout [post]
func AdminLogout(c *gin.Context) {
	clearAdminToken()
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

// AdminGetWarp returns current WARP config status.
// @Summary      Get WARP status
// @Description  Returns current WARP config status (available + config details)
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]any "WARP status"
// @Router       /api/admin/warp [get]
func AdminGetWarp(c *gin.Context, cc *cache.ConfigCache) {
	wc := cc.GetWarpConfig()
	if wc == nil {
		c.JSON(http.StatusOK, gin.H{
			"available": false,
			"config":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"available": true,
		"config":    wc,
	})
}

// AdminGenerateWarp force re-generates the Cloudflare WARP config.
// @Summary      Generate WARP config
// @Description  Force re-generates the Cloudflare WARP WireGuard config
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]any "Generated"
// @Failure      500 {object} map[string]string "Generation failed"
// @Router       /api/admin/warp/generate [post]
func AdminGenerateWarp(c *gin.Context, cc *cache.ConfigCache) {
	wc := cc.ForceGenerateWarp()
	if wc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "warp_generation_failed",
			"message": "WARP config generation failed",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "generated",
		"config": wc,
	})
}

// AdminPostBestConfig receives a scanned config and adds it to the best configs list.
func AdminPostBestConfig(c *gin.Context, cc *cache.ConfigCache) {
	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Server   string `json:"server"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		UUID     string `json:"uuid,omitempty"`
		Password string `json:"password,omitempty"`
		TLS      string `json:"tls,omitempty"`
		Network  string `json:"network,omitempty"`
		RawLink  string `json:"raw_link,omitempty"`
		Country  string `json:"country,omitempty"`
		Host     string `json:"host,omitempty"`
		Path     string `json:"path,omitempty"`
		SNI      string `json:"sni,omitempty"`
		FP       string `json:"fp,omitempty"`
		ALPN     string `json:"alpn,omitempty"`
		Pbk      string `json:"pbk,omitempty"`
		Sid      string `json:"sid,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	cfg := model.VpnConfig{
		ID:       req.ID,
		Name:     req.Name,
		Server:   req.Server,
		Port:     req.Port,
		Protocol: req.Protocol,
		UUID:     req.UUID,
		Password: req.Password,
		TLS:      req.TLS,
		Network:  req.Network,
		RawLink:  req.RawLink,
		Country:  req.Country,
		Host:     req.Host,
		Path:     req.Path,
		SNI:      req.SNI,
		FP:       req.FP,
		ALPN:     req.ALPN,
		Pbk:      req.Pbk,
		Sid:      req.Sid,
	}
	// If raw_link is set, parse it to fill missing fields and generate singboxConfig
	if cfg.RawLink != "" {
		if parsed := parser.ParseConfigLink(cfg.RawLink); parsed != nil {
			cfg = *parsed
			if req.ID != "" {
				cfg.ID = req.ID
			}
			if req.Name != "" {
				cfg.Name = req.Name
			}
			if req.Country != "" {
				cfg.Country = req.Country
			}
		}
	}
	// If no singboxConfig yet but we have enough fields, generate it directly
	if cfg.SingboxConfig == nil && cfg.Server != "" && cfg.Port > 0 {
		if sc := singbox.GenerateOutbound(&cfg); sc != nil {
			cfg.SingboxConfig = sc
		}
	}
	if cfg.ID == "" {
		cfg.ID = cfg.Server + ":" + itoa(cfg.Port)
	}
	if cfg.Name == "" {
		cfg.Name = cfg.Server
	}

	cc.AddBestConfig(cfg)
	c.JSON(http.StatusOK, gin.H{"status": "added", "config": cfg})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// AdminListBestConfigs returns all best configs.
func AdminListBestConfigs(c *gin.Context, cc *cache.ConfigCache) {
	configs := cc.GetBestConfigs()
	c.JSON(http.StatusOK, model.BestConfigListResponse{
		Configs: configs,
		Total:   len(configs),
	})
}

// AdminUpdateBestConfig updates a single best config by ID.
func AdminUpdateBestConfig(c *gin.Context, cc *cache.ConfigCache) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_id"})
		return
	}

	var req struct {
		Name     string `json:"name,omitempty"`
		Server   string `json:"server,omitempty"`
		Port     int    `json:"port,omitempty"`
		Protocol string `json:"protocol,omitempty"`
		UUID     string `json:"uuid,omitempty"`
		Password string `json:"password,omitempty"`
		TLS      string `json:"tls,omitempty"`
		Network  string `json:"network,omitempty"`
		RawLink  string `json:"raw_link,omitempty"`
		Country  string `json:"country,omitempty"`
		Host     string `json:"host,omitempty"`
		Path     string `json:"path,omitempty"`
		SNI      string `json:"sni,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	updated := model.VpnConfig{
		Name:     req.Name,
		Server:   req.Server,
		Port:     req.Port,
		Protocol: req.Protocol,
		UUID:     req.UUID,
		Password: req.Password,
		TLS:      req.TLS,
		Network:  req.Network,
		RawLink:  req.RawLink,
		Country:  req.Country,
		Host:     req.Host,
		Path:     req.Path,
		SNI:      req.SNI,
	}

	// If raw_link is provided, parse it to fill fields and generate singboxConfig
	if updated.RawLink != "" {
		if parsed := parser.ParseConfigLink(updated.RawLink); parsed != nil {
			// Preserve overrides from request
			overrides := updated
			updated = *parsed
			if overrides.Name != "" {
				updated.Name = overrides.Name
			}
			if overrides.Country != "" {
				updated.Country = overrides.Country
			}
		}
	}
	// Generate singboxConfig if missing but fields are available
	if updated.SingboxConfig == nil && updated.Server != "" && updated.Port > 0 {
		if sc := singbox.GenerateOutbound(&updated); sc != nil {
			updated.SingboxConfig = sc
		}
	}

	if result := cc.UpdateBestConfig(id, updated); result != nil {
		c.JSON(http.StatusOK, gin.H{"status": "updated", "config": result})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "config_not_found", "message": "No best config found with the given ID"})
	}
}

// AdminDeleteBestConfigByID deletes a single best config by ID.
func AdminDeleteBestConfigByID(c *gin.Context, cc *cache.ConfigCache) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_id"})
		return
	}

	if cc.DeleteBestConfig(id) {
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "config_not_found", "message": "No best config found with the given ID"})
	}
}

// AdminImportBestConfigs imports configs from a URL, raw_links list, or configs array.
func AdminImportBestConfigs(c *gin.Context, cc *cache.ConfigCache) {
	var req struct {
		URL      string              `json:"url,omitempty"`
		RawLinks []string            `json:"raw_links,omitempty"`
		Configs  []model.VpnConfig   `json:"configs,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var added int

	// Case 1: URL — fetch and parse like subscription
	if req.URL != "" {
		slog.Info("importing best configs from URL", "url", req.URL)
		data, err := fetcher.FetchSubscription(context.Background(), req.URL, 30*time.Second)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "fetch_failed",
				"message": "Failed to fetch URL: " + err.Error(),
			})
			return
		}

		links := parser.ParseSubscription(data)
		for _, link := range links {
			if parsed := parser.ParseConfigLink(link); parsed != nil {
				cc.AddBestConfig(*parsed)
				added++
			}
		}
	}

	// Case 2: raw_links list
	for _, link := range req.RawLinks {
		if parsed := parser.ParseConfigLink(link); parsed != nil {
			cc.AddBestConfig(*parsed)
			added++
		}
	}

	// Case 3: full config objects
	for _, cfg := range req.Configs {
		// Generate singboxConfig if missing
		if cfg.SingboxConfig == nil && cfg.Server != "" && cfg.Port > 0 {
			if sc := singbox.GenerateOutbound(&cfg); sc != nil {
				cfg.SingboxConfig = sc
			}
		}
		if cfg.ID == "" {
			cfg.ID = cfg.Server + ":" + itoa(cfg.Port)
		}
		if cfg.Name == "" {
			cfg.Name = cfg.Server
		}
		cc.AddBestConfig(cfg)
		added++
	}

	if added == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "no_configs_added", "added": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "imported", "added": added})
}

// AdminDeleteBestConfigs clears all best configs.
func AdminDeleteBestConfigs(c *gin.Context, cc *cache.ConfigCache) {
	cc.ClearBestConfigs()
	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}

// AdminDeleteWarp clears the cached Cloudflare WARP config.
// @Summary      Delete WARP config
// @Description  Clears the cached Cloudflare WARP WireGuard config
// @Tags         Admin
// @Security     bearerAuth
// @Success      200 {object} map[string]string "Cleared"
// @Router       /api/admin/warp [delete]
func AdminDeleteWarp(c *gin.Context, cc *cache.ConfigCache) {
	cc.ClearWarpConfig()
	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}

func SetupAdminRoutes(r *gin.Engine, cc *cache.ConfigCache) {
	admin := r.Group("/api/admin")

	admin.POST("/login", func(c *gin.Context) {
		AdminLogin(c, cc)
	})

	admin.Use(adminAuthMiddleware())
	{
		admin.GET("/health", func(c *gin.Context) {
			AdminHealth(c, cc)
		})

		admin.GET("/endpoints", func(c *gin.Context) {
			AdminEndpoints(c, r)
		})

		admin.GET("/config", func(c *gin.Context) {
			AdminGetConfig(c, cc)
		})

		admin.PUT("/config", func(c *gin.Context) {
			AdminUpdateConfig(c, cc)
		})

		admin.POST("/refresh-configs", func(c *gin.Context) {
			AdminRefreshConfigs(c, cc)
		})

		admin.POST("/logout", func(c *gin.Context) {
			AdminLogout(c)
		})

		admin.GET("/warp", func(c *gin.Context) {
			AdminGetWarp(c, cc)
		})

		admin.POST("/warp/generate", func(c *gin.Context) {
			AdminGenerateWarp(c, cc)
		})

		admin.DELETE("/warp", func(c *gin.Context) {
			AdminDeleteWarp(c, cc)
		})

		admin.GET("/best-configs", func(c *gin.Context) {
			AdminListBestConfigs(c, cc)
		})

		admin.POST("/best-configs", func(c *gin.Context) {
			AdminPostBestConfig(c, cc)
		})

		admin.PUT("/best-configs/:id", func(c *gin.Context) {
			AdminUpdateBestConfig(c, cc)
		})

		admin.DELETE("/best-configs/:id", func(c *gin.Context) {
			AdminDeleteBestConfigByID(c, cc)
		})

		admin.DELETE("/best-configs", func(c *gin.Context) {
			AdminDeleteBestConfigs(c, cc)
		})

		admin.POST("/best-configs/import", func(c *gin.Context) {
			AdminImportBestConfigs(c, cc)
		})
	}
}
