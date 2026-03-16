package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error represents a standardized error response
type Error struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e Error) Error() string {
	return e.Message
}

// Standard error codes
const (
	ErrValidation    = "VALIDATION_ERROR"
	ErrUnauthorized  = "UNAUTHORIZED"
	ErrTokenExpired  = "TOKEN_EXPIRED"
	ErrTokenInvalid  = "TOKEN_INVALID"
	ErrForbidden     = "FORBIDDEN"
	ErrNotFound      = "NOT_FOUND"
	ErrConflict      = "CONFLICT"
	ErrRateLimit     = "RATE_LIMIT_EXCEEDED"
	ErrInternal      = "INTERNAL_ERROR"
	ErrCSRFMissing   = "CSRF_MISSING"
	ErrCSRFMismatch  = "CSRF_MISMATCH"
)

// Send sends an error response
func Send(c *gin.Context, status int, err Error) {
	c.JSON(status, gin.H{
		"error": err,
	})
}

// New creates a new Error
func New(code string, message string, details map[string]interface{}) Error {
	return Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewSimple creates a new Error without details
func NewSimple(code string, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}

// Is checks if err is of a specific type
func Is(err Error, target string) bool {
	return err.Code == target
}

// SendBadRequest sends a 400 Bad Request error
func SendBadRequest(c *gin.Context, err Error) {
	Send(c, http.StatusBadRequest, err)
}

// SendUnauthorized sends a 401 Unauthorized error
func SendUnauthorized(c *gin.Context, err Error) {
	Send(c, http.StatusUnauthorized, err)
}

// SendForbidden sends a 403 Forbidden error
func SendForbidden(c *gin.Context, err Error) {
	Send(c, http.StatusForbidden, err)
}

// SendNotFound sends a 404 Not Found error
func SendNotFound(c *gin.Context, err Error) {
	Send(c, http.StatusNotFound, err)
}

// SendConflict sends a 409 Conflict error
func SendConflict(c *gin.Context, err Error) {
	Send(c, http.StatusConflict, err)
}

// SendInternal sends a 500 Internal Server Error
func SendInternal(c *gin.Context, err Error) {
	Send(c, http.StatusInternalServerError, err)
}

// SendRateLimited sends a 429 Too Many Requests error
func SendRateLimited(c *gin.Context, err Error) {
	Send(c, http.StatusTooManyRequests, err)
}
