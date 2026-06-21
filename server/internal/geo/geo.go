package geo

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/oschwald/geoip2-golang"
	"vpn-server/internal/model"
)

type DB struct {
	db *geoip2.Reader
}

func OpenGeoDB(path string) (*DB, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open GeoIP DB %s: %w", path, err)
	}
	return &DB{db: db}, nil
}

func (g *DB) Close() {
	if g.db != nil {
		g.db.Close()
	}
}

func (g *DB) CountryCode(ip string) string {
	if g == nil || g.db == nil {
		return ""
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}

	record, err := g.db.Country(parsed)
	if err != nil {
		return ""
	}

	return record.Country.IsoCode
}

func FilterNonRussia(configs []*model.VpnConfig, g *DB) []*model.VpnConfig {
	if g == nil {
		return configs
	}

	var nonRU, ru, unknown []*model.VpnConfig
	for _, cfg := range configs {
		country := g.CountryCode(cfg.Server)
		cfg.Country = country
		switch country {
		case "":
			unknown = append(unknown, cfg)
		case "RU":
			ru = append(ru, cfg)
		default:
			nonRU = append(nonRU, cfg)
		}
	}

	if len(nonRU) > 0 {
		slog.Info("GeoIP filter", "non_ru", len(nonRU), "ru", len(ru), "unknown", len(unknown))
		return nonRU
	}

	slog.Info("GeoIP filter: all configs are RU/unknown, using as fallback", "ru", len(ru), "unknown", len(unknown))
	return append(ru, unknown...)
}
