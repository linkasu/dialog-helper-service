package auth

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{limiters: make(map[string]*rate.Limiter)}
}

func (r *RateLimiter) Allow(key string, perMinute int) bool {
	if perMinute <= 0 {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	limiter := r.limiters[key]
	if limiter == nil {
		interval := time.Minute / time.Duration(perMinute)
		limiter = rate.NewLimiter(rate.Every(interval), perMinute)
		r.limiters[key] = limiter
	}

	return limiter.Allow()
}
