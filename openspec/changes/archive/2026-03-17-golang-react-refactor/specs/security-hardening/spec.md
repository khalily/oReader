# Security Hardening Specification

## ADDED Requirements

### Requirement: Security headers on all responses
The system SHALL include security headers on all HTTP responses.

#### Scenario: Response headers
- **WHEN** any HTTP response is sent
- **THEN** response includes:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Content-Security-Policy: default-src 'self'; img-src * data:; style-src 'self' 'unsafe-inline'`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy: geolocation=(), microphone=(), camera=()`

### Requirement: CSRF token protection
The system SHALL validate CSRF tokens on all state-changing requests.

#### Scenario: CSRF token generation
- **WHEN** user logs in successfully
- **THEN** server generates CSRF token
- **AND** sets CSRF token in cookie (HttpOnly=false, readable by JS)
- **AND** returns CSRF token in response body

#### Scenario: CSRF token validation
- **WHEN** client makes POST/PUT/DELETE/PATCH request
- **THEN** client includes `X-CSRF-Token` header
- **AND** server validates header matches cookie value
- **AND** mismatch returns 403 with `CSRF_TOKEN_MISMATCH` error

#### Scenario: CSRF token rotation
- **WHEN** access token is refreshed
- **THEN** new CSRF token is generated and returned

### Requirement: RSS content sanitization
The system SHALL sanitize all HTML content from RSS feeds.

#### Scenario: Article content sanitization
- **WHEN** RSS feed item contains HTML content
- **THEN** content is sanitized using UGC policy before storage or display
- **AND** allowed tags: p, br, a, img, ul, ol, li, blockquote, pre, code, strong, em, h1-h6
- **AND** all script tags, onclick attributes, and javascript: URLs are removed

#### Scenario: Feed metadata sanitization
- **WHEN** feed title or description is parsed
- **THEN** content is sanitized using strict policy (text only, no HTML)

### Requirement: Feed URL SSRF protection
The system SHALL validate and restrict feed URL fetching.

#### Scenario: URL scheme validation
- **WHEN** user adds new feed URL
- **THEN** only http:// and https:// schemes are allowed
- **AND** other schemes return 400 error

#### Scenario: Private IP blocking
- **WHEN** feed URL is resolved
- **THEN** system blocks IPs in these ranges:
  - 10.0.0.0/8 (Private)
  - 172.16.0.0/12 (Private)
  - 192.168.0.0/16 (Private)
  - 127.0.0.0/8 (Loopback)
  - 169.254.0.0/16 (Link-local/AWS metadata)
  - ::1/128 (IPv6 loopback)
  - fc00::/7 (IPv6 private)
- **AND** blocked URLs return 400 with `URL_NOT_ALLOWED` error

### Requirement: Feed size limits
The system SHALL enforce size limits on feed content.

#### Scenario: Feed fetch timeout
- **WHEN** fetching feed URL
- **THEN** request times out after 30 seconds
- **AND** timeout is logged as feed fetch failure

#### Scenario: Item count limit
- **WHEN** parsing RSS feed
- **THEN** only first 1000 items are processed
- **AND** excess items are logged but not stored

#### Scenario: Content size limit
- **WHEN** individual item content exceeds 1MB
- **THEN** content is truncated with marker `[truncated]`
- **AND** original size is logged

### Requirement: Password security
The system SHALL enforce secure password handling.

#### Scenario: Password complexity
- **WHEN** user registers or changes password
- **THEN** password must be at least 8 characters
- **AND** bcrypt cost factor is 12 or higher

#### Scenario: Password hashing
- **WHEN** password is stored
- **THEN** password is hashed with bcrypt
- **AND** plaintext password is never logged

### Requirement: JWT signing security
The system SHALL use secure JWT signing configuration.

#### Scenario: Signing algorithm
- **WHEN** JWT is generated
- **THEN** HS256 algorithm is used (or RS256 for production)
- **AND** secret key is at least 256 bits (32 bytes)

#### Scenario: Token claims
- **WHEN** access token is generated
- **THEN** claims include: sub (user_id), exp, iat, csrf_token
- **AND** claims do NOT include sensitive data (password, email)

### Requirement: Rate limiting key combination
The system SHALL use combined keys for authenticated rate limiting.

#### Scenario: Unauthenticated rate limiting
- **WHEN** request has no valid access token
- **THEN** rate limit key is client IP address

#### Scenario: Authenticated rate limiting
- **WHEN** request has valid access token
- **THEN** rate limit key is combination of IP + user_id
- **AND** this prevents single compromised account from using full quota

### Requirement: Dependency vulnerability scanning
The system SHALL detect vulnerable dependencies.

#### Scenario: CI vulnerability check
- **WHEN** CI pipeline runs
- **THEN** `govulncheck` scans Go dependencies
- **AND** `npm audit` scans frontend dependencies
- **AND** pipeline fails on high/critical vulnerabilities
