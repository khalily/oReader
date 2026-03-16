package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTokenBucket_BasicBehavior tests the token bucket algorithm
func TestTokenBucket_BasicBehavior(t *testing.T) {
	t.Run("refill rate creates new tokens over time", func(t *testing.T) {
		// 10 tokens per second, capacity of 10
		bucket := NewTokenBucket(10, time.Second)

		// Drain all tokens
		for i := 0; i < 10; i++ {
			assert.True(t, bucket.TryConsume(), "should have token available")
		}

		// Should be empty now
		assert.False(t, bucket.TryConsume(), "bucket should be empty")

		// Wait for partial refill (0.5 seconds = ~5 tokens)
		time.Sleep(500 * time.Millisecond)

		// Should have some tokens available
		consumed := 0
		for bucket.TryConsume() {
			consumed++
		}
		assert.Greater(t, consumed, 0, "should have refilled some tokens")
		assert.LessOrEqual(t, consumed, 6, "should not exceed expected refill")
	})

	t.Run("does not exceed capacity", func(t *testing.T) {
		capacity := 5
		bucket := NewTokenBucket(int64(capacity), time.Second)

		// Drain all tokens
		for i := 0; i < capacity; i++ {
			assert.True(t, bucket.TryConsume())
		}

		// Wait for more than a full refill
		time.Sleep(1100 * time.Millisecond)

		// Should only have capacity tokens, not more
		consumed := 0
		for bucket.TryConsume() {
			consumed++
		}
		assert.Equal(t, capacity, consumed, "should never exceed capacity")
	})

	t.Run("handles very fast refill rates", func(t *testing.T) {
		// 1000 tokens per second
		bucket := NewTokenBucket(1000, time.Second)

		// Should start full
		consumed := 0
		for bucket.TryConsume() && consumed < 1001 {
			consumed++
		}
		assert.Equal(t, 1000, consumed, "should start at capacity")
	})

	t.Run("handles slow refill rates", func(t *testing.T) {
		// 1 token per minute
		bucket := NewTokenBucket(1, time.Minute)

		// Should start with 1 token
		assert.True(t, bucket.TryConsume(), "should have initial token")
		assert.False(t, bucket.TryConsume(), "should be empty after consuming 1 token")
	})

	t.Run("concurrent consumption is thread-safe", func(t *testing.T) {
		capacity := int64(100)
		bucket := NewTokenBucket(capacity, time.Second)

		var wg sync.WaitGroup
		consumed := int64(0)
		var mu sync.Mutex

		// Launch 100 concurrent consumers
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if bucket.TryConsume() {
					mu.Lock()
					consumed++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, capacity, consumed, "all tokens should be consumed exactly once")
	})

	t.Run("returns available token count", func(t *testing.T) {
		bucket := NewTokenBucket(10, time.Second)

		assert.Equal(t, int64(10), bucket.Available(), "should start at capacity")

		bucket.TryConsume()
		assert.Equal(t, int64(9), bucket.Available(), "should decrease after consumption")

		// Consume 5 more
		for i := 0; i < 5; i++ {
			bucket.TryConsume()
		}
		assert.Equal(t, int64(4), bucket.Available(), "should reflect correct count")
	})

	t.Run("handles zero capacity", func(t *testing.T) {
		bucket := NewTokenBucket(0, time.Second)
		assert.False(t, bucket.TryConsume(), "zero capacity bucket should never allow consumption")
		assert.Equal(t, int64(0), bucket.Available())
	})
}

// TestTokenBucket_Precision tests timing precision of the algorithm
func TestTokenBucket_Precision(t *testing.T) {
	t.Run("precise refill calculation", func(t *testing.T) {
		// 10 tokens per second = 1 token per 100ms
		bucket := NewTokenBucket(10, time.Second)

		// Drain all
		for i := 0; i < 10; i++ {
			bucket.TryConsume()
		}

		// Wait exactly 500ms - should have 5 tokens
		time.Sleep(500 * time.Millisecond)

		consumed := 0
		for bucket.TryConsume() {
			consumed++
		}

		// Allow some tolerance for timing issues
		assert.GreaterOrEqual(t, consumed, 4, "should have at least 4 tokens")
		assert.LessOrEqual(t, consumed, 6, "should have at most 6 tokens")
	})
}
