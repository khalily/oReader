# E2E API Testing Specification

## ADDED Requirements

### Requirement: Playwright test infrastructure
The system SHALL provide Playwright-based end-to-end API testing infrastructure with headless browser support for CI/CD compatibility.

#### Scenario: Test environment setup
- **WHEN** tests are executed
- **THEN** system uses separate test database (oreader_test.db)
- **AND** system disables rate limiting (RATE_LIMIT_ENABLED=false)
- **AND** system runs in headless Chromium mode

### Requirement: Authentication test fixtures
The system SHALL provide reusable authentication fixtures that handle login, CSRF token extraction, and authenticated request setup.

#### Scenario: Authenticated test context
- **WHEN** a test requires authentication
- **THEN** fixture registers/logs in a test user
- **AND** fixture extracts csrf_token from response
- **AND** fixture returns context with cookies and CSRF token for subsequent requests

### Requirement: Authentication API test coverage
The system SHALL provide comprehensive tests for all authentication endpoints.

#### Scenario: Registration test
- **WHEN** POST /api/v1/auth/register with valid email and password
- **THEN** response is 201 Created with user object and csrf_token
- **AND** cookies include access_token and csrf_token

#### Scenario: Login test
- **WHEN** POST /api/v1/auth/login with valid credentials
- **THEN** response is 200 OK with user object and csrf_token
- **AND** cookies include access_token and csrf_token

#### Scenario: Get current user test
- **WHEN** GET /api/v1/auth/me with valid access_token cookie
- **THEN** response is 200 OK with user profile

#### Scenario: Token refresh test
- **WHEN** POST /api/v1/auth/refresh with valid refresh_token cookie
- **THEN** response is 200 OK with user profile
- **AND** new access_token cookie is set

#### Scenario: Logout test
- **WHEN** POST /api/v1/auth/logout
- **THEN** response is 200 OK with success message
- **AND** all auth cookies are cleared

### Requirement: Feed API test coverage
The system SHALL provide comprehensive tests for all feed management endpoints.

#### Scenario: Create feed subscription
- **WHEN** POST /api/v1/feeds with valid feed_url and X-CSRF-Token header
- **THEN** response is 201 Created with feed object and new_item_count

#### Scenario: List feeds
- **WHEN** GET /api/v1/feeds with valid authentication
- **THEN** response is 200 OK with feeds array and total count

#### Scenario: Get single feed
- **WHEN** GET /api/v1/feeds/:id with valid authentication
- **THEN** response is 200 OK with feed object and item_count

#### Scenario: Delete feed
- **WHEN** DELETE /api/v1/feeds/:id with X-CSRF-Token header
- **THEN** response is 204 No Content

#### Scenario: Refresh feed
- **WHEN** POST /api/v1/feeds/:id/refresh with X-CSRF-Token header
- **THEN** response is 200 OK with refresh result

#### Scenario: Mark all items read
- **WHEN** POST /api/v1/feeds/:id/mark-all-read with X-CSRF-Token header
- **THEN** response is 200 OK with count of items marked read

### Requirement: Item API test coverage
The System SHALL provide comprehensive tests for all item management endpoints.

#### Scenario: List items with pagination
- **WHEN** GET /api/v1/items with valid authentication
- **THEN** response is 200 OK with items array, total, has_more, and next_cursor

#### Scenario: Get single item
- **WHEN** GET /api/v1/items/:id with valid authentication
- **THEN** response is 200 OK with item object

#### Scenario: Toggle star status
- **WHEN** POST /api/v1/items/:id/star with X-CSRF-Token header and {starred: true}
- **THEN** response is 200 OK with updated item object

#### Scenario: Toggle read status
- **WHEN** POST /api/v1/items/:id/read with X-CSRF-Token header and {read: true}
- **THEN** response is 200 OK with updated item object

### Requirement: OPML API test coverage
The system SHALL provide comprehensive tests for OPML import/export endpoints.

#### Scenario: Export feeds as OPML
- **WHEN** GET /api/v1/opml/export with valid authentication
- **THEN** response is 200 OK with Content-Type: application/xml
- **AND** response body is valid OPML document

#### Scenario: Import OPML file
- **WHEN** POST /api/v1/opml/import with X-CSRF-Token header and valid OPML file
- **THEN** response is 202 Accepted with job_id and total_feeds

#### Scenario: Get import job status
- **WHEN** GET /api/v1/opml/import/:job_id with valid authentication
- **THEN** response is 200 OK with job status, progress, and counts

### Requirement: Error case test coverage
The system SHALL provide tests for error scenarios including authentication failures, validation errors, and not found cases.

#### Scenario: Unauthorized access
- **WHEN** accessing protected endpoint without valid access_token
- **THEN** response is 401 Unauthorized

#### Scenario: Missing CSRF token
- **WHEN** POST/PUT/DELETE to protected endpoint without X-CSRF-Token header
- **THEN** response is 403 Forbidden with CSRF error

#### Scenario: Resource not found
- **WHEN** requesting non-existent feed or item
- **THEN** response is 404 Not Found

### Requirement: Test database isolation
The system SHALL ensure complete test isolation through separate database and cleanup between runs.

#### Scenario: Fresh database per run
- **WHEN** test suite starts
- **THEN** test database is created fresh with all migrations applied
- **AND** no data from previous runs persists
