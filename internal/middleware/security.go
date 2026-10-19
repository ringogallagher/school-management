package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// SecurityHeaders adds defensive HTTP headers to every response.
// The CSP only allows scripts and styles served by this application (htmx is vendored),
// inline style attributes are allowed for progress bars.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; "+
				"form-action 'self'; frame-ancestors 'none'; base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter is a fixed-window, per-client-IP limiter used to slow down password guessing.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]*window
	now     func() time.Time
}

type window struct {
	start time.Time
	count int
}

// NewRateLimiter allows `limit` requests per `window` for each client IP
func NewRateLimiter(limit int, win time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: win, clients: map[string]*window{}, now: time.Now}
}

// Allow records a request from key and reports whether it is within the limit,
// plus how long to wait when it is not.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	// opportunistic cleanup so the map cannot grow forever
	if len(rl.clients) > 10000 {
		for k, w := range rl.clients {
			if now.Sub(w.start) > rl.window {
				delete(rl.clients, k)
			}
		}
	}

	w, ok := rl.clients[key]
	if !ok || now.Sub(w.start) >= rl.window {
		rl.clients[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= rl.limit {
		return false, w.start.Add(rl.window).Sub(now)
	}
	w.count++
	return true, 0
}

func clientIP(r *http.Request) string {
	// chi's RealIP middleware has already replaced RemoteAddr with X-Real-IP / X-Forwarded-For
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Middleware rejects requests over the limit with 429 Too Many Requests
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retry := rl.Allow(clientIP(r))
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			http.Error(w, `{"code":429,"message":"too many attempts, try again later"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
