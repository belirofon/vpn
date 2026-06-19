package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"vpn-server/internal/cache"
	"vpn-server/internal/config"
	"vpn-server/internal/geo"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AdminEmail:    "admin@test.com",
		AdminPassword: "testpass123",
	}

	r := gin.New()
	SetupAdminRoutes(r, cfg, nil)
	return r, cfg
}

func TestAdminLogin_Success(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := `{"email":"admin@test.com","password":"testpass123"}`
	req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	clearAdminToken()
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := `{"email":"admin@test.com","password":"wrongpass"}`
	req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminLogin_WrongEmail(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := `{"email":"hacker@test.com","password":"testpass123"}`
	req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminLogin_EmptyCredentials(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminEndpoints_RequiresAuth(t *testing.T) {
	r, _ := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/api/admin/endpoints", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", w.Code)
	}
}

func TestAdminEndpoints_WithAuth(t *testing.T) {
	r, cfg := setupTestRouter(t)

	// Login first
	loginBody := `{"email":"admin@test.com","password":"testpass123"}`
	loginReq := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp.Token

	// Use token to access endpoints
	req := httptest.NewRequest("GET", "/api/admin/endpoints", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth, got %d", w.Code)
	}

	var resp struct {
		Endpoints []interface{} `json:"endpoints"`
		Total     int           `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total == 0 {
		t.Fatal("expected non-empty endpoints list")
	}
	clearAdminToken()
	_ = cfg
}

func TestAdminLogout(t *testing.T) {
	r, _ := setupTestRouter(t)

	// Login
	loginBody := `{"email":"admin@test.com","password":"testpass123"}`
	loginReq := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp.Token

	// Logout
	logoutReq := httptest.NewRequest("POST", "/api/admin/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutW := httptest.NewRecorder()
	r.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d", logoutW.Code)
	}

	// After logout, endpoints should be inaccessible
	epReq := httptest.NewRequest("GET", "/api/admin/endpoints", nil)
	epReq.Header.Set("Authorization", "Bearer "+token)
	epW := httptest.NewRecorder()
	r.ServeHTTP(epW, epReq)

	if epW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", epW.Code)
	}
}

func TestAdminHealth_WithCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AdminEmail:       "admin@test.com",
		AdminPassword:    "testpass123",
		SubscriptionURL:  "https://example.com/sub",
		RefreshInterval:  30 * time.Minute,
	}

	cache := cache.NewCache(*cfg, &geo.GeoDB{})
	cache.Init()

	r := gin.New()
	SetupAdminRoutes(r, cfg, cache)

	// Login
	loginBody := `{"email":"admin@test.com","password":"testpass123"}`
	loginReq := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp.Token

	// Health check
	req := httptest.NewRequest("GET", "/api/admin/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] == nil {
		t.Fatal("expected status field in health response")
	}
	if resp["uptime"] == nil {
		t.Fatal("expected uptime field in health response")
	}
	if resp["subscription_url"] != "https://example.com/sub" {
		t.Fatalf("expected subscription_url to match, got %v", resp["subscription_url"])
	}
	clearAdminToken()
}
