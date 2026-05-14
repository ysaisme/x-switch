package security

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	globalRPM int
	perSite   map[string]int
}

type bucket struct {
	tokens   float64
	maxToken float64
	lastTime time.Time
}

func NewRateLimiter(globalRPM int, perSite map[string]int) *RateLimiter {
	return &RateLimiter{
		buckets:   make(map[string]*bucket),
		globalRPM: globalRPM,
		perSite:   perSite,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl.globalRPM <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		if !rl.allow("global", float64(rl.globalRPM)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) AllowSite(siteID string) bool {
	limit, ok := rl.perSite[siteID]
	if !ok {
		limit = rl.globalRPM
	}
	if limit <= 0 {
		return true
	}
	return rl.allow("site:"+siteID, float64(limit))
}

func (rl *RateLimiter) allow(key string, rate float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{
			tokens:   rate,
			maxToken: rate,
			lastTime: now,
		}
		rl.buckets[key] = b
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * (rate / 60.0)
	if b.tokens > b.maxToken {
		b.tokens = b.maxToken
	}
	b.lastTime = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}
