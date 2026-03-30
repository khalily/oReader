package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
// Tokens are added at a constant rate up to a maximum capacity
type TokenBucket struct {
	capacity    int64         // Maximum number of tokens
	tokens      int64         // Current number of tokens
	refillRate  int64         // Tokens per second
	window      time.Duration // Time window for refill calculation
	lastRefill  time.Time     // Last time tokens were refilled
	mu          sync.Mutex    // Protects concurrent access
}

// NewTokenBucket creates a new token bucket with the given capacity and refill window
// The bucket starts full (with 'capacity' tokens)
func NewTokenBucket(capacity int64, window time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: int64(float64(capacity) / window.Seconds()),
		window:     window,
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume one token from the bucket
// Returns true if a token was available, false otherwise
func (tb *TokenBucket) TryConsume() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// Available returns the current number of available tokens
func (tb *TokenBucket) Available() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// refill adds tokens based on the time elapsed since the last refill
// The number of tokens added = elapsed_seconds * refill_rate
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if elapsed <= 0 {
		return
	}

	// Calculate tokens to add based on elapsed time
	tokensToAdd := int64(float64(elapsed.Seconds()) * float64(tb.refillRate))

	// Don't add more than needed to reach capacity
	if tb.tokens + tokensToAdd > tb.capacity {
		tb.tokens = tb.capacity
	} else {
		tb.tokens += tokensToAdd
	}

	tb.lastRefill = now
}
