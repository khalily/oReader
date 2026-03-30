package ratelimit

import (
	"sync"
	"time"
)

// MemoryLimiter implements RateLimiter using in-memory storage
// Each unique key gets its own token bucket
type MemoryLimiter struct {
	buckets map[string]*tokenBucketEntry
	mu      sync.RWMutex
}

// tokenBucketEntry wraps a token bucket with its window duration
type tokenBucketEntry struct {
	bucket  *TokenBucket
	window  time.Duration
	limit   int
	created time.Time
}

// NewMemoryLimiter creates a new in-memory rate limiter
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		buckets: make(map[string]*tokenBucketEntry),
	}
}

// Allow checks if a request should be allowed based on the rate limit
func (m *MemoryLimiter) Allow(key string, limit int, window time.Duration) Result {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create bucket for this key
	entry, exists := m.buckets[key]
	if !exists {
		// Clean up old entries periodically
		m.cleanupOldEntries()

		// Create new bucket starting full
		entry = &tokenBucketEntry{
			bucket:  NewTokenBucket(int64(limit), window),
			window:  window,
			limit:   limit,
			created: time.Now(),
		}
		m.buckets[key] = entry
	}

	// Try to consume a token
	allowed := entry.bucket.TryConsume()

	// Calculate remaining tokens
	remaining := entry.bucket.Available()
	if remaining > int64(limit) {
		remaining = int64(limit)
	}

	// Calculate reset time (when tokens will be fully refilled)
	resetAt := time.Now().Add(window)

	return Result{
		Allowed:   allowed,
		Limit:     limit,
		Remaining: int(remaining),
		ResetAt:   resetAt,
	}
}

// cleanupOldEntries removes buckets that haven't been used recently
// This prevents memory leaks from stale entries
func (m *MemoryLimiter) cleanupOldEntries() {
	// Only cleanup periodically to avoid performance impact
	// Clean up entries older than 10 minutes
	now := time.Now()
	for key, entry := range m.buckets {
		if now.Sub(entry.created) > 10*time.Minute {
			delete(m.buckets, key)
		}
	}
}
