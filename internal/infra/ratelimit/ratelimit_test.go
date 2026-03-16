package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRateLimiter_Interface tests the RateLimiter interface contract
// This test defines the expected behavior that all implementations must follow
func TestRateLimiter_Interface(t *testing.T) {
	// Create an in-memory implementation for testing
	limiter := NewMemoryLimiter()

	t.Run("allows requests within limit", func(t *testing.T) {
		key := "user:123"
		limit := 10
		window := time.Minute

		// All requests within limit should be allowed
		for i := 0; i < limit; i++ {
			result := limiter.Allow(key, limit, window)
			assert.True(t, result.Allowed, "request %d should be allowed", i+1)
			assert.Equal(t, limit, result.Limit)
			assert.Equal(t, limit-i-1, result.Remaining)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		key := "user:exceed"
		limit := 5
		window := time.Minute

		// First 5 requests should be allowed
		for i := 0; i < limit; i++ {
			result := limiter.Allow(key, limit, window)
			assert.True(t, result.Allowed, "request %d should be allowed", i+1)
		}

		// 6th request should be blocked
		result := limiter.Allow(key, limit, window)
		assert.False(t, result.Allowed, "request exceeding limit should be blocked")
		assert.Equal(t, 0, result.Remaining)
	})

	t.Run("independent limits per key", func(t *testing.T) {
		limit := 3
		window := time.Minute

		// User 1 exhausts their limit
		for i := 0; i < limit; i++ {
			result := limiter.Allow("user:1", limit, window)
			assert.True(t, result.Allowed)
		}
		result1 := limiter.Allow("user:1", limit, window)
		assert.False(t, result1.Allowed)

		// User 2 should still have their full limit available
		result2 := limiter.Allow("user:2", limit, window)
		assert.True(t, result2.Allowed, "different user should have independent limit")
		assert.Equal(t, limit-1, result2.Remaining)
	})

	t.Run("resets after window expires", func(t *testing.T) {
		key := "user:reset"
		limit := 2
		window := 100 * time.Millisecond

		// Exhaust the limit
		for i := 0; i < limit; i++ {
			result := limiter.Allow(key, limit, window)
			assert.True(t, result.Allowed)
		}

		// Should be blocked
		result := limiter.Allow(key, limit, window)
		assert.False(t, result.Allowed)

		// Wait for window to expire
		time.Sleep(window + 50*time.Millisecond)

		// Should be allowed again
		result = limiter.Allow(key, limit, window)
		assert.True(t, result.Allowed, "limit should reset after window expires")
	})

	t.Run("returns correct reset time", func(t *testing.T) {
		key := "user:reset-time"
		limit := 5
		window := time.Second

		result := limiter.Allow(key, limit, window)
		assert.True(t, result.Allowed)
		assert.WithinDuration(t, time.Now().Add(window), result.ResetAt, 100*time.Millisecond)
	})

	t.Run("concurrent requests are handled correctly", func(t *testing.T) {
		key := "user:concurrent"
		limit := 100
		window := time.Minute

		var wg sync.WaitGroup
		allowedCount := 0
		var mu sync.Mutex

		// Launch 100 concurrent requests
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := limiter.Allow(key, limit, window)
				if result.Allowed {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// All 100 requests should be allowed (within limit of 100)
		assert.Equal(t, 100, allowedCount, "all concurrent requests within limit should be allowed")
	})

	t.Run("different limits for different endpoints", func(t *testing.T) {
		key := "user:multi-endpoint"
		loginLimit := 5
		feedLimit := 100
		window := time.Minute

		// Exhaust login endpoint limit
		for i := 0; i < loginLimit; i++ {
			result := limiter.Allow(key+":login", loginLimit, window)
			assert.True(t, result.Allowed)
		}
		result := limiter.Allow(key+":login", loginLimit, window)
		assert.False(t, result.Allowed, "login endpoint should be rate limited")

		// Feed endpoint should still work
		result = limiter.Allow(key+":feeds", feedLimit, window)
		assert.True(t, result.Allowed, "feeds endpoint should have separate limit")
	})

	t.Run("handles zero limit gracefully", func(t *testing.T) {
		key := "user:zero"
		result := limiter.Allow(key, 0, time.Minute)
		assert.False(t, result.Allowed, "zero limit should block all requests")
		assert.Equal(t, 0, result.Limit)
		assert.Equal(t, 0, result.Remaining)
	})

	t.Run("handles large window durations", func(t *testing.T) {
		key := "user:large-window"
		limit := 1
		window := 24 * time.Hour

		result := limiter.Allow(key, limit, window)
		assert.True(t, result.Allowed)
		assert.WithinDuration(t, time.Now().Add(window), result.ResetAt, time.Second)
	})
}

// TestMemoryLimiter_Cleanup tests that old entries are cleaned up
func TestMemoryLimiter_Cleanup(t *testing.T) {
	limiter := NewMemoryLimiter()

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		// Create entries with very short window
		key := "user:cleanup"
		limit := 1
		window := 50 * time.Millisecond

		// Use the limit
		result := limiter.Allow(key, limit, window)
		require.True(t, result.Allowed)

		// Wait for window to expire
		time.Sleep(window + 50*time.Millisecond)

		// Trigger cleanup by making a new request
		newKey := "user:new"
		limiter.Allow(newKey, 10, time.Minute)

		// Original key should have a fresh start after cleanup
		result = limiter.Allow(key, limit, window)
		assert.True(t, result.Allowed)
	})
}

// TestResult_Structure tests the Result struct
func TestResult_Structure(t *testing.T) {
	t.Run("result contains all required fields", func(t *testing.T) {
		resetTime := time.Now().Add(time.Minute)
		result := Result{
			Allowed:   true,
			Limit:     100,
			Remaining: 99,
			ResetAt:   resetTime,
		}

		assert.True(t, result.Allowed)
		assert.Equal(t, 100, result.Limit)
		assert.Equal(t, 99, result.Remaining)
		assert.Equal(t, resetTime, result.ResetAt)
	})
}
