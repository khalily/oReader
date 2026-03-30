# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Workflow

- 代码变更后，同步更新 README.md 和 CLAUDE.md
- 构建/开发/运行方式修改后，同步更新 Makefile、Dockerfile、CI 配置（.github/workflows/）和 versions.env，保障多环境一致性
- 统一日志格式，API 出入口、关键流程、错误场景添加日志
- 修改代码时基于 TDD 方式走完整验证流程（Red-Green-Refactor）
  - 例外：配置文件、纯文案/注释修改、文档类改动可简化流程
- 脚本/配置中使用绝对路径，脚本自动获取路径
- 二进制、构建文件等添加到 .gitignore，避免 commit
- docs/superpowers/ 下的文件也需要 commit（plans、specs 等）
- 代码 commit 时自动触发 pre-commit lint 检查（`.claude/hooks/pre-commit-lint.sh`）：按暂存文件类型运行 golangci-lint / eslint / flake8+black / redocly。lint 失败会阻止 commit，可用 `--no-verify` 跳过

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, article management, and academic paper import with PDF-to-Markdown conversion.

### Monorepo Structure

```
backend/                    # Go backend (module: github.com/khalily/oreader)
  cmd/server/               # Main application entry point
  cmd/migrate-to-markdown/  # HTML→Markdown migration tool
  internal/                 # handler, service, repository, model, middleware, worker, infra, testutil
  migrations/               # SQL migration files
  Dockerfile                # Production build (pure Go, CGO_ENABLED=0)
  Dockerfile.dev            # Development build (air hot-reload)
  Makefile                  # Backend-specific targets

frontend/                   # React frontend (Vite + TypeScript)
  src/                      # components, hooks, pages, stores, lib, types, test
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

docker/                     # Docker Compose + environment configs
  docker-compose.yml        # Development environment
  docker-compose.prod.yml   # Production environment
  docker-compose.test.yml   # Test environment
  .env.example              # Production env template
  .env.test                 # Test environment overrides

versions.env                # Single source of truth for tool versions
scripts/                    # check-versions.sh, ci-load-versions.sh
docs/                       # API docs, OpenAPI spec, deployment guide
```

### Environment Setup

```bash
cp .env.example .env   # Required before first run
```

## Quick Reference

### Development Commands

```bash
# === Docker Compose (Recommended) ===

make docker-dev          # Start dev (backend + frontend + converter + MySQL)
make docker-prod         # Start prod
make docker-test         # Run tests in Docker
make docker-down         # Stop development
make docker-logs         # Follow logs
make docker-clean        # Remove containers + volumes + images

# === Local Development (Without Docker) ===
# Requires: Go 1.25+, Node.js 22+, MySQL 9.0, Python 3.11+

cd backend && go run ./cmd/server                    # Backend
cd frontend && npm run dev                           # Frontend
cd services/converter && uv sync && python src/server.py  # Converter
```

### Testing

```bash
make test                # Backend tests (requires MySQL)
make test-all            # All tests (backend + frontend + converter)

cd backend && make test                            # Backend directly
cd frontend && npm test -- --run                   # Frontend (Vitest)
cd frontend && npm run test:e2e                    # E2E (Playwright)
cd services/converter && uv run pytest tests/ -v   # Converter
```

- Backend tests require MySQL (`testutil.SetupTestDB()`)
- Backend test types: unit, security (`*_security_test.go`), integration (`*_integration_test.go`), OpenAPI contract
- Coverage threshold: 80% (`.testcoverage.yml`)

### Lint & Format

```bash
make lint                # Backend linter (golangci-lint v2)
make lint-all            # All linters (backend + frontend + converter)
make fmt                 # Format all code
```

### Database Migrations

```bash
make migrate-up                # Apply migrations
make migrate-down              # Rollback
make migrate-create name=xxx   # Create new migration
```

## Architecture

### Key Patterns

1. **Authentication**: Dual-token JWT (access + refresh) in HttpOnly cookies. CSRF via double-submit cookie pattern. OAuth (GitHub) supported.

2. **Repository Pattern**: Services depend on repository interfaces, not implementations. Enables mock-based testing.

3. **Database**: MySQL 9.0 only. Pure Go build (`CGO_ENABLED=0`), no CGO dependency. Docker Compose provides MySQL.

4. **Go Modules**: Main module `github.com/khalily/oreader` (`backend/go.mod`). Proto code in separate module `github.com/khalily/oreader/proto/go` (`proto/go/go.mod`), linked via `replace` directive.

