package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/model"
)

func SetupRoutes(r *gin.Engine, c *cache.ConfigCache) {
	r.GET("/health", func(ctx *gin.Context) {
		status, msg := c.Status()
		ctx.JSON(http.StatusOK, model.StatusResponse{
			Status:  status,
			Message: msg,
		})
	})

	api := r.Group("/api")
	{
		api.GET("/status", func(ctx *gin.Context) {
			status, msg := c.Status()
			best := c.GetBestConfig()
			updated := c.GetUpdated()

			resp := model.StatusResponse{
				Status:        status,
				Message:       msg,
				ConfigsTested: len(c.GetConfigs()),
				Updated:       updated,
			}
			if best != nil {
				resp.BestName = best.Name
				resp.BestLatency = best.LatencyMs
			}
			ctx.JSON(http.StatusOK, resp)
		})

		api.GET("/best-config", func(ctx *gin.Context) {
			status, msg := c.Status()
			if status != model.StatusReady {
				ctx.JSON(http.StatusServiceUnavailable, model.StatusResponse{
					Status:  status,
					Message: msg,
				})
				return
			}

			best := c.GetBestConfig()
			if best == nil {
				ctx.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
					Error:   model.ErrNoAvailableConfigs,
					Message: "No available non-Russia configs found",
				})
				return
			}

			ctx.JSON(http.StatusOK, model.BestConfigResponse{
				Config:  best,
				Updated: c.GetUpdated(),
			})
		})

		api.GET("/configs", func(ctx *gin.Context) {
			status, msg := c.Status()
			if status != model.StatusReady {
				ctx.JSON(http.StatusServiceUnavailable, model.StatusResponse{
					Status:  status,
					Message: msg,
				})
				return
			}

			configs := c.GetConfigs()
			ctx.JSON(http.StatusOK, model.ConfigListResponse{
				Configs: configs,
				Updated: c.GetUpdated(),
				Total:   len(configs),
			})
		})

		api.POST("/refresh", func(ctx *gin.Context) {
			status, msg := c.Status()
			if status == model.StatusReady || status == model.StatusError {
				go c.Refresh()
				ctx.JSON(http.StatusOK, gin.H{"status": "refreshing"})
			} else {
				ctx.JSON(http.StatusConflict, model.StatusResponse{
					Status:  status,
					Message: msg,
				})
			}
		})
	}
}
