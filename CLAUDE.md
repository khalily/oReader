# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, article management, and academic paper import with PDF-to-Markdown conversion.

### Monorepo Structure

```
backend/                    # Go backend (module: github.com/khalily/oreader)
  cmd/server/               # Main application entry point
  cmd/migrate-to-markdown/  # HTML→Markdown migration tool
  internal/                 # Internal packages (handler, service, repository, etc.)
  migrations/               # SQL migration files
  Dockerfile                # Production build (pure Go, CGO_ENABLED=0)
  Dockerfile.dev            # Development build (air hot-reload)
  Makefile                  # Backend-specific targets

frontend/                   # React frontend (Vite + TypeScript)
  src/                      # Source code
  Dockerfile                # Production build (nginx)
  Dockerfile.dev            # Development build (Vite dev server)
  nginx.conf                # Nginx config for SPA + API proxy
  Makefile                  # Frontend-specific targets

services/converter/         # Python gRPC converter service
  src/                      # converter.py, server.py
  tests/                    # Converter unit tests
  pyproject.toml            # uv project config
  Dockerfile                # Production build
  Makefile                  # Converter-specific targets

proto/                      # Protobuf definitions & generated code
  paper.proto               # Service definition
  go/                       # Generated Go code (independent Go module)
  python/                   # Generated Python code

docker/                     # Docker Compose configurations
  docker-compose.yml        # Development environment
  docker-compose.prod.yml   # Production environment
  docker-compose.test.yml   # Test environment

versions.env                # Single source of truth for tool versions
scripts/                    # Build/utility scripts
docs/                       # Documentation
```

### Environment Setup

```bash
cp .env.example .env   # Required before first run
```

### Development Commands

```bash
# === Docker Compose (Recommended) ===

make docker-dev          # Start dev environment (backend + frontend + converter + MySQL)
make docker-prod         # Start prod environment
make docker-test         # Run tests in Docker
make docker-down         # Stop development
make docker-logs         # Follow logs
make docker-clean        # Remove containers + volumes + images

# === Local Development (Without Docker) ===
# Requires: Go 1.25+, Node.js 22+, MySQL 8.0+, Python 3.11+

# Backend
cd backend && go run ./cmd/server

# Frontend
cd frontend && npm run dev

# Converter service (Python gRPC)
cd services/converter && uv sync && python src/server.py

# === Testing ===

make test                # Backend tests (requires MySQL)
make test-all            # All tests (backend + frontend + converter)

cd backend && make test  # Backend tests directly
cd frontend && npm test  # Frontend tests
cd services/converter && uv run pytest tests/ -v  # Converter tests

# === Lint & Format ===

make lint                # Backend linter
make lint-all            # All linters
make fmt                 # Format all code

# === Database Migrations ===

make migrate-up                # Apply migrations
make migrate-down              # Rollback
make migrate-create name=xxx   # Create new migration
```

## Architecture

### Key Patterns

1. **Authentication**: Dual-token JWT system (access + refresh tokens) stored in HttpOnly cookies. CSRF protection via double-submit cookie pattern.

2. **Repository Pattern**: Services depend on repository interfaces, not concrete implementations. This enables testing with mocks.

3. **Database**: MySQL only (all environments). No SQLite — pure Go, no CGO dependency (`CGO_ENABLED=0`). Docker Compose provides MySQL for development.

4. **Go Module Path**: `github.com/khalily/oreader` (in `backend/go.mod`). Proto generated code is in a separate Go module `github.com/khalily/oreader/proto/go` (in `proto/go/go.mod`), referenced via `replace` directive.

5. **Docker Compose**: Three compose files in `docker/` — dev, prod, test. Build context is project root for backend and converter (needed for proto replace directive). Frontend uses `frontend/` as context.

6. **API Proxy**: In development, Vite proxies `/api/*` requests to the Go backend. In production, nginx handles proxying. Target is configurable via `VITE_API_TARGET` env var.

7. **Frontend State**: Zustand stores manage client-side state. TanStack Query handles server state via custom hooks.

8. **Background Worker**: `internal/worker/` implements feed refresh as a background goroutine with configurable interval.

9. **Pure Go Build**: `CGO_ENABLED=0` — all drivers pure Go. Static binary suitable for `scratch`/`alpine` images.

## Papers Feature Architecture

The Papers feature allows users to upload PDF academic papers, auto-convert to Markdown, and extract metadata via a Python gRPC converter service.

### Architecture Flow

```
User uploads PDF → Go handler validates & saves to disk → Go service spawns goroutine
  → gRPC Convert() call to Python converter → MinerU parses PDF → LLM refines Markdown
  → gRPC ExtractMetadata() → Go service updates Paper record → Frontend polls status
```

### Key Environment Variables

| Component | Variable | Description |
|-----------|----------|-------------|
| Go backend | `PAPER_GRPC_ADDR` | Converter service address (default: `localhost:50051`; Docker: `converter:50051`) |
| Go backend | `PAPER_UPLOAD_DIR` | PDF storage directory (default: `uploads/papers`) |
| Go backend | `PAPER_MAX_UPLOAD_SIZE` | Max upload size in bytes (default: 52428800 = 50MB) |
| Python converter | `LLM_API_KEY` | OpenAI-compatible API key (optional, skips LLM if not set) |
| Python converter | `LLM_BASE_URL` | API base URL (default: OpenAI) |
| Python converter | `GRPC_PORT` | Server port (default: 50051) |

## Markdown Rendering Pipeline

### Backend (Go) - `backend/internal/infra/markdown/converter.go`

The backend converts HTML to Markdown before storing in database:

1. **LaTeX delimiter conversion**: Pandoc style `\(...\)` → `$...$` (inline), `\[...\]` → `$$...$$` (block)
2. **Table support**: `plugin.Table()` enabled for proper Markdown table generation
3. **Code cleanup**: Removes extra backticks from Jekyll/Rouge syntax highlighting
4. **Language identifiers are NOT preserved**: Output is ````\ncode\n``` `` instead of ````python\ncode\n``` ``. The frontend must handle this.

### Frontend (React) - `frontend/src/components/ui/MarkdownRenderer.tsx`

Uses `MarkdownHooks` from react-markdown v10 (required for async plugins like Shiki).

Plugin pipeline: `remarkGfm`, `remarkMath`, `rehypeKatex`, `rehypeAddDefaultLang`, `rehypeShiki`

**Critical**: Plugin arrays MUST be module-level constants (not created inside component) to prevent infinite re-processing.

## Known Gotchas

### All Go tests require MySQL

Tests use `testutil.SetupTestDB()` which connects to a real MySQL instance. Use `make docker-test` or run against a local MySQL.

### MinerU API changed in v1.3.12+

The `magic_pdf.pipe` module no longer exists. New API uses `PymuDocDataset` and `read_api`.

### Converter requires `magic-pdf.json`

MinerU expects `~/magic-pdf.json` config file. The Docker image generates a default CPU-mode config at build time.

### Vite proxy target depends on environment

- Local dev: `VITE_API_TARGET=http://localhost:8080` (default)
- Docker Compose: `VITE_API_TARGET=http://backend:8080` (Docker DNS)
