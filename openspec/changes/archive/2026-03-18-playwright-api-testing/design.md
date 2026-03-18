## Context

The oReader backend is a Go/Gin REST API with cookie-based authentication and CSRF protection. Currently, only unit tests exist for individual components. We need end-to-end API tests that validate the full request/response cycle including authentication flows, CSRF token handling, and database persistence.

The key challenge is handling the authentication flow correctly:
1. Login/Register sets `access_token` (HttpOnly) and `csrf_token` (readable) cookies
2. CSRF token must be extracted from response body and sent as `X-CSRF-Token` header for POST/PUT/DELETE/PATCH requests
3. Tests need isolated database state to avoid pollution between runs

## Goals / Non-Goals

**Goals:**
- Comprehensive E2E API test coverage for all REST endpoints
- Proper authentication and CSRF token handling in tests
- Isolated test database that resets between test runs
- Automated test-fix cycle to catch and fix bugs
- Headless execution for CI/CD compatibility

**Non-Goals:**
- Frontend UI testing (API-only)
- OAuth/GitHub authentication testing (skipped)
- Performance/load testing
- Visual regression testing

## Decisions

### D1: Playwright for API Testing
**Choice:** Playwright (not Jest/Vitest directly)
**Rationale:**
- Built-in cookie management and automatic cookie handling
- Excellent TypeScript support matching the frontend stack
- Can be extended to UI testing later if needed
- Better async/await handling than traditional test runners

**Alternatives Considered:**
- Vitest + supertest: Lighter weight but requires manual cookie management
- Go testing: Would work but team prefers TypeScript for E2E tests

### D2: Separate Test Database
**Choice:** SQLite file `oreader_test.db` with cleanup between runs
**Rationale:**
- No external database dependency required
- Fast test execution
- Complete isolation from development/production data

**Implementation:**
- Set `DATABASE_URL=oreader_test.db` in test environment
- Delete and recreate database before each test run
- Go auto-migration handles schema creation

### D3: Rate Limiting Bypass
**Choice:** Disable rate limiting via `RATE_LIMIT_ENABLED=false`
**Rationale:**
- Tests run fast without artificial delays
- Rate limiting is tested in unit tests
- Avoids flaky tests from rate limit hits

### D4: RSS Feed Mocking Strategy
**Choice:** Use real OpenAI News RSS feed
**Rationale:**
- Tests real-world RSS parsing behavior
- OpenAI RSS is stable and publicly available
- Fallback to BBC News RSS if needed

**Alternatives Considered:**
- Mock RSS server: More control but adds complexity
- Local RSS files: Doesn't test actual HTTP fetching

### D5: Test Structure
**Choice:** Organize by API domain (auth, feeds, items, opml)
**Rationale:**
- Mirrors API structure
- Easier to find and maintain tests
- Clear separation of concerns

```
web/tests/e2e/
├── playwright.config.ts     # Configuration
├── fixtures/
│   └── auth.ts              # Auth helpers and fixtures
├── auth.spec.ts             # Authentication tests
├── feeds.spec.ts            # Feed management tests
├── items.spec.ts            # Item management tests
└── opml.spec.ts             # OPML import/export tests
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| External RSS feed unavailable | Fallback to BBC News RSS; add retry logic |
| Test database conflicts | Unique database file per test run; cleanup in teardown |
| CSRF token timing issues | Extract token from response body immediately after login |
| Flaky tests from async operations | Use Playwright's built-in waiting and retries |
| Rate limiting accidentally enabled | Verify `RATE_LIMIT_ENABLED=false` in test setup |
