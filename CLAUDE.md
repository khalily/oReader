# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, and article management.

## Development Commands

### Backend (Go)

```bash
# Start development server (API-only mode, no frontend build required)
make dev
# Or with environment: DATABASE_URL=oreader.db JWT_SECRET_KEY=xxx make dev

# Run all tests with coverage
make test

# Run specific package tests
go test ./internal/service -v
go test ./internal/handler -v -run TestOAuthHandler

# Build production binary (with embedded frontend)
make build

# Lint and security check
make lint
make vulncheck
```

### Frontend (React/TypeScript)

```bash
cd web

# Start development server (proxies to backend at localhost:8080)
npm run dev

# Build for production
npm run build

# Run unit tests
npm run test

# Run E2E tests (Playwright)
npm run test:e2e
```

### Full Development Environment

```bash
# Run both backend and frontend with one command
make dev-full
# Or: ./dev.sh
```

## Architecture

### Backend Structure (`internal/`)

- **`handler/`** - HTTP handlers for API endpoints. Each handler depends on services, not repositories directly.
- **`service/`** - Business logic layer. Defines interfaces in `interfaces.go`.
- **`repository/`** - Data access layer implementing service interfaces.
- **`model/`** - GORM models with UUID v7 IDs via embedded `Base` struct.
- **`middleware/`** - HTTP middleware: auth, CORS, rate limiting, security headers.
- **`infra/`** - Infrastructure: JWT, CSRF, cookies, password hashing, RSS parsing.
- **`config/`** - Configuration via environment variables (viper).

### Frontend Structure (`web/src/`)

- **`pages/`** - Route-level components (ItemsPage, LoginPage, OAuth pages).
- **`components/`** - Reusable UI components organized by domain (auth/, feed/, items/, ui/).
- **`hooks/`** - React Query hooks for API calls (useAuth, useFeeds, useItems).
- **`stores/`** - Zustand stores for client state (authStore).
- **`lib/api/`** - Axios client with interceptors for CSRF and token refresh.

### Key Patterns

1. **Authentication**: Dual-token JWT system (access + refresh tokens) stored in HttpOnly cookies. CSRF protection via double-submit cookie pattern.

2. **Repository Pattern**: Services depend on repository interfaces, not concrete implementations. This enables testing with mocks.

3. **API Proxy**: In development, Vite proxies `/api/*` requests to the Go backend. The frontend runs on port 5173, backend on 8080.

4. **OAuth Flow**: GitHub OAuth callback is handled by frontend (`OAuthCallbackPage`) which calls backend with `format=json` to get JSON response instead of 302 redirect.

## Required Environment Variables

- `DATABASE_URL` - SQLite file path or MySQL connection string
- `JWT_SECRET_KEY` - Minimum 32 characters

## OAuth Configuration (Optional)

- `GITHUB_CLIENT_ID` - GitHub OAuth app client ID
- `GITHUB_CLIENT_SECRET` - GitHub OAuth app client secret
- `GITHUB_CALLBACK_HOST` - Callback host (e.g., `localhost:5173` for dev)
- `FRONTEND_URL` - Frontend URL for redirects (default: `http://localhost:5173`)

## Documentation Language

- 提案、规范等文档使用中文撰写
- 代码、变量名、函数名使用英文


## Debug
Add temporary debug logging to show [request/response/cookies] values, then test the flow again

## Test
Set up common mocks first: vi.mock('lucide-react'), vi.mock for useToast/useTheme, IntersectionObserver mock - then write the actual test

## Coding

bugfix and feature using TDD red-green-refactor.

## LaTeX Math Rendering

The application supports LaTeX math rendering through KaTeX:

- **Inline math**: Surround with `$...$` (e.g., `$E = mc^2$`)
- **Block math**: Surround with `$$...$$` (e.g., `$$\frac{\partial L}{\partial w} = \nabla$$`)
- **Tables**: Math expressions work within table cells
- **Syntax highlighting**: Code blocks also support syntax highlighting

Dependencies:
- `react-markdown` with `remark-math` plugin
- `rehype-katex` for KaTeX rendering
- KaTeX CSS assets are automatically included in production build
