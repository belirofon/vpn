// Package main VPN Server & Client
//
// Auto-proxy subscription fetcher · latency tester · geo-filter · one-tap mobile client.
//
//	@title          VPN Server & Client
//	@version        1.0
//	@description    Auto-proxy subscription fetcher · latency tester · geo-filter · one-tap mobile client
//	@host           localhost:8080
//	@BasePath       /
//	@schemes        http https
//
//	@securityDefinitions.apikey  bearerAuth
//	@in                          header
//	@name                        Authorization
//	@description                 Type 'Bearer <token>' to authenticate admin requests
//
//	@tag.name         Public
//	@tag.description  Public endpoints for VPN configuration
//	@tag.name         Admin
//	@tag.description  Authenticated admin management endpoints
//
// swagger:meta
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"vpn-server/internal/cache"
	"vpn-server/internal/config"
	"vpn-server/internal/geo"
	"vpn-server/internal/handler"
	_ "vpn-server/docs"
)

func main() {
	config.LoadDotEnv()
	cfg := config.LoadConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var geoDB *geo.DB
	if cfg.GeoIPDBPath != "" {
		db, err := geo.OpenGeoDB(cfg.GeoIPDBPath)
		if err != nil {
			logger.Warn("GeoIP not available, geo-filtering disabled", "error", err)
		} else {
			geoDB = db
			defer geoDB.Close()
		}
	} else {
		logger.Info("GEOIP_DB_PATH not set, geo-filtering disabled")
	}

	c := cache.NewCache(cfg, geoDB, logger)
	c.Init()
	c.Start()
	defer c.Stop()

	r := gin.Default()
	r.Use(corsMiddleware(cfg.CORSOrigins))

	handler.SetupRoutes(r, c)
	handler.SetupUpdateRoutes(r)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func corsMiddleware(origins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		switch {
		case origins == "*":
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		case origins == origin:
			c.Writer.Header().Set("Access-Control-Allow-Origin", origins)
		case origins != "":
			c.Writer.Header().Set("Access-Control-Allow-Origin", origins)
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
