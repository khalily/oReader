# Rate Limiting Specification

## ADDED Requirements

### Requirement: API rate limiting for abuse prevention
The system SHALL enforce rate limits on API endpoints to prevent abuse and protect server resources.

#### Scenario: Request within rate limit
- **WHEN** client makes requests within configured rate limit
- **THEN** system processes requests normally
- **AND** response includes rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)

#### Scenario: Request exceeds rate limit
- **WHEN** client exceeds rate limit for an endpoint
- **THEN** system returns 429 Too Many Requests
- **AND** response includes Retry-After header
- **AND** response body includes error message and reset time

### Requirement: Differentiated rate limits by endpoint type
The system SHALL apply different rate limits based on endpoint sensitivity.

#### Scenario: Auth endpoint rate limit (strict)
- **WHEN** client calls /api/v1/auth/login or /api/v1/auth/register
- **THEN** system allows maximum 10 requests per minute per IP
- **AND** system returns 429 when exceeded

#### Scenario: API endpoint rate limit (standard)
- **WHEN** client calls other authenticated API endpoints
- **THEN** system allows maximum 60 requests per minute per user
- **AND** unauthenticated requests limited to 30 requests per minute per IP

#### Scenario: Feed refresh rate limit
- **WHEN** client calls POST /api/v1/feeds/:id/refresh
- **THEN** system allows maximum 6 requests per minute per feed
- **AND** system prevents refresh spam for same feed

### Requirement: Rate limit key identification
The system SHALL identify clients using appropriate keys for rate limiting.

#### Scenario: Authenticated user
- **WHEN** request includes valid authentication
- **THEN** system uses user_id as rate limit key

#### Scenario: Unauthenticated request
- **WHEN** request has no authentication
- **THEN** system uses client IP address as rate limit key
- **AND** system handles X-Forwarded-For header for reverse proxy deployments

### Requirement: Rate limit response headers
The system SHALL include rate limit information in response headers.

#### Scenario: Headers included in all responses
- **WHEN** client makes any API request
- **THEN** response includes:
  - X-RateLimit-Limit: Maximum requests per window
  - X-RateLimit-Remaining: Requests remaining in current window
  - X-RateLimit-Reset: Unix timestamp when window resets

### Requirement: Sliding window rate limiting
The system SHALL use sliding window algorithm for smoother rate limiting.

#### Scenario: Request near window boundary
- **WHEN** client makes requests near the end of rate limit window
- **THEN** system counts requests in rolling window
- **AND** system does not allow burst at window boundary

### Requirement: Rate limit bypass for health checks
The system SHALL exempt health check endpoints from rate limiting.

#### Scenario: Health check not rate limited
- **WHEN** client calls GET /health or GET /api/v1/health
- **THEN** system does not apply rate limiting
- **AND** system does not include rate limit headers
