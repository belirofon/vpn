package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/config"
	"vpn-server/internal/geo"
	"vpn-server/internal/handler"
)

func main() {
	config.LoadDotEnv()
	cfg := config.LoadConfig()

	// GeoIP (optional)
	var geoDB *geo.GeoDB
	if cfg.GeoIPDBPath != "" {
		db, err := geo.OpenGeoDB(cfg.GeoIPDBPath)
		if err != nil {
			log.Printf("WARN: GeoIP not available (%v), geo-filtering disabled", err)
		} else {
			geoDB = db
			defer geoDB.Close()
		}
	} else {
		log.Println("INFO: GEOIP_DB_PATH not set, geo-filtering disabled")
	}

	cache := cache.NewCache(cfg, geoDB)
	cache.Init()
	cache.Start()
	defer cache.Stop()

	// Gin router
	r := gin.Default()
	r.Use(corsMiddleware(cfg.CORSOrigins))

	handler.SetupRoutes(r, cache, &cfg)

	// HTTP server
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("INFO: server starting on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL: server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("INFO: shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: server shutdown error: %v", err)
	}

	log.Println("INFO: server stopped")
}

func corsMiddleware(origins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origins == "*" || origins == origin {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origins)
		} else if origins != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origins)
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
