# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, article management, and academic paper import with PDF-to-Markdown conversion.

### Environment Setup

```bash
cp .env.example .env   # Required before first run
```

### Development Commands

```bash
# === Docker Compose (Recommended) ===

# Full dev environment (backend + frontend + converter + MySQL)
docker compose up
docker compose up --build          # Rebuild images

# Production environment
docker compose --profile prod up -d

# Test environment
docker compose --profile test up --abort-on-container-exit

# Stop
docker compose down                          # Development
docker compose --profile prod down           # Production

# Makefile shortcuts
make docker-dev          # docker compose up --build
make docker-prod         # docker compose --profile prod up -d --build
make docker-test         # docker compose --profile test up --abort-on-container-exit
make docker-down         # Stop development
make docker-logs         # Follow logs
make docker-clean        # Remove containers + volumes + images

# === Local Development (Without Docker) ===
# Requires: Go 1.25+, Node.js 22+, MySQL 8.0+, Python 3.11+

# Backend only (API mode, no embedded frontend)
make dev

# Frontend only
make frontend-dev

# Converter service (Python gRPC)
make converter-install   # Install Python dependencies
make converter-dev       # Start gRPC converter server

# === Testing ===

# Go tests (requires MySQL)
go test ./internal/...
make test                # With coverage

# Frontend tests
cd web && npm test
cd web && npm run test:watch

# Converter tests
make converter-test

# Run ALL tests (Go + Frontend + Converter)
make test-all

# === Build ===

# Build production binary with embedded frontend (CGO_ENABLED=0)
make build

# Build frontend only
make frontend-build

# Lint
make lint                # Go (golangci-lint)
cd web && npm run lint   # Frontend

# Database migrations (requires MySQL)
make migrate-create name=xxx   # Create new migration
make migrate-up                # Apply migrations
make migrate-down              # Rollback
```

## Architecture

### Directory Structure

```
cmd/
  server/              # Main application entry point
  migrate-to-markdown/ # HTML→Markdown migration tool
converter/
  server.py            # Paper converter gRPC server
  converter.py         # PDF parsing (MinerU) + LLM metadata extraction
  requirements.txt     # Python dependencies
  proto/               # Protobuf definitions & generated Go/Python code
  tests/               # Converter unit tests
internal/
  config/              # Configuration loading
  handler/             # HTTP handlers - feeds, items, papers
  infra/
    grpc/              # gRPC client for paper converter service
    markdown/          # Markdown conversion infrastructure
  middleware/           # Auth, CSRF, logging middleware
  model/               # Domain models (User, Feed, Item, Paper, PaperTag)
  repository/          # Data access layer - including paper_repository
  service/             # Business logic - including paper_service
  testutil/            # Test helpers
  worker/              # Background feed refresh worker
migrations/            # SQL migration files
web/src/
  components/
    papers/            # Paper UI components (PaperUpload, PaperList, PaperMeta)
    auth/              # Login, register, social auth
    feed/              # Sidebar, FeedCard, AddFeedDialog
    items/             # ArticlePanel, ItemList
    ui/                # Shared UI (MarkdownRenderer, CopyButton, etc.)
  hooks/               # Custom hooks (useAuth, useFeeds, useItems, useStats, usePapers)
  pages/
    papers/            # PapersPage, PaperViewPage
    items/             # ItemsPage, ItemViewPage
    auth/              # LoginPage, RegisterPage
    oauth/             # OAuthCallbackPage
  lib/api/             # Axios API client
  stores/              # Zustand state stores (authStore, itemsStore)
  types/               # TypeScript types (feed.ts, paper.ts, index.ts)
```

### Key Patterns

1. **Authentication**: Dual-token JWT system (access + refresh tokens) stored in HttpOnly cookies. CSRF protection via double-submit cookie pattern.

2. **Repository Pattern**: Services depend on repository interfaces, not concrete implementations. This enables testing with mocks.

3. **Database**: MySQL only (all environments). No SQLite — pure Go, no CGO dependency (`CGO_ENABLED=0`). Docker Compose provides MySQL for development, tests run against MySQL via `testutil.SetupTestDB()`.

4. **Docker Compose Profiles**: Single `docker-compose.yml` with profiles: default (dev), `prod`, `test`, `tools`. Services communicate via Docker DNS (e.g., `backend:8080`, `converter:50051`, `db:3306`).

5. **API Proxy**: In development, Vite proxies `/api/*` requests to the Go backend. Target is configurable via `VITE_API_TARGET` env var (Docker: `http://backend:8080`, local: `http://localhost:8080`).

6. **OAuth Flow**: GitHub OAuth callback is handled by frontend (`OAuthCallbackPage`) which calls backend with `format=json` to get JSON response instead of 302 redirect.

9. **State Management**: Zustand stores in `web/src/stores/` manage client-side state (auth, items). TanStack Query handles server state caching and revalidation via custom hooks in `web/src/hooks/`.

10. **API Client**: Centralized Axios instance in `web/src/lib/api/axios.ts` handles request/response interceptors, auth token injection, and error transformation.

