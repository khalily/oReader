package ratelimit

import "time"

// RateLimiter defines the interface for rate limiting implementations
// Implementations can be in-memory, Redis-based, or use any other backend
type RateLimiter interface {
	// Allow checks if a request should be allowed based on the rate limit
	// key: Unique identifier for the rate limit (e.g., "user:123", "ip:192.168.1.1")
	// limit: Maximum number of requests allowed within the window
	// window: Time duration for the rate limit window
	// Returns a Result with the decision and metadata
	Allow(key string, limit int, window time.Duration) Result
}

// Result contains the result of a rate limit check
type Result struct {
	Allowed   bool      // True if the request is allowed
	Limit     int       // The rate limit configured
	Remaining int       // Number of requests remaining in the window
	ResetAt   time.Time // When the rate limit window resets
}
