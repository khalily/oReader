# Implementation Tasks (TDD + Phase Commits)

> **Methodology**: Test-Driven Development (TDD)
> - 🔴 Write failing test first
> - 🟢 Write minimal code to pass test
> - 🔵 Refactor if needed
>
> **Git Strategy**: Commit after each completed Phase

---

## ⚠️ Priority Fix List (From Architecture Review)

These items were identified as critical/high priority by the architect review and should be addressed during implementation:

### Critical (Must Implement)
| # | Issue | Phase | Status |
|---|-------|-------|--------|
| C1 | Add explicit CSRF protection beyond SameSite cookies | Phase 3 | Pending |
| C2 | Define repository/service interfaces for proper layering | Phase 2 | Pending |
| C3 | Add content sanitization for RSS feed content | Phase 5 | Pending |
| C4 | Block SSRF attacks in feed URL fetching | Phase 5 | Pending |
| C5 | Document all environment variables and configuration | Phase 1 | Pending |

### High Priority (Should Implement)
| # | Issue | Phase | Status |
|---|-------|-------|--------|
| H1 | Use golang-migrate instead of AutoMigrate | Phase 2 | Pending |
| H2 | Design rate limiter with interface (in-memory + Redis) | Phase 4 | Pending |
| H3 | Add structured logging (zerolog) | Phase 2 | Pending |
| H4 | Add security headers middleware | Phase 2 | Pending |
| H5 | Define consistent API error response format | Phase 2 | Pending |

---

## Phase 1: Project Setup

- [x] 1.1 Initialize Go module (`go mod init oreader`)
- [x] 1.2 Create project directory structure (`cmd/`, `internal/`, `web/`, `migrations/`)
- [x] 1.3 Create Makefile with `test`, `build`, `run`, `migrate` commands
- [x] 1.4 Add Go test dependencies (`testify`, `mockery`)
- [x] 1.5 Configure test coverage reporting
- [x] 1.6 Initialize React frontend with Vite + TypeScript
- [x] 1.7 Add frontend dependencies
- [x] 1.8 Configure Tailwind CSS and shadcn/ui
- [x] 1.9 Create `.env.example` with all documented environment variables **[C5]**
- [x] 1.10 Add `govulncheck` to CI configuration
- [x] 1.11 Add `npm audit` to CI configuration

**Git Commit**: `git commit -m "feat: project setup with Go + React structure"`

---

## Phase 2: Backend Core Infrastructure

### 🔴 Write Tests
- [x] 2.1 Write tests for configuration loading
- [x] 2.2 Write tests for database connection
- [x] 2.3 Write tests for data models validation

### 🟢 Implement
- [x] 2.4 Implement configuration loading with viper
- [x] 2.5 Define configuration struct with all required fields
- [x] 2.6 Create database connection with GORM
- [x] 2.7 Define data models with multi-tenant support:
  - **User**: id (UUID v7), email, password_hash, nickname, avatar_url, auth_provider, github_id, created_at, updated_at
  - **Feed**: id (UUID v7), feed_url (unique), title, description, image_url, last_fetched_at, last_fetch_status, consecutive_failures
  - **UserFeed**: id (UUID v7), user_id, feed_id, position, created_at (subscription relationship)
  - **Item**: id (UUID v7), feed_id, guid (unique per feed), title, link, description, content, pub_date, creator
  - **UserItemState**: id (UUID v7), user_id, item_id, is_starred, is_read, read_at, created_at
  - **RefreshToken**: id (UUID v7), user_id, token_hash, expires_at, revoked, created_at
- [x] 2.8 **[H1]** Create golang-migrate migration files (not AutoMigrate)
- [x] 2.9 **[C2]** Define repository interfaces in `internal/service/interfaces.go`
- [x] 2.10 **[H3]** Initialize zerolog with environment-based formatting
- [x] 2.11 **[H4]** Implement security headers middleware
- [x] 2.12 **[H5]** Implement error response helpers with standard format
- [x] 2.13 Create Gin router with route groups
- [x] 2.14 Implement CORS middleware
- [x] 2.15 Implement request logging middleware with request_id
- [x] 2.16 Create main.go entry point