11. **Background Worker**: `internal/worker/` implements feed refresh as a background goroutine with configurable interval (`REFRESH_INTERVAL` env var, default 15m).

12. **Pure Go Build**: `CGO_ENABLED=0` — all drivers (MySQL via `gorm.io/driver/mysql`) are pure Go. No gcc/musl-dev required in Docker images. Static binary suitable for `scratch`/`alpine` images.

## Papers Feature Architecture

The Papers feature allows users to upload PDF academic papers, auto-convert to Markdown, and extract metadata via a Python gRPC converter service.

### Architecture Flow

```
User uploads PDF → Go handler validates & saves to disk → Go service spawns goroutine
  → gRPC Convert() call to Python converter → MinerU parses PDF → LLM refines Markdown
  → gRPC ExtractMetadata() → Go service updates Paper record → Frontend polls status
```

### Components

1. **Python gRPC Converter** (`converter/`):
   - `server.py`: gRPC server with `Convert` (streaming) and `ExtractMetadata` RPCs
   - `converter.py`: MinerU PDF parsing, LLM metadata extraction & Markdown refinement
   - `proto/paper.proto`: Protobuf service definition
   - Requires `LLM_API_KEY` for metadata extraction (gracefully skips if not set)

2. **Go gRPC Client** (`internal/infra/grpc/paper_client.go`):
   - `PaperConverterClient` interface for testability
   - Non-blocking dial (lazy connection); errors surface on RPC calls

3. **Go Backend**:
   - Handler (`internal/handler/paper_handler.go`): upload, list, get, update, delete, retry, download, tags
   - Service (`internal/service/paper_service.go`): async conversion via goroutine, PDF stored on disk (goroutine reads from disk, not memory), mutex prevents concurrent updates on same paper
   - Model (`internal/model/paper.go`): Paper, PaperTag (+ PaperCollection reserved)

4. **Frontend**:
   - `hooks/usePapers.ts`: TanStack Query hooks, auto-polling status for pending/processing papers
   - `components/papers/`: PaperUpload (drag-drop + apiClient), PaperList, PaperMeta
   - `pages/papers/`: PapersPage (search/filter/pagination), PaperViewPage (MarkdownRenderer)

### Key Environment Variables

| Component | Variable | Description |
|-----------|----------|-------------|
| Go backend | `PAPER_GRPC_ADDR` | Converter service address (default: `localhost:50051`; Docker: `converter:50051`) |
| Go backend | `PAPER_UPLOAD_DIR` | PDF storage directory (default: `uploads/papers`) |
| Go backend | `PAPER_MAX_UPLOAD_SIZE` | Max upload size in bytes (default: 52428800 = 50MB) |
| Python converter | `LLM_API_KEY` | OpenAI-compatible API key (optional, skips LLM if not set) |
| Python converter | `LLM_BASE_URL` | API base URL (default: OpenAI) |
| Python converter | `GRPC_PORT` | Server port (default: 50051) |

In Docker Compose, `PAPER_GRPC_ADDR` is automatically set to `converter:50051` (Docker DNS). When running locally, use `localhost:50051`.

### Testing

- Go: `paper_handler_test.go`, `paper_service_test.go`, `paper_repository_test.go`, `paper_client_test.go`, `model/paper_test.go` — all use `testutil.SetupTestDB()` which connects to MySQL
- Python: `converter/tests/test_converter.py` (mocks OpenAI client)
- Frontend: follow existing patterns in `src/hooks/__tests__/` when adding paper-specific tests
- Docker: `docker compose --profile test up` runs Go tests against MySQL in containers


## Markdown Rendering Pipeline

### Backend (Go) - `internal/infra/markdown/converter.go`

The backend converts HTML to Markdown before storing in database:

1. **LaTeX delimiter conversion**:
   - Pandoc style `\(...\)` → `$...$` (inline)
   - Pandoc style `\[...\]` → `$$...$$` (block)
   - Handles 1-4 levels of backslash escaping (tables have extra escaping)

2. **Table support**:
   - `plugin.Table()` enabled for proper Markdown table generation

3. **Code cleanup**:
   - Removes extra backticks from Jekyll/Rouge syntax highlighting
   - Pattern: `` `` `content` `` `` → `` `content` ``

4. **⚠️ Language identifiers are NOT preserved**: The HTML→Markdown conversion strips language identifiers from code blocks. Output is ````\ncode\n``` `` instead of ````python\ncode\n``` ``. The frontend must handle this (see `rehypeAddDefaultLang` below).

### Frontend (React) - `web/src/components/ui/MarkdownRenderer.tsx`

#### Async rendering with `MarkdownHooks`

`react-markdown` v10 exports three rendering modes:
- `Markdown` — synchronous, calls `processor.runSync()`. **CRASHES with async rehype plugins.**
- `MarkdownAsync` — async/await, server-side only.
- `MarkdownHooks` — async + React hooks (`useState`/`useEffect`), **required for client-side with async plugins.**

We use `MarkdownHooks` because `@shikijs/rehype` is async (loads highlighter on first call).

#### Plugin pipeline

