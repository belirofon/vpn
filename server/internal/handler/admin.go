package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/model"
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
	}
}
