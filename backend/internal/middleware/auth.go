package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/khalily/oreader/internal/infra/cookie"
	"github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/jwt"
)

// AuthMiddleware validates JWT tokens for authentication
func AuthMiddleware(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get access token from cookie
		accessToken := cookie.GetAccessToken(c.Request)
		if accessToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": errors.NewSimple(errors.ErrUnauthorized, "Access token required"),
			})
			return
		}

		// Validate access token
		claims, err := jwtService.ValidateAccessToken(accessToken)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": errors.NewSimple(errors.ErrTokenExpired, "Access token has expired"),
				})
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": errors.NewSimple(errors.ErrTokenInvalid, "Invalid access token"),
				})
			}
			return
		}

		// Store user ID in context
		c.Set("user_id", claims.UserID)
		c.Set("csrf_token", claims.CSRFToken)

		c.Next()
	}
}

// CSRFMiddleware validates CSRF tokens for state-changing requests
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check CSRF for state-changing methods
		method := strings.ToUpper(c.Request.Method)
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			c.Next()
			return
		}

		// Get CSRF token from header
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": errors.NewSimple(errors.ErrCSRFMissing, "CSRF token required in X-CSRF-Token header"),
			})
			return
		}

		// Get CSRF token from cookie (set by auth middleware)
		cookieToken, exists := c.Get("csrf_token")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": errors.NewSimple(errors.ErrCSRFMissing, "CSRF token not found in session - please re-login"),
			})
			return
		}

		// Validate CSRF token
		cookieTokenStr, ok := cookieToken.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": errors.NewSimple(errors.ErrCSRFMismatch, "Invalid CSRF token format in session"),
			})
			return
		}

		if headerToken != cookieTokenStr {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": errors.NewSimple(errors.ErrCSRFMismatch, "CSRF token mismatch"),
			})
			return
		}

		c.Next()
	}
}

// OptionalAuthMiddleware extracts user info if present but doesn't require it
func OptionalAuthMiddleware(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := cookie.GetAccessToken(c.Request)
		if accessToken == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateAccessToken(accessToken)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("csrf_token", claims.CSRFToken)
		c.Next()
	}
}
