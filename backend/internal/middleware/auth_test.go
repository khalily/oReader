package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khalily/oreader/internal/infra/jwt"
	"github.com/khalily/oreader/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestJWTService(t *testing.T) *jwt.Service {
	secretKey := testutil.GetTestJWTSecret()
	return jwt.NewService(secretKey, 15*time.Minute, 7*24*time.Hour)
}

func TestAuthMiddleware(t *testing.T) {
	jwtService := setupTestJWTService(t)

	t.Run("allows request with valid token", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			assert.True(t, exists)
			assert.NotEmpty(t, userID)
			c.JSON(http.StatusOK, gin.H{"user_id": userID})
		})

		accessToken, _, err := jwtService.GenerateAccessToken("user-123")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects request without token", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with invalid token", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-token"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects request with expired token", func(t *testing.T) {
		// Create a service with very short TTL
		shortService := jwt.NewService(testutil.GetTestJWTSecret(), 10*time.Millisecond, 7*24*time.Hour)
		router := gin.New()
		router.Use(AuthMiddleware(shortService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		accessToken, _, err := shortService.GenerateAccessToken("user-123")
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(50 * time.Millisecond)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("sets user_id and csrf_token in context", func(t *testing.T) {
		router := gin.New()
		var capturedUserID, capturedCSRF string
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			csrfToken, _ := c.Get("csrf_token")
			capturedUserID = userID.(string)
			capturedCSRF = csrfToken.(string)
			c.JSON(http.StatusOK, gin.H{})
		})

		accessToken, csrfToken, err := jwtService.GenerateAccessToken("user-456")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user-456", capturedUserID)
		assert.Equal(t, csrfToken, capturedCSRF)
	})
}

func TestCSRFMiddleware(t *testing.T) {
	t.Run("allows GET requests without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.GET("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("GET", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("allows HEAD requests without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.HEAD("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("HEAD", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("allows OPTIONS requests without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.OPTIONS("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("OPTIONS", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects POST request without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.POST("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("POST", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects PUT request without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.PUT("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("PUT", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects DELETE request without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.DELETE("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("DELETE", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects PATCH request without CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.PATCH("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("PATCH", "/resource", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects POST with mismatched CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("csrf_token", "cookie-csrf-token")
			c.Next()
		})
		router.Use(CSRFMiddleware())
		router.POST("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("POST", "/resource", nil)
		req.Header.Set("X-CSRF-Token", "different-csrf-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("allows POST with matching CSRF token", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("csrf_token", "matching-csrf-token")
			c.Next()
		})
		router.Use(CSRFMiddleware())
		router.POST("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("POST", "/resource", nil)
		req.Header.Set("X-CSRF-Token", "matching-csrf-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects request when CSRF token not in context", func(t *testing.T) {
		router := gin.New()
		router.Use(CSRFMiddleware())
		router.POST("/resource", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req, _ := http.NewRequest("POST", "/resource", nil)
		req.Header.Set("X-CSRF-Token", "some-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestOptionalAuthMiddleware(t *testing.T) {
	jwtService := setupTestJWTService(t)

	t.Run("allows request without token", func(t *testing.T) {
		router := gin.New()
		router.Use(OptionalAuthMiddleware(jwtService))
		router.GET("/optional", func(c *gin.Context) {
			_, exists := c.Get("user_id")
			assert.False(t, exists)
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("GET", "/optional", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("extracts user info with valid token", func(t *testing.T) {
		router := gin.New()
		var capturedUserID string
		router.Use(OptionalAuthMiddleware(jwtService))
		router.GET("/optional", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			assert.True(t, exists)
			capturedUserID = userID.(string)
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		accessToken, _, err := jwtService.GenerateAccessToken("user-789")
		require.NoError(t, err)

		req, _ := http.NewRequest("GET", "/optional", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user-789", capturedUserID)
	})

	t.Run("ignores invalid token and continues", func(t *testing.T) {
		router := gin.New()
		var userExists bool
		router.Use(OptionalAuthMiddleware(jwtService))
		router.GET("/optional", func(c *gin.Context) {
			_, userExists = c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req, _ := http.NewRequest("GET", "/optional", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-token"})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, userExists)
	})

	t.Run("ignores expired token and continues", func(t *testing.T) {
		shortService := jwt.NewService(testutil.GetTestJWTSecret(), 10*time.Millisecond, 7*24*time.Hour)
		router := gin.New()
		var userExists bool
		router.Use(OptionalAuthMiddleware(shortService))
		router.GET("/optional", func(c *gin.Context) {
			_, userExists = c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		accessToken, _, err := shortService.GenerateAccessToken("user-123")
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(50 * time.Millisecond)

		req, _ := http.NewRequest("GET", "/optional", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, userExists)
	})
}

func TestAuthAndCSRFMiddlewareChain(t *testing.T) {
	jwtService := setupTestJWTService(t)

	t.Run("full auth flow with CSRF protection", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.Use(CSRFMiddleware())

		router.POST("/protected", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": userID})
		})

		accessToken, csrfToken, err := jwtService.GenerateAccessToken("user-chain")
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		req.Header.Set("X-CSRF-Token", csrfToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects POST without CSRF even with valid auth", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.Use(CSRFMiddleware())

		router.POST("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		accessToken, _, err := jwtService.GenerateAccessToken("user-chain")
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
		// No CSRF header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
