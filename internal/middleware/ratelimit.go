package middleware

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"oreader/internal/infra/errors"
	"oreader/internal/infra/ratelimit"
)

// RouteLimit defines the rate limit for a specific route
type RouteLimit struct {
	Requests int
	Window   time.Duration
}

// RouteLimits maps route paths to their rate limits
type RouteLimits struct {
	// Per-route limits (key is the route path)
	Routes map[string]RouteLimit
	// Default limit for routes without specific configuration
	Default RouteLimit
}

// RateLimitMiddleware creates a rate limiting middleware
// It uses IP address for unauthenticated requests and user_id + IP for authenticated requests
func RateLimitMiddleware(limiter ratelimit.RateLimiter, limits RouteLimits) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the full path for matching
		path := c.FullPath()

		// Find the appropriate limit for this route
		var limit RouteLimit
		if routeLimit, exists := limits.Routes[path]; exists {
			limit = routeLimit
		} else {
			limit = limits.Default
		}

		// Build the rate limit key
		key := buildRateLimitKey(c)

		// Add the path to make limits per-endpoint
		key = fmt.Sprintf("%s:%s", key, path)

		// Check the rate limit
		result := limiter.Allow(key, limit.Requests, limit.Window)

		// Set rate limit headers
		setRateLimitHeaders(c, result)

		// Block if rate limited
		if !result.Allowed {
			retryAfter := calculateRetryAfter(result.ResetAt)
			c.Header("Retry-After", strconv.Itoa(int(retryAfter)))

			errors.SendRateLimited(c, errors.NewSimple(errors.ErrRateLimit,
				fmt.Sprintf("Rate limit exceeded. Try again in %d seconds", retryAfter)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// buildRateLimitKey builds a unique key for rate limiting
// Uses user_id for authenticated requests, IP for unauthenticated
func buildRateLimitKey(c *gin.Context) string {
	// Check for authenticated user
	if userID, exists := c.Get("user_id"); exists {
		// For authenticated users, combine user_id with IP
		// This prevents a single user from bypassing limits via multiple IPs
		ip := getClientIP(c)
		return fmt.Sprintf("user:%s:%s", userID, ip)
	}

	// For unauthenticated requests, use IP only
	ip := getClientIP(c)
	return fmt.Sprintf("ip:%s", ip)
}

// getClientIP extracts the client IP from the request
// Checks X-Real-IP, X-Forwarded-For, and RemoteAddr in order
func getClientIP(c *gin.Context) string {
	// Check X-Real-IP header (highest priority - set by trusted reverse proxy)
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Check X-Forwarded-For header
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// The first one is the original client
		if idx := strings.Index(forwardedFor, ","); idx != -1 {
			return strings.TrimSpace(forwardedFor[:idx])
		}
		return forwardedFor
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}

	return c.Request.RemoteAddr
}

// setRateLimitHeaders sets the rate limit response headers
func setRateLimitHeaders(c *gin.Context, result ratelimit.Result) {
	c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
}

// calculateRetryAfter calculates seconds until the rate limit resets
func calculateRetryAfter(resetTime time.Time) int64 {
	untilReset := time.Until(resetTime)
	if untilReset < 0 {
		return 0
	}
	return int64(untilReset.Seconds()) + 1
}
