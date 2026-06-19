package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/config"
	"vpn-server/internal/model"
)

var (
	adminToken   string
	adminTokenMu sync.RWMutex
)

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
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

func SetupAdminRoutes(r *gin.Engine, cfg *config.Config, cc *cache.ConfigCache) {
	admin := r.Group("/api/admin")

	admin.POST("/login", func(c *gin.Context) {
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
	})

	admin.Use(adminAuthMiddleware())
	{
		admin.GET("/health", func(c *gin.Context) {
			status, msg := cc.Status()
			configs := cc.GetConfigs()
			uptime := time.Since(cc.GetStartedAt()).Truncate(time.Second).String()

			c.JSON(http.StatusOK, gin.H{
				"status":         status,
				"message":        msg,
				"configs_tested": len(configs),
				"uptime":         uptime,
				"subscription_url": cfg.SubscriptionURL,
				"refresh_interval": cfg.RefreshInterval.String(),
			})
		})

		admin.GET("/endpoints", func(c *gin.Context) {
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
		})

		admin.GET("/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"subscription_url": cfg.SubscriptionURL,
				"refresh_interval": cfg.RefreshInterval.String(),
				"ping_timeout":     cfg.PingTimeout.String(),
				"mock_configs":     cfg.MockConfigs,
				"skip_verify_tls":  cfg.SkipVerifyTLS,
				"cors_origins":     cfg.CORSOrigins,
			})
		})

		admin.PUT("/config", func(c *gin.Context) {
			var req struct {
				SubscriptionURL *string `json:"subscription_url"`
				RefreshInterval *string `json:"refresh_interval"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
				return
			}

			updates := []string{}
			needsRefresh := false
			if req.SubscriptionURL != nil {
				cfg.SubscriptionURL = *req.SubscriptionURL
				cc.SetSubscriptionURL(*req.SubscriptionURL)
				updates = append(updates, "SUBSCRIPTION_URL")
				needsRefresh = true
			}
			if req.RefreshInterval != nil {
				d, err := time.ParseDuration(*req.RefreshInterval)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_refresh_interval"})
					return
				}
				cfg.RefreshInterval = d
				cc.SetRefreshInterval(d)
				updates = append(updates, "REFRESH_INTERVAL")
			}

			// Auto-trigger refresh when subscription URL changes
			if needsRefresh {
				go cc.Refresh()
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "updated",
				"updated": updates,
				"refresh": needsRefresh,
			})
		})

		admin.POST("/refresh-configs", func(c *gin.Context) {
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
		})

		admin.POST("/logout", func(c *gin.Context) {
			clearAdminToken()
			c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
		})
	}
}
