## Why

The oReader API currently lacks automated end-to-end testing, making it difficult to catch regressions and validate that all endpoints work correctly together. We need comprehensive Playwright-based API tests that run in headless mode to ensure API reliability and catch bugs before deployment.

## What Changes

- Add Playwright test infrastructure in the `tests/e2e/` directory
- Create comprehensive API test suites for all endpoints:
  - Authentication APIs (register, login, refresh, logout, me)
  - Feed management APIs (CRUD, refresh, mark-all-read)
  - Item management APIs (list, get, star, read)
  - OPML import/export APIs
- Implement test database isolation using separate SQLite file
- Add test environment configuration with rate limiting bypass
- Fix any bugs discovered during testing (test-fix cycle)

## Capabilities

### New Capabilities

- `e2e-api-testing`: Comprehensive Playwright-based end-to-end API testing infrastructure covering all oReader REST endpoints with proper authentication, CSRF handling, and test database isolation

### Modified Capabilities

- `user-auth`: The login endpoint response will be modified to include `csrf_token` in the response body (currently missing, causing inconsistency with register endpoint)

## Impact

- **New Files**:
  - `web/tests/e2e/` - Playwright test directory
  - `web/tests/e2e/playwright.config.ts` - Playwright configuration
  - `web/tests/e2e/fixtures/auth.ts` - Authentication test fixtures
  - `web/tests/e2e/*.spec.ts` - Test suites for each API area
  - `.env.test` - Test environment configuration

- **Modified Files**:
  - `web/package.json` - Add Playwright dependencies
  - `internal/handler/auth.go` - Fix login response to include csrf_token

- **Test Database**: `oreader_test.db` (separate from production)

- **External Dependencies**:
  - OpenAI News RSS (`https://openai.com/news/rss.xml`) for feed testing
  - Playwright browsers (Chromium)
