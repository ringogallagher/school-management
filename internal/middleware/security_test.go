package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_BlocksAfterLimitAndResets(t *testing.T) {
	now := time.Unix(1000, 0)
	rl := NewRateLimiter(3, time.Minute)
	rl.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := rl.Allow("1.2.3.4"); !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if ok, retry := rl.Allow("1.2.3.4"); ok || retry <= 0 {
		t.Fatal("4th request should be blocked with a retry delay")
	}
	if ok, _ := rl.Allow("5.6.7.8"); !ok {
		t.Error("a different client must not be affected")
	}
	now = now.Add(time.Minute)
	if ok, _ := rl.Allow("1.2.3.4"); !ok {
		t.Error("limit should reset after the window")
	}
}

func TestRateLimiter_Middleware429(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "9.9.9.9:1234"

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request: got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Errorf("second request: expected 429 with Retry-After, got %d", rec.Code)
	}
}

func TestRequireRole_NoClaimsDoesNotPanic(t *testing.T) {
	h := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without claims, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing header %s", h)
		}
	}
}
