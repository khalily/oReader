package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"oreader/internal/infra/ratelimit"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockRateLimiter is a mock implementation for testing
type mockRateLimiter struct {
	allowResponse ratelimit.Result
	keyCaptured   string
}

func (m *mockRateLimiter) Allow(key string, limit int, window time.Duration) ratelimit.Result {
	m.keyCaptured = key
	return m.allowResponse
}

// TestRateLimitMiddleware_BasicBehavior tests basic middleware functionality
func TestRateLimitMiddleware_BasicBehavior(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes: map[string]RouteLimit{
				"/api/v1/test": {Requests: 100, Window: time.Minute},
			},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "100")
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: false, Limit: 5, Remaining: 0, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes: map[string]RouteLimit{
				"/api/v1/test": {Requests: 5, Window: time.Minute},
			},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
	})

	t.Run("uses default limit for unconfigured routes", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 60, Remaining: 59, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/unconfigured", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/unconfigured", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, limiter.keyCaptured)
	})
}

// TestRateLimitMiddleware_Headers tests rate limit headers are set correctly
func TestRateLimitMiddleware_Headers(t *testing.T) {
	t.Run("sets all rate limit headers on allowed request", func(t *testing.T) {
		resetTime := time.Now().Add(time.Minute)
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 95, ResetAt: resetTime},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes: map[string]RouteLimit{
				"/api/v1/test": {Requests: 100, Window: time.Minute},
			},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "95", w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("sets rate limit headers on blocked request", func(t *testing.T) {
		resetTime := time.Now().Add(time.Minute)
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: false, Limit: 5, Remaining: 0, ResetAt: resetTime},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes: map[string]RouteLimit{
				"/api/v1/test": {Requests: 5, Window: time.Minute},
			},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("sets Retry-After header with seconds until reset", func(t *testing.T) {
		resetTime := time.Now().Add(30 * time.Second)
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: false, Limit: 5, Remaining: 0, ResetAt: resetTime},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes: map[string]RouteLimit{
				"/api/v1/test": {Requests: 5, Window: time.Minute},
			},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		retryAfter := w.Header().Get("Retry-After")
		assert.NotEmpty(t, retryAfter)
		// Should be around 30 seconds
		assert.GreaterOrEqual(t, retryAfter, "25")
		assert.LessOrEqual(t, retryAfter, "35")
	})
}

// TestRateLimitMiddleware_KeyExtraction tests the key extraction logic
func TestRateLimitMiddleware_KeyExtraction(t *testing.T) {
	t.Run("uses IP address for unauthenticated requests", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, limiter.keyCaptured, "192.168.1.100")
	})

	t.Run("uses user_id for authenticated requests", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(func(c *gin.Context) {
			// Simulate auth middleware setting user_id
			c.Set("user_id", "user-uuid-123")
			c.Next()
		})
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, limiter.keyCaptured, "user:user-uuid-123")
	})

	t.Run("combines IP and user_id for authenticated requests", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-uuid-456")
			c.Next()
		})
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "10.0.0.5:9000"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should include both user_id and IP
		assert.Contains(t, limiter.keyCaptured, "user:user-uuid-456")
		assert.Contains(t, limiter.keyCaptured, "10.0.0.5")
	})

	t.Run("handles X-Forwarded-For header", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, limiter.keyCaptured, "203.0.113.1")
	})

	t.Run("handles X-Real-IP header", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: true, Limit: 100, Remaining: 99, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set("X-Real-IP", "198.51.100.1")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, limiter.keyCaptured, "198.51.100.1")
	})
}

// TestRateLimitMiddleware_ErrorFormat tests the error response format
func TestRateLimitMiddleware_ErrorFormat(t *testing.T) {
	t.Run("returns correct error structure", func(t *testing.T) {
		limiter := &mockRateLimiter{
			allowResponse: ratelimit.Result{Allowed: false, Limit: 5, Remaining: 0, ResetAt: time.Now().Add(time.Minute)},
		}

		router := gin.New()
		router.Use(RateLimitMiddleware(limiter, RouteLimits{
			Routes:  map[string]RouteLimit{},
			Default: RouteLimit{Requests: 60, Window: time.Minute},
		}))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "error")
		assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
		assert.Contains(t, w.Body.String(), "code")
		assert.Contains(t, w.Body.String(), "message")
	})
}

// TestGetClientIP tests the IP extraction logic
func TestGetClientIP(t *testing.T) {
	t.Run("extracts IP from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:8080"

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		ip := getClientIP(c)
		assert.Equal(t, "192.168.1.100", ip)
	})

	t.Run("prioritizes X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		ip := getClientIP(c)
		assert.Equal(t, "203.0.113.1", ip)
	})

	t.Run("prioritizes X-Real-IP over X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		req.Header.Set("X-Real-IP", "198.51.100.1")

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		ip := getClientIP(c)
		assert.Equal(t, "198.51.100.1", ip)
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "[2001:db8::1]:8080"

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		ip := getClientIP(c)
		assert.Equal(t, "2001:db8::1", ip)
	})

	t.Run("handles X-Forwarded-For with multiple IPs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		ip := getClientIP(c)
		assert.Equal(t, "203.0.113.1", ip) // Should use first IP
	})
}
