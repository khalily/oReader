package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorDetail represents a detailed error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Standard error codes
const (
	ErrValidation     = "VALIDATION_ERROR"
	ErrUnauthorized   = "UNAUTHORIZED"
	ErrTokenExpired   = "TOKEN_EXPIRED"
	ErrTokenInvalid   = "TOKEN_INVALID"
	ErrCSRFMissing     = "CSRF_MISSING"
	ErrCSRFMismatch    = "CSRF_MISMATCH"
	ErrForbidden       = "FORBIDDEN"
	ErrNotFound         = "NOT_FOUND"
	ErrConflict         = "CONFLICT"
	ErrRateLimit       = "RATE_LIMIT_EXCEEDED"
	ErrInternal         = "INTERNAL_ERROR"
	ErrBadRequest       = "BAD_REQUEST"
)

// New creates a standardized error response
func New(code string, message string, details map[string]interface{}) ErrorDetail {
	return ErrorDetail{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Send sends a standardized error response
func Send(c *gin.Context, status int, err ErrorDetail) {
	c.JSON(status, gin.H{
		"error": err,
	})
}

// BadRequest sends a 400 Bad Request error
func BadRequest(c *gin.Context, message string, details map[string]interface{}) {
	Send(c, http.StatusBadRequest, New(ErrBadRequest, message, details))
}

// Unauthorized sends a 401 Unauthorized error
func Unauthorized(c *gin.Context, message string, details map[string]interface{}) {
	Send(c, http.StatusUnauthorized, New(ErrUnauthorized, message, details))
}

// Forbidden sends a 403 Forbidden error
func Forbidden(c *gin.Context, message string, details map[string]interface{}) {
	Send(c, http.StatusForbidden, New(ErrForbidden, message, details))
}

// NotFound sends a 404 Not Found error
func NotFound(c *gin.Context, resource string) {
	Send(c, http.StatusNotFound, New(ErrNotFound, resource+" not found", nil))
}

// Conflict sends a 409 Conflict error
func Conflict(c *gin.Context, message string, details map[string]interface{}) {
	Send(c, http.StatusConflict, New(ErrConflict, message, details))
}

// Internal sends a 500 Internal Server Error
func Internal(c *gin.Context, message string, details map[string]interface{}) {
	Send(c, http.StatusInternalServerError, New(ErrInternal, message, details))
}

// RateLimit sends a 429 Too Many Requests error
func RateLimit(c *gin.Context, details map[string]interface{}) {
	Send(c, http.StatusTooManyRequests, New(ErrRateLimit, "Rate limit exceeded", details))
}