### 🔵 Verify & Refactor
- [x] 2.17 Run all tests: `make test`
- [x] 2.18 Ensure >80% coverage on config and models
- [x] 2.19 Verify migrations run successfully with `make migrate-up`
- [x] 2.20 **验证多用户场景**: 两个用户订阅同一 RSS 源，各自标记阅读/收藏状态互不影响

**Git Commit**: `git commit -m "feat(backend): core infrastructure with models, interfaces, and logging"`

---

## Phase 3: Authentication System

### 🔴 Write Tests
- [x] 3.1 Write tests for JWT token generation/validation
- [x] 3.2 Write tests for refresh token CRUD
- [x] 3.3 Write tests for password hashing (bcrypt cost 12)
- [x] 3.4 Write tests for CSRF token generation and validation **[C1]**
- [x] 3.5 Write tests for auth service (register, login, logout, refresh)
- [x] 3.6 Write tests for auth handler endpoints
- [x] 3.7 Write tests for auth middleware
- [x] 3.8 Write tests for CSRF middleware **[C1]**

### 🟢 Implement
- [x] 3.9 Implement JWT service (`internal/infra/jwt/`) with HS256 and 256-bit secret
- [x] 3.10 Implement refresh token generation and hashing
- [x] 3.11 **[C1]** Implement CSRF token generation and validation
- [x] 3.12 Implement cookie utilities (`internal/infra/cookie/`) with HttpOnly + SameSite=Strict
- [x] 3.13 Implement password hashing with bcrypt (cost 12)
- [x] 3.14 Implement User repository (implements interface)
- [x] 3.15 Implement RefreshToken repository (implements interface)
- [x] 3.16 Implement auth service
- [x] 3.17 Implement auth handler (register, login, logout, refresh, me)
- [x] 3.18 Implement JWT authentication middleware
- [x] 3.19 **[C1]** Implement CSRF middleware for state-changing requests
- [x] 3.20 Add authentication event logging (login, logout, refresh)

### 🔵 Verify & Refactor
- [x] 3.21 Run all tests: `make test`
- [x] 3.22 Ensure >80% coverage on auth package
- [x] 3.23 Test authentication flow manually including CSRF
- [x] 3.24 Verify TOKEN_EXPIRED error code is returned correctly

**Git Commit**: `git commit -m "feat(auth): dual-token authentication with HttpOnly cookies and CSRF protection"`

---

## Phase 4: Rate Limiting

### 🔴 Write Tests
- [x] 4.1 Write tests for token bucket limiter
- [x] 4.2 Write tests for rate limit middleware
- [x] 4.3 Write tests for rate limit headers
- [x] 4.4 **[H2]** Write tests for rate limiter interface

### 🟢 Implement
- [x] 4.5 **[H2]** Define RateLimiter interface in `internal/infra/ratelimit/`
- [x] 4.6 **[H2]** Implement in-memory token bucket rate limiter
- [x] 4.7 **[H2]** Add Redis rate limiter skeleton (configurable)
- [x] 4.8 Implement rate limit middleware
- [x] 4.9 Configure rate limits per endpoint type
- [x] 4.10 Add rate limit headers to responses (X-RateLimit-*)
- [x] 4.11 Use IP + user_id combination for authenticated rate limiting

### 🔵 Verify & Refactor
- [x] 4.12 Run all tests: `make test`
- [x] 4.13 Verify rate limiting works under load
- [x] 4.14 Verify RATE_LIMIT_EXCEEDED error format

**Git Commit**: `git commit -m "feat(middleware): API rate limiting with interface-based design"`

---

## Phase 5: RSS Parsing & Subscription

### 🔴 Write Tests
- [x] 5.1 Write tests for RSS parser wrapper
- [x] 5.2 **[C4]** Write tests for URL validation (SSRF protection)
- [x] 5.3 **[C3]** Write tests for content sanitization
- [x] 5.4 Write tests for feed repository
- [x] 5.5 Write tests for item repository
- [x] 5.6 Write tests for feed service
- [x] 5.7 Write tests for feed handler endpoints

