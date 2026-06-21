package geo

import (
	"testing"

	"vpn-server/internal/model"
)

func TestFilterNonRussia_NilGeoDB(t *testing.T) {
	configs := []*model.VpnConfig{{Server: "192.168.1.1"}, {Server: "10.0.0.1"}}

	result := FilterNonRussia(configs, nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 configs with nil DB, got %d", len(result))
	}
}

func TestFilterNonRussia_NoConfigs(t *testing.T) {
	result := FilterNonRussia(nil, nil)

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestOpenGeoDB_InvalidPath(t *testing.T) {
	db, err := OpenGeoDB("/nonexistent/path.mmdb")

	if err == nil {
		t.Error("expected error for invalid path")
	}
	if db != nil {
		db.Close()
		t.Error("expected nil DB for invalid path")
	}
}

func TestGeoDB_CountryCode_NilReceiver(t *testing.T) {
	var nilDB *DB
	code := nilDB.CountryCode("8.8.8.8")

	if code != "" {
		t.Errorf("expected empty string for nil receiver, got %q", code)
	}
}

func TestGeoDB_CountryCode_InvalidIP(t *testing.T) {
	db := &DB{}
	code := db.CountryCode("not-an-ip")

	if code != "" {
		t.Errorf("expected empty string for invalid IP, got %q", code)
	}
}

func TestFilterNonRussia_AllNonRU(t *testing.T) {
	// Without a real GeoIP DB, all configs pass through
	configs := []*model.VpnConfig{
		{Server: "192.168.1.10"},
		{Server: "192.168.1.20"},
	}

	result := FilterNonRussia(configs, &DB{})

	if len(result) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(result))
	}
}
