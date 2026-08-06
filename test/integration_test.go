package test

import (

	"encoding/json"
	"io"

	"net/http"
	"net/http/httptest"
	"testing"

	"time"

	"ebpf-gateway/internal/auth"
	"ebpf-gateway/internal/config"
	"ebpf-gateway/internal/proxy"

	"github.com/golang-jwt/jwt/v5"
)

func testConfig(backendA, backendB string) *config.Config {
	return &config.Config{
		Listen: config.ListenConfig{Address: ":0"},
		Auth: config.AuthConfig{
			Mode: "both",
			ApiKeys: []config.ApiKeyConfig{
				{Key: "test-key-1", Name: "test-user", Scopes: []string{"read", "write"}},
				{Key: "test-key-2", Name: "readonly-user", Scopes: []string{"read"}},
				{Key: "test-key-admin", Name: "admin-user", Scopes: []string{"read", "write", "admin"}},

			},
			Jwt: config.JwtConfig{
				Algorithm: "HS256",
				Secret:    "test-secret-key-for-integration-tests",
				Issuer:    "test-issuer",
			},
		},
		Routes: []config.RouteConfig{
			{Path: "/api/users", Upstream: backendA, AuthRequired: true, RequiredScopes: []string{"read"}, TimeoutMs: 5000},
			{Path: "/api/orders", Upstream: backendB, AuthRequired: true, RequiredScopes: []string{"read"}, TimeoutMs: 5000},
			{Path: "/api/admin", Upstream: backendA, AuthRequired: true, RequiredScopes: []string{"admin"}, TimeoutMs: 5000},
			{Path: "/health", Upstream: backendA, AuthRequired: false, TimeoutMs: 2000},
		},
		HealthCheck: config.HealthCheckConfig{
			TimeoutMs:          2000,
			Path:               "/health",
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		},
	}
}

func buildHandler(cfg *config.Config) http.Handler {
	router, _ := proxy.NewRouter(cfg, nil)
	enforcer := auth.NewZeroTrustEnforcer(cfg.Auth, cfg.Routes)
	return proxy.Chain(
		proxy.RequestID(),
		proxy.CORS(),
		proxy.Logging(),
		enforcer.Middleware(),
	)(router)
}

func TestHealthCheck(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("expected OK, got %s", string(body))
	}
}

func TestProxyForwarding(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "test", "user_id": r.Header.Get("X-User-ID")})

	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["user_id"] != "test-user" {
		t.Errorf("expected user_id test-user, got %s", result["user_id"])
	}
}

func TestAPIKeyAuth(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":     r.Header.Get("X-User-ID"),
			"auth_method": r.Header.Get("X-Auth-Method"),
			"scopes":      r.Header.Get("X-User-Scopes"),
		})
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["auth_method"] != "apikey" {
		t.Errorf("expected apikey, got %s", result["auth_method"])
	}
	if result["user_id"] != "test-user" {
		t.Errorf("expected test-user, got %s", result["user_id"])
	}
}

func TestJWTAuth(t *testing.T) {

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"user_id":     r.Header.Get("X-User-ID"),
			"auth_method": r.Header.Get("X-Auth-Method"),
		})
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    "jwt-test-user",
		"iss":    "test-issuer",
		"scopes": []string{"read", "write"},
		"exp":    time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte("test-secret-key-for-integration-tests"))

	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["auth_method"] != "jwt" {
		t.Errorf("expected jwt, got %s", result["auth_method"])
	}
	if result["user_id"] != "jwt-test-user" {

		t.Errorf("expected jwt-test-user, got %s", result["user_id"])
	}
}

func TestUnauthorizedRejection(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not have been called")
	}))

	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/users")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestInsufficientScopes(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not have been called")
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)

	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/admin", nil)
	req.Header.Set("X-API-Key", "test-key-2")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestAdminAccess(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"admin": "true"})
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)

	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/admin", nil)
	req.Header.Set("X-API-Key", "test-key-admin")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRequestIDPropagation(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"request_id": r.Header.Get("X-Request-ID")})
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	responseID := resp.Header.Get("X-Request-ID")
	if responseID == "" {
		t.Error("expected X-Request-ID in response")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["request_id"] == "" {
		t.Error("expected X-Request-ID propagated to backend")
	}
	if result["request_id"] != responseID {
		t.Errorf("IDs mismatch: response=%s backend=%s", responseID, result["request_id"])
	}
}

func TestCORSPreflight(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not be called for OPTIONS")
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/users", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("unexpected CORS origin: %s", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestInvalidAPIKey(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not have been called")
	}))
	defer backend.Close()

	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMultiRouteForwarding(t *testing.T) {
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "a"})
	}))
	defer backendA.Close()
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "b"})
	}))
	defer backendB.Close()
	cfg := testConfig(backendA.URL, backendB.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	reqA, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	reqA.Header.Set("X-API-Key", "test-key-1")
	respA, _ := http.DefaultClient.Do(reqA)
	defer respA.Body.Close()

	var resultA map[string]string
	json.NewDecoder(respA.Body).Decode(&resultA)
	if resultA["service"] != "a" {
		t.Errorf("expected service a, got %s", resultA["service"])
	}

	reqB, _ := http.NewRequest("GET", ts.URL+"/api/orders", nil)
	reqB.Header.Set("X-API-Key", "test-key-1")
	respB, _ := http.DefaultClient.Do(reqB)
	defer respB.Body.Close()
	var resultB map[string]string
	json.NewDecoder(respB.Body).Decode(&resultB)
	if resultB["service"] != "b" {
		t.Errorf("expected service b, got %s", resultB["service"])
	}

}

func TestExpiredJWT(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not have been called")
	}))
	defer backend.Close()
	cfg := testConfig(backend.URL, backend.URL)
	ts := httptest.NewServer(buildHandler(cfg))
	defer ts.Close()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    "expired-user",
		"iss":    "test-issuer",
		"scopes": []string{"read"},
		"exp":    time.Now().Add(-time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte("test-secret-key-for-integration-tests"))

	req, _ := http.NewRequest("GET", ts.URL+"/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {

		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestProxyHeaderInjection(t *testing.T) {

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
