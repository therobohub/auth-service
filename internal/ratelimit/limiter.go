package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter manages per-repository rate limiting
type Limiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewLimiter creates a new rate limiter
func NewLimiter(rps float64, burst int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// Allow checks if a request for the given repository is allowed
func (l *Limiter) Allow(repository string) bool {
	limiter := l.getLimiter(repository)
	return limiter.Allow()
}

// Wait waits until a request for the given repository is allowed
func (l *Limiter) Wait(repository string) error {
	limiter := l.getLimiter(repository)
	return limiter.Wait(context.TODO())
}

func (l *Limiter) getLimiter(repository string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[repository]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = l.limiters[repository]
	if exists {
		return limiter
	}

	// Create new limiter for this repository
	limiter = rate.NewLimiter(l.rps, l.burst)
	l.limiters[repository] = limiter

	return limiter
}

// Reset clears all rate limiters (useful for testing)
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limiters = make(map[string]*rate.Limiter)
}

// GetLimiterCount returns the number of active limiters (useful for testing)
func (l *Limiter) GetLimiterCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.limiters)
}
