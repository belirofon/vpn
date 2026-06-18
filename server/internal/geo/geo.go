package geo

import (
	"fmt"
	"log"
	"net"

	"github.com/oschwald/geoip2-golang"
	"vpn-server/internal/model"
)

type GeoDB struct {
	db *geoip2.Reader
}

func OpenGeoDB(path string) (*GeoDB, error) {
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open GeoIP DB %s: %w", path, err)
	}
	return &GeoDB{db: db}, nil
}

func (g *GeoDB) Close() {
	if g.db != nil {
		g.db.Close()
	}
}

func (g *GeoDB) CountryCode(ip string) string {
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

func FilterNonRussia(configs []*model.VpnConfig, g *GeoDB) []*model.VpnConfig {
	if g == nil {
		return configs
	}

	// First pass: try to get non-RU configs
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

	// Prefer non-RU, fall back to all if empty
	if len(nonRU) > 0 {
		log.Printf("INFO: GeoIP: %d non-RU, %d RU, %d unknown", len(nonRU), len(ru), len(unknown))
		return nonRU
	}

	log.Printf("INFO: GeoIP: all configs are RU/unknown (%d RU, %d unknown), using as fallback", len(ru), len(unknown))
	return append(ru, unknown...)
}