### 🟢 Implement
- [x] 5.8 Implement RSS parser wrapper using gofeed (`internal/infra/rss/`)
- [x] 5.9 **[C4]** Implement URL validation with private IP blocking
- [x] 5.10 **[C4]** Implement URL scheme validation (http/https only)
- [x] 5.11 **[C3]** Implement HTML sanitization with bluemonday (`internal/infra/sanitize/`)
- [x] 5.12 Implement favicon extraction
- [x] 5.13 Implement Feed repository (implements interface)
- [x] 5.14 Implement Item repository (implements interface)
- [x] 5.15 Implement feed service with sanitization
- [x] 5.16 Implement feed handler (CRUD + manual refresh)
- [x] 5.17 Add feed size limits (1000 items, 1MB content, 30s timeout)

### 🔵 Verify & Refactor
- [x] 5.18 Run all tests: `make test`
- [x] 5.19 Test with real RSS feeds
- [x] 5.20 Test SSRF protection with blocked IPs
- [x] 5.21 Test XSS payloads are sanitized

**Git Commit**: `git commit -m "feat(feeds): RSS subscription management with SSRF protection and content sanitization"`

---

## Phase 6: Article Management

### 🔴 Write Tests
- [ ] 6.1 Write tests for item service
- [ ] 6.2 Write tests for item handler endpoints
- [ ] 6.3 Write tests for star/read operations

### 🟢 Implement
- [ ] 6.4 Implement item service
- [ ] 6.5 Implement item handler (list, get, star, read, mark-all-read)
- [ ] 6.6 Implement filtering and pagination with cursor-based approach
- [ ] 6.7 Implement bulk operations (bulk mark-as-read)

### 🔵 Verify & Refactor
- [ ] 6.8 Run all tests: `make test`
- [ ] 6.9 Verify article operations work correctly
- [ ] 6.10 Verify pagination returns correct metadata

**Git Commit**: `git commit -m "feat(items): article management with star/read"`

---

## Phase 7: Background Refresh Worker

### 🔴 Write Tests
- [ ] 7.1 Write tests for refresh worker service
- [ ] 7.2 Write tests for concurrent refresh logic
- [ ] 7.3 Write tests for graceful shutdown

### 🟢 Implement
- [ ] 7.4 Implement refresh worker service
- [ ] 7.5 Implement concurrent feed refresh with bounded semaphore (max 10)
- [ ] 7.6 Add context timeout for individual fetches (30s)
- [ ] 7.7 Implement periodic ticker
- [ ] 7.8 Implement startup refresh trigger
- [ ] 7.9 Add graceful shutdown handling (30s timeout)
- [ ] 7.10 Add feed refresh operation logging
- [ ] 7.11 Track consecutive failures and last_fetch_status in Feed model

### 🔵 Verify & Refactor
- [ ] 7.12 Run all tests: `make test`
- [ ] 7.13 Test background refresh manually
- [ ] 7.14 Verify logging includes all required fields

**Git Commit**: `git commit -m "feat(worker): background RSS refresh with bounded concurrency"`

---

## Phase 8: Feed Import/Export (OPML)

### 🔴 Write Tests
- [ ] 8.1 Write tests for OPML export generation
- [ ] 8.2 Write tests for OPML import parsing
- [ ] 8.3 Write tests for bulk import handling
- [ ] 8.4 Write tests for import handler endpoints

### 🟢 Implement
- [ ] 8.5 Implement OPML export generator (`internal/infra/opml/`)
- [ ] 8.6 Implement OPML import parser
- [ ] 8.7 Implement bulk import with async processing
- [ ] 8.8 Store import job state in database (not memory)
- [ ] 8.9 Implement import/export handlers
- [ ] 8.10 Add import progress tracking

