package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// RequestLoggerMiddleware logs request details
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate request ID
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("request_id", requestID)
		}

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		logger := log.Info()
		if requestID != "" {
			logger = logger.Str("request_id", requestID)
		}
		logger.Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
		Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Msg("Request completed")
	}
}
