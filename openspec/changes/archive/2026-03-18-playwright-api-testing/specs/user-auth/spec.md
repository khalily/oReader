# User Authentication Delta Specification

## MODIFIED Requirements

### Requirement: User login with dual-token authentication
The system SHALL authenticate users with email and password, issuing dual tokens (access and refresh) stored in HttpOnly cookies, and returning the CSRF token in the response body.

#### Scenario: Successful login
- **WHEN** user submits correct email and password to POST /api/v1/auth/login
- **THEN** system sets access_token cookie (HttpOnly, SameSite=Lax in dev, 15 minutes)
- **AND** system sets csrf_token cookie (readable by JS, 15 minutes)
- **AND** system sets refresh_token cookie (HttpOnly, SameSite=Lax in dev, 7 days, Path=/api/v1/auth/refresh)
- **AND** system returns user profile in response body
- **AND** system returns csrf_token in response body for client use

#### Scenario: Invalid credentials
- **WHEN** user submits incorrect email or password
- **THEN** system returns 401 Unauthorized
- **AND** system does not reveal which field is incorrect