### 🔵 Verify & Refactor
- [ ] 8.11 Run all tests: `make test`
- [ ] 8.12 Test OPML import/export with real files
- [ ] 8.13 Verify job state persists across restarts

**Git Commit**: `git commit -m "feat(feeds): OPML import/export with persistent job tracking"`

---

## Phase 9: OAuth Integration (Reserved)

### 🔴 Write Tests
- [ ] 9.1 Write tests for OAuth state generation/validation
- [ ] 9.2 Write tests for GitHub OAuth flow (mocked)

### 🟢 Implement
- [ ] 9.3 Implement OAuth handler skeleton
- [ ] 9.4 Implement GitHub OAuth endpoints (initiate, callback)
- [ ] 9.5 Implement user creation/linking for OAuth
- [ ] 9.6 Add reserved endpoints for other providers (501)
- [ ] 9.7 Store OAuth state in database for validation

### 🔵 Verify & Refactor
- [ ] 9.8 Run all tests: `make test`

**Git Commit**: `git commit -m "feat(oauth): GitHub OAuth integration (reserved)"`

---

## Phase 10: Frontend Core Setup

- [ ] 10.1 Create Axios instance with `withCredentials`
- [ ] 10.2 **[C1]** Configure CSRF token handling in Axios interceptor
- [ ] 10.3 Create API response types (`web/src/types/`)
- [ ] 10.4 Create typed API client (`web/src/lib/api/`)
- [ ] 10.5 Create auth store with Zustand (isAuthenticated, user, csrfToken)
- [ ] 10.6 Implement axios response interceptor for auto-refresh with request queuing
- [ ] 10.7 Handle TOKEN_EXPIRED error code to trigger refresh
- [ ] 10.8 Set up React Query provider
- [ ] 10.9 Create React Router configuration
- [ ] 10.10 Create Layout component (Header + Sidebar)
- [ ] 10.11 Implement protected route wrapper

**Git Commit**: `git commit -m "feat(frontend): core setup with routing, state, and CSRF handling"`

---

## Phase 11: Frontend Authentication

- [ ] 11.1 Create auth API hooks with React Query
- [ ] 11.2 Create Login page component
- [ ] 11.3 Create Register page component
- [ ] 11.4 Implement forms with React Hook Form + Zod validation
- [ ] 11.5 Implement logout functionality
- [ ] 11.6 Add form error handling and loading states
- [ ] 11.7 Display API error messages using standard error format

**Git Commit**: `git commit -m "feat(frontend): authentication pages"`

---

## Phase 12: Frontend Feed Management

- [ ] 12.1 Create feed API hooks with React Query
- [ ] 12.2 Create Sidebar component with feed list
- [ ] 12.3 Create AddFeed dialog component
- [ ] 12.4 Implement feed subscription form
- [ ] 12.5 Implement feed deletion with confirmation
- [ ] 12.6 Create OPML import/export UI components

**Git Commit**: `git commit -m "feat(frontend): feed management UI"`

---

## Phase 13: Frontend Article Display

- [ ] 13.1 Create item API hooks with React Query
- [ ] 13.2 Create ItemList component
- [ ] 13.3 Create ItemView page for article reading
- [ ] 13.4 Implement star/unstar toggle
- [ ] 13.5 Implement mark as read functionality
- [ ] 13.6 Create StarredItems page
- [ ] 13.7 Implement cursor-based pagination/infinite scroll

**Git Commit**: `git commit -m "feat(frontend): article display and reading"`

---

## Phase 14: Frontend Polish

- [ ] 14.1 Create loading skeletons
- [ ] 14.2 Implement error boundary components (App-level + Feature-level)
- [ ] 14.3 Add toast notifications
- [ ] 14.4 Implement responsive design
- [ ] 14.5 Add keyboard shortcuts
- [ ] 14.6 Implement dark mode toggle (optional)

**Git Commit**: `git commit -m "feat(frontend): polish with loading, errors, responsive"`

---

## Phase 15: Static File Embedding

