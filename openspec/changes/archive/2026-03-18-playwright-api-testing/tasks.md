## 1. Infrastructure Setup

- [x] 1.1 Install Playwright and dependencies in web/ directory
- [x] 1.2 Create tests/e2e/ directory structure
- [x] 1.3 Create playwright.config.ts with base URL, headless mode, and test database settings
- [x] 1.4 Create .env.test with test environment configuration (DATABASE_URL=oreader_test.db, RATE_LIMIT_ENABLED=false)
- [x] 1.5 Add npm scripts for running E2E tests (test:e2e, test:e2e:ui)

## 2. Test Fixtures and Helpers

- [x] 2.1 Create tests/e2e/fixtures/auth.ts with authentication helpers
- [x] 2.2 Implement login helper that extracts csrf_token from response
- [x] 2.3 Implement authenticated request helper with automatic CSRF header injection
- [x] 2.4 Create test user factory for generating unique test users

## 3. Bug Fix - Login Response

- [x] 3.1 Fix Login endpoint to include csrf_token in response body (internal/handler/auth.go)
- [x] 3.2 Verify Login response matches Register response structure
- [x] 3.3 Test the fix manually or with unit test

## 4. Authentication API Tests

- [x] 4.1 Create tests/e2e/auth.spec.ts
- [x] 4.2 Test: POST /api/v1/auth/register - successful registration
- [x] 4.3 Test: POST /api/v1/auth/register - duplicate email error
- [x] 4.4 Test: POST /api/v1/auth/register - invalid email format
- [x] 4.5 Test: POST /api/v1/auth/login - successful login with csrf_token in response
- [x] 4.6 Test: POST /api/v1/auth/login - invalid credentials
- [x] 4.7 Test: GET /api/v1/auth/me - get current user
- [x] 4.8 Test: POST /api/v1/auth/refresh - token refresh
- [x] 4.9 Test: POST /api/v1/auth/logout - logout clears cookies

## 5. Feed API Tests

- [x] 5.1 Create tests/e2e/feeds.spec.ts
- [x] 5.2 Test: POST /api/v1/feeds - create feed subscription (use OpenAI RSS)
- [x] 5.3 Test: GET /api/v1/feeds - list feeds
- [x] 5.4 Test: GET /api/v1/feeds/:id - get single feed
- [x] 5.5 Test: POST /api/v1/feeds/:id/refresh - refresh feed
- [x] 5.6 Test: POST /api/v1/feeds/:id/mark-all-read - mark all items read
- [x] 5.7 Test: DELETE /api/v1/feeds/:id - delete feed
- [x] 5.8 Test: Error cases - invalid URL, duplicate feed, not found

## 6. Item API Tests

- [x] 6.1 Create tests/e2e/items.spec.ts
- [x] 6.2 Test: GET /api/v1/items - list items with pagination
- [x] 6.3 Test: GET /api/v1/items?feed_id=X - filter by feed
- [x] 6.4 Test: GET /api/v1/items?starred=true - filter by starred
- [x] 6.5 Test: GET /api/v1/items/:id - get single item
- [x] 6.6 Test: POST /api/v1/items/:id/star - toggle star status
- [x] 6.7 Test: POST /api/v1/items/:id/read - toggle read status
- [x] 6.8 Test: Error cases - item not found, unauthorized access

## 7. OPML API Tests

- [x] 7.1 Create tests/e2e/opml.spec.ts
- [x] 7.2 Test: GET /api/v1/opml/export - export feeds as OPML
- [x] 7.3 Test: POST /api/v1/opml/import - import OPML file
- [x] 7.4 Test: GET /api/v1/opml/import/:job_id - get import status

## 8. Error Handling Tests

- [x] 8.1 Test: 401 Unauthorized - missing access token
- [x] 8.2 Test: 403 Forbidden - missing CSRF token on POST
- [x] 8.3 Test: 403 Forbidden - CSRF token mismatch
- [x] 8.4 Test: 404 Not Found - non-existent resources
- [x] 8.5 Test: Rate limiting behavior (optional, if testing with rate limits enabled)

## 9. Test-Fix Cycle

- [x] 9.1 Run all tests and capture failures
- [x] 9.2 Fix each discovered bug
- [x] 9.3 Re-run tests until all pass
- [x] 9.4 Document any additional bugs found and fixed

## 10. Documentation and Cleanup

- [x] 10.1 Add README.md to tests/e2e/ with setup and run instructions
- [x] 10.2 Update main README.md with E2E testing section
- [x] 10.3 Clean up any test artifacts
- [x] 10.4 Verify tests run in CI environment (if applicable)