5. **API Proxy**: Vite proxies `/auth`, `/api`, `/health` to backend in dev. Nginx handles proxying in prod. Configurable via `VITE_API_TARGET`.

6. **Frontend State**: Zustand for client state. TanStack Query for server state via custom hooks. react-router-dom for routing.

7. **Background Worker**: `internal/worker/` runs feed refresh as goroutine with configurable interval (`REFRESH_INTERVAL`).

### Docker Services

| Service | Dev | Prod | Test | Port |
|---------|-----|------|------|------|
| backend | air hot-reload | static binary | go test -race | 8080 |
| frontend | Vite dev server | nginx | — | 5173 (dev) / 80 (prod) |
| converter | gRPC server | gRPC server | gRPC server | 50051 |
| db (MySQL 9.0) | required | required | required | 3306 |
| Redis 7 | — | rate limiting | — | 6379 |
| phpMyAdmin | `--profile tools` | — | — | 8081 |

### CI/CD Pipeline

All workflows read versions from `versions.env` via `scripts/ci-load-versions.sh`.

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `ci.yml` | Push/PR any branch | load-versions → version-drift → openapi-lint → openapi-test → backend-lint → backend-test → frontend-lint → frontend-test → converter-lint → converter-test → docker-build → security-scan (govulncheck + npm audit) |
| `integration.yml` | PR with path filters | Docker Compose integration test |
| `deploy.yml` | Push to main/master, tags | Build & push images to GHCR → deploy staging/production |

### Version Management

`versions.env` is the single source of truth for tool versions. Propagation:

```
versions.env → Makefile (include) → docker compose (env vars)
             → Dockerfiles (ARG defaults)
             → CI workflows (scripts/ci-load-versions.sh)
```

Run `make check-versions` to verify consistency across all config files.

### OpenAPI

- Spec: `docs/openapi.yaml` (OpenAPI 3.1)
- Lint: `.redocly.yaml` — CI runs `openapi-lint` job
- Contract tests: `backend/internal/testutil/openapi_contract_test.go` validates routes against spec

## Papers Feature

Upload PDF academic papers, auto-convert to Markdown, and extract metadata via Python gRPC converter.

**Flow**: Upload PDF → Go handler saves to disk → goroutine calls gRPC Convert() → MinerU parses + LLM refines → gRPC ExtractMetadata() → update record → frontend polls status.

### Key Environment Variables

| Component | Variable | Default | Description |
|-----------|----------|---------|-------------|
| Go backend | `PAPER_GRPC_ADDR` | `localhost:50051` | Converter address (Docker: `converter:50051`) |
| Go backend | `PAPER_UPLOAD_DIR` | `uploads/papers` | PDF storage directory |
| Go backend | `PAPER_MAX_UPLOAD_SIZE` | `52428800` (50MB) | Max upload size |
| Go backend | `PAPER_GRPC_TIMEOUT` | `5m` | gRPC call timeout |
| Converter | `LLM_API_KEY` | — | OpenAI-compatible key (optional, skips LLM if unset) |
| Converter | `LLM_BASE_URL` | OpenAI | API base URL |
| Converter | `LLM_MODEL` | `gpt-4o-mini` | Model for Markdown refinement |
| Converter | `GRPC_PORT` | `50051` | Server port |

## Markdown Rendering

- Backend (`internal/infra/markdown/converter.go`): HTML → Markdown with LaTeX support (`\(...\)` → `$...$`, `\[...\]` → `$$...$$`). Language identifiers are NOT preserved in code blocks.
- Frontend (`MarkdownRenderer.tsx`): react-markdown v10 with `MarkdownHooks` (required for async Shiki). Plugins: `remarkGfm`, `remarkMath`, `rehypeKatex`, `rehypeAddDefaultLang`, `rehypeShiki`.
- **Critical**: Plugin arrays MUST be module-level constants to prevent infinite re-processing.

## Known Gotchas

- **Go tests require MySQL**: `testutil.SetupTestDB()` connects to real MySQL. Use `make docker-test` or local MySQL.
- **MinerU API v1.3.12+**: `magic_pdf.pipe` no longer exists. Use `PymuDocDataset` and `read_api`.
- **Converter needs `magic-pdf.json`**: MinerU expects `~/magic-pdf.json`. Docker image generates default CPU-mode config.
- **Vite proxy target**: Local dev `http://localhost:8080`, Docker Compose `http://backend:8080`.
- **Dependabot paths**: Uses monorepo structure — Go `backend/`, npm `frontend/`, pip `services/converter/`.
