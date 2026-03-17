# User Authentication Specification

## ADDED Requirements

### Requirement: User registration with email and password
The system SHALL allow new users to register with email and password. Passwords MUST be hashed using bcrypt before storage.

#### Scenario: Successful registration
- **WHEN** user submits valid email and password to POST /api/v1/auth/register
- **THEN** system creates a new user with UUID identifier
- **AND** system returns user profile without password hash

#### Scenario: Duplicate email registration
- **WHEN** user submits an email that already exists
- **THEN** system returns 409 Conflict error
- **AND** system does not reveal whether email exists

#### Scenario: Invalid email format
- **WHEN** user submits invalid email format
- **THEN** system returns 400 Bad Request with validation error

### Requirement: User login with dual-token authentication
The system SHALL authenticate users with email and password, issuing dual tokens (access and refresh) stored in HttpOnly cookies.

#### Scenario: Successful login
- **WHEN** user submits correct email and password to POST /api/v1/auth/login
- **THEN** system sets access_token cookie (HttpOnly, SameSite=Strict, 15 minutes)
- **AND** system sets refresh_token cookie (HttpOnly, SameSite=Strict, 7 days, Path=/api/v1/auth/refresh)
- **AND** system returns user profile in response body

#### Scenario: Invalid credentials
- **WHEN** user submits incorrect email or password
- **THEN** system returns 401 Unauthorized
- **AND** system does not reveal which field is incorrect

### Requirement: Token refresh mechanism
The system SHALL allow clients to refresh access tokens using valid refresh tokens.

#### Scenario: Successful token refresh
- **WHEN** client calls POST /api/v1/auth/refresh with valid refresh_token cookie
- **THEN** system validates refresh token against database
- **AND** system issues new access_token via Set-Cookie
- **AND** system returns user profile

#### Scenario: Expired or revoked refresh token
- **WHEN** client calls POST /api/v1/auth/refresh with invalid/expired/revoked refresh token
- **THEN** system returns 401 Unauthorized
- **AND** system clears both token cookies

### Requirement: User logout with token revocation
The system SHALL allow users to logout, revoking their refresh token.

#### Scenario: Successful logout
- **WHEN** authenticated user calls POST /api/v1/auth/logout
- **THEN** system marks refresh token as revoked in database
- **AND** system clears both token cookies (Max-Age=0)

### Requirement: Protected API authentication
The system SHALL require valid access token for protected endpoints.

#### Scenario: Valid access token
- **WHEN** client accesses protected endpoint with valid access_token cookie
- **THEN** system validates JWT signature and expiration
- **AND** system injects user_id into request context
- **AND** request proceeds to handler

#### Scenario: Expired access token
- **WHEN** client accesses protected endpoint with expired access_token cookie
- **THEN** system returns 401 Unauthorized
- **AND** response body indicates token expired for client refresh logic

#### Scenario: Missing access token
- **WHEN** client accesses protected endpoint without access_token cookie
- **THEN** system returns 401 Unauthorized

### Requirement: Get current user profile
The system SHALL allow authenticated users to retrieve their profile.

#### Scenario: Get profile
- **WHEN** authenticated user calls GET /api/v1/auth/me
- **THEN** system returns user profile (id, email, nickname, avatar_url, auth_provider)

### Requirement: User identifier security
The system SHALL use UUID v7 for user identifiers to prevent enumeration attacks.

#### Scenario: User ID format
- **WHEN** user is created
- **THEN** user ID is a UUID v7 string
- **AND** user ID is not sequential or predictable
