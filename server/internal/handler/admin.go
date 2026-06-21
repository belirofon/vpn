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

func SetupAdminRoutes(r *gin.Engine, cc *cache.ConfigCache) {
	cfg := cc.Config()

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
			currentCfg := cc.Config()
			c.JSON(http.StatusOK, gin.H{
				"subscription_url": currentCfg.SubscriptionURL,
				"refresh_interval": currentCfg.RefreshInterval.String(),
				"ping_timeout":     currentCfg.PingTimeout.String(),
				"mock_configs":     currentCfg.MockConfigs,
				"skip_verify_tls":  currentCfg.SkipVerifyTLS,
				"cors_origins":     currentCfg.CORSOrigins,
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
