package config

import (
	"os"
	"testing"
	"time"
)

func TestParseDotEnv_Normal(t *testing.T) {
	content := `KEY=value
# comment
ANOTHER=val`

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 2 {
		t.Errorf("expected 2 vars parsed (KEY, ANOTHER), got %d", n)
	}
	if os.Getenv("KEY") != "value" {
		t.Errorf("KEY = %q, want %q", os.Getenv("KEY"), "value")
	}
	if os.Getenv("ANOTHER") != "val" {
		t.Errorf("ANOTHER = %q, want %q", os.Getenv("ANOTHER"), "val")
	}
}

func TestParseDotEnv_QuotedValues(t *testing.T) {
	content := `KEY="quoted value"
SINGLE='single quoted'
MIXED="value with spaces"`

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 3 {
		t.Errorf("expected 3 vars, got %d", n)
	}
	if os.Getenv("KEY") != "quoted value" {
		t.Errorf("KEY = %q, want %q", os.Getenv("KEY"), "quoted value")
	}
	if os.Getenv("SINGLE") != "single quoted" {
		t.Errorf("SINGLE = %q, want %q", os.Getenv("SINGLE"), "single quoted")
	}
}

func TestParseDotEnv_EmptyLinesAndComments(t *testing.T) {
	content := `
# comment line
# another comment

KEY=val

`
	os.Clearenv()
	n := parseDotEnv(content)

	if n != 1 {
		t.Errorf("expected 1 var, got %d", n)
	}
}

func TestParseDotEnv_WhitespaceAroundEquals(t *testing.T) {
	content := `KEY = value
  SPACED =  spaced value  `

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 2 {
		t.Errorf("expected 2 vars, got %d", n)
	}
	if os.Getenv("SPACED") != "spaced value" {
		t.Errorf("SPACED = %q, want %q", os.Getenv("SPACED"), "spaced value")
	}
}

func TestParseDotEnv_NoOverrideExisting(t *testing.T) {
	os.Clearenv()
	os.Setenv("EXISTING", "original")

	content := `EXISTING=should_not_override
NEW=will_be_set`

	n := parseDotEnv(content)

	if n != 1 {
		t.Errorf("expected 1 new var (NEW), got %d", n)
	}
	if os.Getenv("EXISTING") != "original" {
		t.Errorf("EXISTING = %q, want %q (should not override)", os.Getenv("EXISTING"), "original")
	}
	if os.Getenv("NEW") != "will_be_set" {
		t.Errorf("NEW = %q, want %q", os.Getenv("NEW"), "will_be_set")
	}
}

func TestParseDotEnv_MalformedLines(t *testing.T) {
	content := `NO_EQUALS_SIGN
=only_value
KEY=val`

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 1 {
		t.Errorf("expected 1 var (KEY), got %d", n)
	}
}

func TestParseDotEnv_WindowsLineEndings(t *testing.T) {
	content := "KEY=value\r\nANOTHER=val\r\n# comment\r\n\r\nTHIRD=test\r\n"

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 3 {
		t.Errorf("expected 3 vars parsed with \\r\\n, got %d", n)
	}
	if os.Getenv("KEY") != "value" {
		t.Errorf("KEY = %q, want %q", os.Getenv("KEY"), "value")
	}
	if os.Getenv("ANOTHER") != "val" {
		t.Errorf("ANOTHER = %q, want %q", os.Getenv("ANOTHER"), "val")
	}
	if os.Getenv("THIRD") != "test" {
		t.Errorf("THIRD = %q, want %q", os.Getenv("THIRD"), "test")
	}
}

func TestParseDotEnv_EmptyKey(t *testing.T) {
	content := `=value
KEY=val`

	os.Clearenv()
	n := parseDotEnv(content)

	if n != 1 {
		t.Errorf("expected 1 var, got %d", n)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.SubscriptionURL != "" {
		t.Errorf("SubscriptionURL = %q, want empty", cfg.SubscriptionURL)
	}
	if cfg.MockConfigs {
		t.Error("MockConfigs should be false by default")
	}
	if cfg.GeoIPDBPath != "./GeoLite2-Country.mmdb" {
		t.Errorf("GeoIPDBPath = %q, want %q", cfg.GeoIPDBPath, "./GeoLite2-Country.mmdb")
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("LISTEN_ADDR", ":9090")
	os.Setenv("SUBSCRIPTION_URL", "https://example.com/config")
	os.Setenv("MOCK_CONFIGS", "true")
	os.Setenv("SKIP_VERIFY_TLS", "false")
	os.Setenv("CORS_ORIGINS", "https://app.example.com")

	cfg := LoadConfig()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
	}
	if cfg.SubscriptionURL != "https://example.com/config" {
		t.Errorf("SubscriptionURL = %q, want %q", cfg.SubscriptionURL, "https://example.com/config")
	}
	if !cfg.MockConfigs {
		t.Error("MockConfigs should be true")
	}
	if cfg.SkipVerifyTLS {
		t.Error("SkipVerifyTLS should be false")
	}
	if cfg.CORSOrigins != "https://app.example.com" {
		t.Errorf("CORSOrigins = %q, want %q", cfg.CORSOrigins, "https://app.example.com")
	}
}

func TestGetDuration_Valid(t *testing.T) {
	os.Clearenv()
	os.Setenv("TEST_DURATION", "5m")

	d := getDuration("TEST_DURATION", time.Minute)
	if d != 5*time.Minute {
		t.Errorf("got %v, want %v", d, 5*time.Minute)
	}
}

func TestGetDuration_Invalid(t *testing.T) {
	os.Clearenv()
	os.Setenv("TEST_DURATION", "not-a-duration")

	d := getDuration("TEST_DURATION", time.Minute)
	if d != time.Minute {
		t.Errorf("got %v, want default %v", d, time.Minute)
	}
}
