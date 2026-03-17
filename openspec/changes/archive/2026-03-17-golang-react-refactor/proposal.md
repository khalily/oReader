## Why

The current oReader RSS reader is built with Flask (backend) and AngularJS 1.x (frontend), both of which are outdated technologies. AngularJS 1.x has reached EOL, and the team is more proficient in Go. Additionally, the project aims to add more features in the future, requiring a more modern and maintainable architecture.

## What Changes

**BREAKING**: Complete rewrite of the application stack.

### Backend
- Replace Flask with Go + Gin framework
- Replace SQLAlchemy with GORM
- Replace custom lxml RSS parser with gofeed library
- Replace itsdangerous token with golang-jwt
- Add dual-token authentication (Access + Refresh Token)
- Add background RSS refresh worker (goroutine)

### Frontend
- Replace AngularJS 1.x with React 18 + TypeScript
- Replace angular-route with React Router
- Replace $resource with React Query + Axios
- Replace angular-store with Zustand
- Add shadcn/ui + Tailwind CSS for UI components
- Use Vite as build tool

### Security Enhancements
- Replace auto-increment user IDs with UUID v7
- Implement dual-token mechanism (15min access + 7day refresh)
- Store tokens in HttpOnly + SameSite=Strict cookies
- Store refresh tokens in database for revocation support
- Add explicit CSRF token protection (defense-in-depth)
- Sanitize all RSS content with bluemonday (XSS prevention)
- Block SSRF attacks by validating feed URLs (private IP blocking)
- Add security headers middleware (X-Frame-Options, CSP, etc.)
- Implement rate limiting with interface (in-memory + Redis ready)

### Deployment
- Single binary deployment (frontend embedded in Go binary)
- Docker containerization
- SQLite for development, MySQL 8.0 for production

### Features
- Multi-user registration/login with JWT
- RSS subscription management (add/delete/list)
- Article reading (list/detail/star/read)
- Scheduled auto-refresh + manual refresh
- Reserved GitHub OAuth login interface
- API rate limiting for abuse prevention
- OPML import/export for feed subscriptions

### Development Methodology
- Test-Driven Development (TDD) - write tests first, then implementation
- Git commit after each completed phase

## Capabilities

### New Capabilities

- `user-auth`: User registration, login, logout, and JWT token management with dual-token mechanism
- `rss-subscription`: RSS feed subscription management (add, delete, list feeds)
- `rss-parsing`: RSS/Atom feed parsing using gofeed library
- `article-management`: Article listing, reading, starring, and read status management
- `background-refresh`: Background worker for automatic RSS feed refresh
- `oauth-integration`: Third-party OAuth login support (GitHub OAuth reserved)
- `rate-limiting`: API rate limiting to prevent abuse and protect resources
- `feed-import-export`: OPML import and export for feed subscriptions
- `security-hardening`: CSRF protection, content sanitization, SSRF prevention, security headers
- `logging-strategy`: Structured JSON logging with request context and audit events
- `error-format`: Consistent API error responses with codes and details
- `data-model`: Multi-tenant feed sharing with per-user item states (UserFeed, UserItemState)

### Modified Capabilities

None - This is a complete rewrite, not modifying existing capabilities.

## Impact

### Codebase
- Complete rewrite in Go and TypeScript
- New project structure following Go conventions (cmd/, internal/, pkg/)
- New frontend structure with React component architecture

### APIs
- All API endpoints remain RESTful but with new implementation
- New endpoints: `/api/v1/auth/refresh`, `/api/v1/auth/github`, `/api/v1/auth/github/callback`
- Cookie-based authentication replaces Authorization header

### Dependencies
- Backend: Go 1.21+, Gin, GORM, golang-jwt, gofeed, viper, bcrypt
- Frontend: React 18, TypeScript, Vite, React Query, Zustand, shadcn/ui, Tailwind CSS
- Database: SQLite (dev) / MySQL 8.0 (prod)
- Deployment: Docker, single binary

### Data Migration
- No data migration planned - fresh start with new database schema
- New schema uses UUID primary keys instead of auto-increment integers
