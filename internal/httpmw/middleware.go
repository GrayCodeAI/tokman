package httpmw

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// DefaultRequests is the default max requests per window.
	DefaultRequests = 100
	// DefaultWindow is the default time window for rate limiting.
	DefaultWindow = time.Minute
)

// RateLimiter provides per-IP rate limiting for HTTP handlers.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	requests int           // max requests per window
	window   time.Duration // time window
}

type clientLimiter struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter creates a rate limiter with the given requests per window.
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*clientLimiter),
		requests: requests,
		window:   window,
	}
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			for ip, c := range rl.clients {
				if time.Now().After(c.resetAt) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// NewDefault creates a rate limiter with 100 requests per minute.
func NewDefault() *RateLimiter {
	return NewRateLimiter(DefaultRequests, DefaultWindow)
}

// Middleware wraps an http.Handler with per-IP rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		rl.mu.Lock()
		client, exists := rl.clients[ip]
		if !exists || time.Now().After(client.resetAt) {
			client = &clientLimiter{resetAt: time.Now().Add(rl.window)}
			rl.clients[ip] = client
		}
		client.count++
		allowed := client.count <= rl.requests
		rl.mu.Unlock()

		if !allowed {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
