package limiter

import (
	"errors"
	"sync"
	"time"
)

var ErrInvalidConfig = errors.New("limiter: capacity and refillRate must be positive")

// TokenBucket enforces a thread-safe token-bucket rate limit with continuous refill.
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64
	lastRefill time.Time
	now        func() time.Time
}

// NewTokenBucket creates a rate limiter holding up to capacity tokens, refilling at refillRate tokens/sec.
func NewTokenBucket(capacity, refillRate float64) (*TokenBucket, error) {
	if capacity <= 0 || refillRate <= 0 {
		return nil, ErrInvalidConfig
	}
	nowFn := time.Now
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: nowFn(),
		now:        nowFn,
	}, nil
}

// Allow consumes 1 token if available.
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN consumes n tokens atomically if available after refilling elapsed time.
func (tb *TokenBucket) AllowN(n float64) bool {
	if n <= 0 {
		return false
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed > 0 {
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
	if tb.tokens < n {
		return false
	}
	tb.tokens -= n
	return true
}

// Available returns the current token balance after applying elapsed refill.
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := tb.now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed > 0 {
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
	return tb.tokens
}
