package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stalkarr/internal/config"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := config.Init(dir); err != nil {
		t.Fatalf("config init failed: %v", err)
	}
	resetRateLimiter()
}

func newTestRouter() *gin.Engine {
	return NewRouter(nil)
}

func postJSON(router http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getWithToken(router http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- Auth tests ---

func TestSetupAndLogin(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()

	w := postJSON(router, "/api/setup", map[string]string{
		"username": "admin",
		"password": "testpassword",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %s", w.Code, w.Body.String())
	}

	w = postJSON(router, "/api/login", map[string]string{
		"username": "admin",
		"password": "testpassword",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}

	var res map[string]string
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["token"] == "" {
		t.Fatal("expected token in response, got none")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()

	postJSON(router, "/api/setup", map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})

	w := postJSON(router, "/api/login", map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSetupOnlyWorksOnce(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()

	postJSON(router, "/api/setup", map[string]string{
		"username": "admin",
		"password": "password",
	})

	w := postJSON(router, "/api/setup", map[string]string{
		"username": "admin2",
		"password": "password2",
	})
	if w.Code != http.StatusForbidden && w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 403 or 429 on second setup, got %d", w.Code)
	}
}

func TestProtectedRouteRequiresToken(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()

	w := getWithToken(router, "/api/settings", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()

	w := getWithToken(router, "/api/settings", "not.a.valid.token")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", w.Code)
	}
}

// --- API key leak tests ---

func getToken(t *testing.T, router http.Handler) string {
	t.Helper()
	postJSON(router, "/api/setup", map[string]string{
		"username": "admin",
		"password": "password",
	})
	w := postJSON(router, "/api/login", map[string]string{
		"username": "admin",
		"password": "password",
	})
	var res map[string]string
	json.Unmarshal(w.Body.Bytes(), &res)
	return res["token"]
}

func TestAPIKeyNeverLeaksInGetSettings(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()
	token := getToken(t, router)

	body, _ := json.Marshal(map[string]any{
		"name":    "Sonarr",
		"url":     "http://localhost:8989",
		"api_key": "super-secret-api-key-1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/settings/sonarr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("failed to save instance: %d %s", w.Code, w.Body.String())
	}

	w = getWithToken(router, "/api/settings", token)
	if w.Code != http.StatusOK {
		t.Fatalf("get settings failed: %d", w.Code)
	}

	responseBody := w.Body.String()
	if contains(responseBody, "super-secret-api-key-1234") {
		t.Fatal("SECURITY: raw API key found in GET /api/settings response")
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	instances := result["sonarr"].([]any)
	for _, inst := range instances {
		m := inst.(map[string]any)
		if _, hasKey := m["api_key"]; hasKey {
			t.Fatal("SECURITY: api_key field present in settings response")
		}
	}
}

func TestAPIKeyHintOnlyShowsLastFour(t *testing.T) {
	setupTestRouter(t)
	router := newTestRouter()
	token := getToken(t, router)

	body, _ := json.Marshal(map[string]any{
		"name":    "Sonarr",
		"url":     "http://localhost:8989",
		"api_key": "abcdefgh1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/settings/sonarr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var res map[string]any
	json.Unmarshal(w.Body.Bytes(), &res)

	hint, _ := res["api_key_hint"].(string)
	if hint != "••••••••1234" {
		t.Fatalf("expected hint ••••••••1234, got %s", hint)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBruteForceProtection(t *testing.T) {
	loginAttemptsMu.Lock()
	loginAttempts = make(map[string]*loginAttempt)
	loginAttemptsMu.Unlock()

	setupTestRouter(t)
	router := newTestRouter()

	postJSON(router, "/api/setup", map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})

	body := map[string]string{"username": "admin", "password": "wrongpassword"}

	for i := 0; i < 4; i++ {
		w := postJSON(router, "/api/login", body)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	w := postJSON(router, "/api/login", body)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d: %s", w.Code, w.Body.String())
	}

	t.Logf("lockout response body: %s", w.Body.String())

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["locked_out"] != true {
		t.Fatalf("expected locked_out: true in response, got: %s", w.Body.String())
	}
	if _, ok := resp["retry_after_secs"]; !ok {
		t.Fatal("expected retry_after_secs in lockout response")
	}
}
