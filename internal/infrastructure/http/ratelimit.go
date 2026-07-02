package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newRateLimiter(r rate.Limit, b int) *rateLimiter {
	rl := &rateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

var webAuthnLimiter = newRateLimiter(rate.Every(6*time.Second), 10)

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !webAuthnLimiter.get(ip).Allow() {
			httpError(w, http.StatusTooManyRequests, "too many requests")
			return
		}

		next.ServeHTTP(w, r)
	})
}