```
remarkPlugins: [remarkGfm, remarkMath]
rehypePlugins:  [rehypeKatex, rehypeAddDefaultLang, [rehypeShiki, shikiOptions]]
```

**Critical**: `remarkPlugins` and `rehypePlugins` MUST be module-level constants (not created inside the component). `MarkdownHooks` uses `useEffect` with these arrays in the dependency — new references each render cause infinite re-processing.

#### `rehypeAddDefaultLang` — bridge for missing language identifiers

Since the backend strips language identifiers (see above), code blocks arrive as ````\ncode\n``` `` without a `language-xxx` class on the `<code>` element. Shiki ONLY processes `<code>` elements with `className` containing `language-xxx`. This plugin runs before Shiki and adds `language-text` to any code block lacking a language class.

#### Shiki dual-theme setup

```ts
const shikiOptions = {
  themes: { light: 'github-light', dark: 'github-dark' },
  defaultColor: false,  // CSS variable mode, not inline color
}
```

With `defaultColor: false`, Shiki generates CSS variables (`--shiki-light`, `--shiki-dark`, `--shiki-light-bg`, `--shiki-dark-bg`) on each `<span>` but does NOT set `color`/`background-color`. The CSS mapping in `index.css` is required:

```css
.shiki, .shiki span {
  color: var(--shiki-light);
  background-color: var(--shiki-light-bg);
}
.dark .shiki, .dark .shiki span {
  color: var(--shiki-dark);
  background-color: var(--shiki-dark-bg);
}
```

#### Component callback patterns

**Must exclude `node` prop** from all component callbacks (`pre`, `img`, `a`, etc.) — `node` is a HAST node object that leaks to DOM as `[object Object]`:

```tsx
// Correct pattern — destructure out node, then spread rest
a({ href, children, ...restProps }) {
  const { node: _node, ...props } = restProps as any
  return <a href={href} target="_blank" rel="noopener noreferrer" {...props}>{children}</a>
}
```

#### CSS override for Tailwind Typography

Tailwind Typography plugin adds backtick decorators to `<code>` elements. Override in `index.css`:

```css
.prose :not(pre) > code::before,
.prose :not(pre) > code::after {
  content: none !important;
}
```

### Testing

- Unit tests: `converter_test.go` (backend), `MarkdownRenderer.test.tsx` + `MarkdownRenderer.styles.test.tsx` (frontend)
- `@shikijs/rehype` must be mocked as async in tests: `vi.mock('@shikijs/rehype', () => ({ default: () => async (tree: any) => tree }))`
- All test assertions must use `waitFor` / `findBy*` (not sync `getBy*`) because `MarkdownHooks` renders asynchronously
- CSS pseudo-elements (`::before`/`::after`) cannot be tested in jsdom — require E2E tests

## Frontend Stack Constraints

### Known Issues

- `prismjs` remains in `package.json` dependencies but is unused (replaced by Shiki). Safe to remove.

### react-markdown v10 rendering modes

| Mode | Sync/Async | Environment | Use case |
|------|-----------|-------------|----------|
| `Markdown` | `runSync()` | Client + Server | Only sync rehype plugins |
| `MarkdownAsync` | `await run()` | Server (RSC) | Async plugins, SSR |
| `MarkdownHooks` | `run()` via hooks | Client | Async plugins, CSR |

**Rule**: If any rehype plugin is async (returns a Promise), you MUST use `MarkdownHooks` on the client. Using `Markdown` will throw `runSync finished async. Use run instead`.

### Plugin reference stability

`MarkdownHooks`'s `useEffect` dependency array is:
```
[options.children, options.rehypePlugins, options.remarkPlugins, options.remarkRehypeOptions]
```

If `rehypePlugins` or `remarkPlugins` are created inside the component function, every render produces a new array reference → `useEffect` re-fires → full re-processing → performance degradation. Always define these as module-level constants.

## Known Gotchas

### `go mod tidy` fails with node_modules error

`go mod tidy` may error with: `import path should not have @version` pointing to `web/node_modules/`. This is caused by node packages containing Go code. **`go build` and `go vet` work fine** — the tidy error is non-blocking. Do not attempt to fix this by modifying node_modules.

### All Go tests require MySQL

Tests use `testutil.SetupTestDB()` which connects to a real MySQL instance. Running `go test ./internal/...` without a running MySQL will fail. Use Docker Compose: `docker compose --profile test up --abort-on-container-exit`.

### MinerU API changed in v1.3.12+

The `magic_pdf.pipe` module no longer exists. New API uses `PymuDocDataset` and `read_api`. If converter logs show `No module named 'magic_pdf.pipe'`, the converter code needs updating to the new MinerU API.

### Converter requires `magic-pdf.json`

MinerU expects `~/magic-pdf.json` config file. The Docker image generates a default CPU-mode config at build time. When running locally, create this file manually (see `converter/Dockerfile` for the default content).

### Vite proxy target depends on environment

- Local dev: `VITE_API_TARGET=http://localhost:8080` (default)
- Docker Compose: `VITE_API_TARGET=http://backend:8080` (Docker DNS)
If API calls fail from the frontend container, check this env var.
