# E2E API Testing with Playwright

This directory contains end-to-end API tests for the oReader backend using Playwright.

## Setup

### Prerequisites

- Node.js 18+ installed
- Go 1.21+ installed
- Backend dependencies installed (`go mod download`)
- Frontend dependencies installed (`cd web && npm install`)

### Installation

Playwright is already included in the project's `package.json`. Install dependencies:

```bash
cd web
npm install
```

Install Playwright browsers:

```bash
npx playwright install chromium
```

## Running Tests

### Run All E2E Tests

```bash
cd web
npm run test:e2e
```

### Run Tests with UI

```bash
cd web
npm run test:e2e:ui
```

### Run Specific Test File

```bash
cd web
npx playwright test --config=tests/e2e/playwright.config.ts tests/e2e/auth.spec.ts
```

## Test Structure

```
tests/e2e/
├── playwright.config.ts     # Playwright configuration
├── setup.ts                  # Test database initialization
├── fixtures/
│   └── auth.ts              # Authentication helpers and fixtures
├── auth.spec.ts             # Authentication endpoint tests
├── feeds.spec.ts            # Feed management tests
├── items.spec.ts            # Item management tests
├── opml.spec.ts             # OPML import/export tests
└── error-handling.spec.ts   # Error scenario tests
```

## Test Database

Tests use a separate SQLite database (`oreader_test.db`) to avoid affecting development or production data.

- Database is created fresh before each test run
- Located at project root
- Deleted and recreated between runs

## Test Environment

Tests run with the following environment variables:

- `ENV=development` - Run in development mode
- `DATABASE_URL=oreader_test.db` - Use test database
- `RATE_LIMIT_ENABLED=false` - Disable rate limiting for faster tests

## Authentication Flow

Tests handle authentication automatically using the fixtures in `fixtures/auth.ts`:

1. Register/login a test user
2. Extract `csrf_token` from response body
3. Inject CSRF token in subsequent POST/PUT/DELETE requests via `X-CSRF-Token` header

### Using Auth Fixtures

```typescript
import { registerUser, createTestUser, authPost } from './fixtures/auth';

test('example test', async ({ request }) => {
  const testUser = createTestUser();
  const { auth } = await registerUser(request, testUser);

  // Make authenticated request
  const response = await authPost(request, '/api/v1/feeds', auth, {
    feed_url: 'https://example.com/feed.xml'
  });
});
```

## External Dependencies

Tests use real RSS feeds to validate actual parsing behavior:

- **Primary**: OpenAI News RSS (`https://openai.com/news/rss.xml`)
- **Fallback**: BBC News RSS (if needed)

## Debugging Failed Tests

### View Test Report

```bash
npx playwright show-report
```

### Run in Debug Mode

```bash
npx playwright test --debug
```

### View Browser

```bash
npx playwright test --headed
```

## CI/CD Integration

Tests are configured to run in CI environments:

- Headless mode by default
- Automatic retries on CI (2 retries)
- Sequential execution to avoid database conflicts
- Test artifacts saved on failure

## Troubleshooting

### Server Won't Start

- Ensure port 8080 is not in use
- Check that `web/dist` directory exists (run `npm run build` in web/)
- Verify Go dependencies are installed

### CSRF Token Errors

- Ensure you're extracting `csrf_token` from login/register response
- Verify `X-CSRF-Token` header is included in POST/PUT/DELETE requests

### Database Errors

- Delete `oreader_test.db` manually if corrupted
- Check file permissions in project directory

### Test Timeouts

- Increase timeout in `playwright.config.ts` if needed
- Check network connectivity for RSS feed tests