- [ ] 15.1 Configure Vite build output to `web/dist`
- [ ] 15.2 Create `embed.go` for static files
- [ ] 15.3 Implement static file serving in Gin
- [ ] 15.4 Configure SPA fallback routing
- [ ] 15.5 Test production build locally

**Git Commit**: `git commit -m "feat(build): frontend embedding for single binary"`

---

## Phase 16: Docker & Deployment

- [ ] 16.1 Create multi-stage Dockerfile (Go build + Node build + final image)
- [ ] 16.2 Create `docker-compose.yml` for development
- [ ] 16.3 Create `docker-compose.prod.yml` for production
- [ ] 16.4 Configure MySQL service
- [ ] 16.5 Create health check endpoint
- [ ] 16.6 Add `.dockerignore`
- [ ] 16.7 Test Docker build and run
- [ ] 16.8 Add graceful shutdown in Docker (SIGTERM handling)

**Git Commit**: `git commit -m "feat(deploy): Docker configuration with multi-stage build"`

---

## Phase 17: Final Testing & Documentation

- [ ] 17.1 Run full test suite: `make test`
- [ ] 17.2 Verify test coverage >80%
- [ ] 17.3 Run integration tests (HTTP endpoint tests)
- [ ] 17.4 Run `govulncheck` and `npm audit`
- [ ] 17.5 Security review (cookies, JWT, validation, CSRF, SSRF)
- [ ] 17.6 Create README.md with setup instructions
- [ ] 17.7 Create API documentation (OpenAPI/Swagger)
- [ ] 17.8 Document deployment guide with environment variables

**Git Commit**: `git commit -m "docs: README, API documentation, and deployment guide"`

---

## Phase 18: Final Verification

- [ ] 18.1 Verify all API endpoints work correctly
- [ ] 18.2 Verify authentication flow end-to-end (including CSRF)
- [ ] 18.3 Verify RSS subscription and parsing
- [ ] 18.4 Verify background refresh
- [ ] 18.5 Verify OPML import/export
- [ ] 18.6 Verify rate limiting (in-memory)
- [ ] 18.7 Verify Docker deployment
- [ ] 18.8 Verify security headers on all responses
- [ ] 18.9 Verify content sanitization (XSS test)
- [ ] 18.10 Verify SSRF protection (blocked IPs)

**Git Commit**: `git commit -m "release: oReader v2.0.0"`

---

## Summary

| Phase | Description | TDD | Critical Fixes | High Fixes | Commit |
|-------|-------------|-----|----------------|------------|--------|
| 1 | Project Setup | - | C5 | - | ✅ |
| 2 | Backend Core | ✅ | C2 | H1, H3, H4, H5 | ✅ |
| 3 | Authentication | ✅ | C1 | - | ✅ |
| 4 | Rate Limiting | ✅ | - | H2 | ✅ |
| 5 | RSS/Feeds | ✅ | C3, C4 | - | ✅ |
| 6 | Articles | ✅ | - | - | ✅ |
| 7 | Background Worker | ✅ | - | - | ✅ |
| 8 | OPML Import/Export | ✅ | - | - | ✅ |
| 9 | OAuth (Reserved) | ✅ | - | - | ✅ |
| 10 | Frontend Core | - | C1 | - | ✅ |
| 11 | Frontend Auth | - | - | - | ✅ |
| 12 | Frontend Feeds | - | - | - | ✅ |
| 13 | Frontend Articles | - | - | - | ✅ |
| 14 | Frontend Polish | - | - | - | ✅ |
| 15 | Static Embedding | - | - | - | ✅ |
| 16 | Docker Deploy | - | - | - | ✅ |
| 17 | Testing & Docs | - | - | - | ✅ |
| 18 | Final Verification | - | - | - | ✅ |

**Total: 18 Phases, 18 Git Commits**

**Critical Fixes: C1-CSRF, C2-Interfaces, C3-Sanitization, C4-SSRF, C5-Config**
**High Fixes: H1-Migrations, H2-RateLimiter Interface, H3-Logging, H4-Security Headers, H5-Error Format**
