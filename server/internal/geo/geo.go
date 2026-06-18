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

	filtered := make([]*model.VpnConfig, 0, len(configs))
	for _, cfg := range configs {
		country := g.CountryCode(cfg.Server)
		cfg.Country = country
		if country == "" {
			log.Printf("WARN: unknown country for %s (%s), skipping", cfg.Server, cfg.Name)
			continue
		}
		if country == "RU" {
			continue
		}
		filtered = append(filtered, cfg)
	}
	return filtered
}
