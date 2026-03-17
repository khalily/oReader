# API Error Format Specification

## ADDED Requirements

### Requirement: Consistent error response structure
The system SHALL return all API errors in a consistent JSON structure.

#### Scenario: Error response format
- **WHEN** API returns an error
- **THEN** response body follows this structure:
  ```json
  {
    "error": {
      "code": "ERROR_CODE",
      "message": "Human readable message",
      "details": {}
    }
  }
  ```
- **AND** HTTP status code matches error type

### Requirement: Standard error codes
The system SHALL use standard error codes for common error types.

#### Scenario: Error code list
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Input validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `TOKEN_EXPIRED` | 401 | Access token expired |
| `FORBIDDEN` | 403 | Permission denied |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g., duplicate) |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

### Requirement: Validation error details
The system SHALL include field-level details for validation errors.

#### Scenario: Field validation error
- **WHEN** input validation fails
- **THEN** error response includes field name and validation message
- **AND** response format:
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Invalid input",
      "details": {
        "field": "email",
        "value": "invalid-email",
        "rule": "email_format"
      }
    }
  }
  ```

#### Scenario: Multiple validation errors
- **WHEN** multiple fields fail validation
- **THEN** details include array of all errors:
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Multiple validation errors",
      "details": {
        "errors": [
          {"field": "email", "rule": "required"},
          {"field": "password", "rule": "min_length", "value": 8}
        ]
      }
    }
  }
  ```

### Requirement: Token expiration handling
The system SHALL distinguish token expiration from other auth errors.

#### Scenario: Access token expired
- **WHEN** access token is expired
- **THEN** error code is `TOKEN_EXPIRED` (not `UNAUTHORIZED`)
- **AND** response:
  ```json
  {
    "error": {
      "code": "TOKEN_EXPIRED",
      "message": "Access token has expired"
    }
  }
  ```
- **AND** frontend uses this to trigger automatic refresh

### Requirement: Rate limit error headers
The system SHALL include rate limit information in 429 responses.

#### Scenario: Rate limit exceeded
- **WHEN** rate limit is exceeded
- **THEN** response includes headers:
  - `X-RateLimit-Limit`: Maximum requests per window
  - `X-RateLimit-Remaining`: 0
  - `X-RateLimit-Reset`: Unix timestamp when limit resets
- **AND** response body:
  ```json
  {
    "error": {
      "code": "RATE_LIMIT_EXCEEDED",
      "message": "Too many requests",
      "details": {
        "retry_after": 60
      }
    }
  }
  ```

### Requirement: Internal error handling
The system SHALL hide internal error details from clients.

#### Scenario: Internal server error
- **WHEN** unexpected error occurs
- **THEN** response uses generic message:
  ```json
  {
    "error": {
      "code": "INTERNAL_ERROR",
      "message": "An unexpected error occurred"
    }
  }
  ```
- **AND** full error details are logged server-side
- **AND** response includes `X-Request-ID` for support lookup
