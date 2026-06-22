package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/model"
)

// GetBestConfigs returns all admin-scanned best configs.
// @Summary      Get best configs
// @Description  Returns all admin-scanned (best) proxy configurations
// @Tags         Public
// @Success      200 {object} model.BestConfigListResponse
// @Router       /api/best-configs [get]
func GetBestConfigs(c *gin.Context, cc *cache.ConfigCache) {
	configs := cc.GetBestConfigs()
	c.JSON(http.StatusOK, model.BestConfigListResponse{
		Configs: configs,
		Total:   len(configs),
	})
}

// Health returns the server health status.
// @Summary      Server health check
// @Description  Returns the current server health status (loading/testing/ready/error)
// @Tags         Public
// @Success      200 {object} model.StatusResponse
// @Router       /health [get]
func Health(c *gin.Context, cc *cache.ConfigCache) {
	status, msg := cc.Status()
	c.JSON(http.StatusOK, model.StatusResponse{
		Status:  status,
		Message: msg,
	})
}

// GetStatus returns the server status with tested config summary.
// @Summary      Server status
// @Description  Returns the server status, number of tested configs, and the best config name/latency
// @Tags         Public
// @Success      200 {object} model.StatusResponse
// @Router       /api/status [get]
func GetStatus(c *gin.Context, cc *cache.ConfigCache) {
	status, msg := cc.Status()
	best := cc.GetBestConfig()
	updated := cc.GetUpdated()

	resp := model.StatusResponse{
		Status:        status,
		Message:       msg,
		ConfigsTested: len(cc.GetConfigs()),
		Updated:       updated,
	}
	if best != nil {
		resp.BestName = best.Name
		resp.BestLatency = best.LatencyMs
	}
	c.JSON(http.StatusOK, resp)
}

// GetBestConfig returns the best performing (lowest latency) non-Russian config.
// @Summary      Get best config
// @Description  Returns the lowest-latency non-Russian proxy configuration. Returns 503 if not ready or no configs available.
// @Tags         Public
// @Success      200 {object} model.BestConfigResponse
// @Failure      503 {object} model.StatusResponse "Server not ready"
// @Failure      503 {object} model.ErrorResponse  "No available configs"
// @Router       /api/best-config [get]
func GetBestConfig(c *gin.Context, cc *cache.ConfigCache) {
	status, msg := cc.Status()
	if status != model.StatusReady {
		c.JSON(http.StatusServiceUnavailable, model.StatusResponse{
			Status:  status,
			Message: msg,
		})
		return
	}

	best := cc.GetBestConfig()
	if best == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Error:   model.ErrNoAvailableConfigs,
			Message: "No available non-Russia configs found",
		})
		return
	}

	c.JSON(http.StatusOK, model.BestConfigResponse{
		Config:  best,
		Updated: cc.GetUpdated(),
	})
}

// GetConfigs returns all tested and geo-filtered configurations sorted by latency.
// @Summary      List all configs
// @Description  Returns all tested and geo-filtered proxy configurations, sorted by latency
// @Tags         Public
// @Success      200 {object} model.ConfigListResponse
// @Failure      503 {object} model.StatusResponse "Server not ready"
// @Router       /api/configs [get]
func GetConfigs(c *gin.Context, cc *cache.ConfigCache) {
	status, msg := cc.Status()
	if status != model.StatusReady {
		c.JSON(http.StatusServiceUnavailable, model.StatusResponse{
			Status:  status,
			Message: msg,
		})
		return
	}

	configs := cc.GetConfigs()
	c.JSON(http.StatusOK, model.ConfigListResponse{
		Configs: configs,
		Updated: cc.GetUpdated(),
		Total:   len(configs),
	})
}

// GetWarpConfig returns the current Cloudflare WARP WireGuard config.
// @Summary      Get WARP config
// @Description  Returns the current Cloudflare WARP WireGuard configuration if generated and WARP is enabled
// @Tags         Public
// @Success      200 {object} model.WarpConfigResponse
// @Failure      404 {object} model.ErrorResponse "WARP not available"
// @Router       /api/warp-config [get]
func GetWarpConfig(c *gin.Context, cc *cache.ConfigCache) {
	wc := cc.GetWarpConfig()
	if wc == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error:   "warp_not_available",
			Message: "WARP config not generated or WARP is disabled",
		})
		return
	}
	c.JSON(http.StatusOK, model.WarpConfigResponse{
		Config:  wc,
		Updated: cc.GetUpdated(),
	})
}

// PostRefresh triggers an immediate refresh of the config cache.
// @Summary      Refresh config cache
// @Description  Triggers an immediate asynchronous refresh of the config cache (fetch → parse → test → geo → reality → sort)
// @Tags         Public
// @Success      200 {object} map[string]string "Refreshing started"
// @Failure      409 {object} model.StatusResponse "Refresh already in progress"
// @Router       /api/refresh [post]
func PostRefresh(c *gin.Context, cc *cache.ConfigCache) {
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

func SetupRoutes(r *gin.Engine, c *cache.ConfigCache) {
	r.GET("/health", func(ctx *gin.Context) {
		Health(ctx, c)
	})

	api := r.Group("/api")
	{
		api.GET("/status", func(ctx *gin.Context) {
			GetStatus(ctx, c)
		})

		api.GET("/best-config", func(ctx *gin.Context) {
			GetBestConfig(ctx, c)
		})

		api.GET("/configs", func(ctx *gin.Context) {
			GetConfigs(ctx, c)
		})

		api.GET("/warp-config", func(ctx *gin.Context) {
			GetWarpConfig(ctx, c)
		})

		api.POST("/refresh", func(ctx *gin.Context) {
			PostRefresh(ctx, c)
		})

		api.GET("/best-configs", func(ctx *gin.Context) {
			GetBestConfigs(ctx, c)
		})
	}

	SetupAdminRoutes(r, c)
}
