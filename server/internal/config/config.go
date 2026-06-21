package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadDotEnv reads a .env file from cwd or parent directories
// and sets matching env vars (won't override already-set vars).
// Skip if SKIP_DOTENV=true is set in the environment.
func LoadDotEnv() {
	if os.Getenv("SKIP_DOTENV") == "true" {
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		path := filepath.Join(dir, ".env")
		if data, err := os.ReadFile(path); err == nil {
			n := parseDotEnv(string(data))
			slog.Info("loaded .env", "path", path, "vars", n)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// parseDotEnv parses key=value pairs from .env content.
// Supports \n and \r\n line endings, quoted values, and comments.
func parseDotEnv(content string) int {
	// Normalize \r\n to \n for Windows compatibility.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"'")
		if k == "" {
			continue
		}
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
			count++
		}
	}
	return count
}

type Config struct {
	SubscriptionURL string
	ListenAddr      string
	RefreshInterval time.Duration
	PingTimeout     time.Duration
	GeoIPDBPath     string
	MockConfigs     bool
	SkipVerifyTLS   bool   // SKIP_VERIFY_TLS — skip TLS cert verification (default: true for proxy testing compat)
	CORSOrigins     string // CORS_ORIGINS — allowed CORS origins (default: *)
	AdminEmail      string // ADMIN_EMAIL — admin login email
	AdminPassword   string // ADMIN_PASSWORD — admin login password
	WarpEnabled     bool   // WARP_ENABLED — enable WARP Cloudflare config generation
}

func LoadConfig() Config {
	cfg := Config{
		SubscriptionURL: os.Getenv("SUBSCRIPTION_URL"),
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		RefreshInterval: getDuration("REFRESH_INTERVAL", 30*time.Minute),
		PingTimeout:     getDuration("PING_TIMEOUT", 5*time.Second),
		GeoIPDBPath:     getEnv("GEOIP_DB_PATH", "./GeoLite2-Country.mmdb"),
		MockConfigs:     os.Getenv("MOCK_CONFIGS") == "true",
		SkipVerifyTLS:   os.Getenv("SKIP_VERIFY_TLS") != "false",
		CORSOrigins:     getEnv("CORS_ORIGINS", "*"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		WarpEnabled:     os.Getenv("WARP_ENABLED") == "true",
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}
