# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, and article management.

### Environment Setup

```bash
cp .env.example .env   # Required before first run
```

### Development Commands

```bash
# Full stack (backend + frontend)
./dev.sh

# Backend only (API mode, no frontend build)
make dev

# Frontend only
make frontend-dev

# Go tests
go test ./internal/...

# Go tests with coverage
make test

# Frontend tests
cd web && npm test

# Frontend tests (watch mode)
cd web && npm run test:watch

# Build frontend for production
make frontend-build

# Build production binary (with embedded frontend)
make build

# Lint
make lint                          # Go (golangci-lint)
cd web && npm run lint             # Frontend

# Database migrations
make migrate-create name=xxx       # Create new migration
make migrate-up                    # Apply migrations
make migrate-down                  # Rollback
```

## Architecture

### Directory Structure

```
cmd/
  server/              # Main application entry point
  migrate-to-markdown/ # HTML→Markdown migration tool
internal/
  config/              # Configuration loading
  handler/             # HTTP handlers (controllers)
  infra/markdown/      # Markdown conversion infrastructure
  middleware/           # Auth, CSRF, logging middleware
  model/               # Domain models
  repository/          # Data access layer (SQLite)
  service/             # Business logic layer
  testutil/            # Test helpers
  worker/              # Background feed refresh worker
migrations/            # SQL migration files
web/src/
  components/          # React components (auth, feed, items, ui)
  hooks/               # Custom React hooks (useAuth, useFeeds, useItems, useStats)
  lib/api/             # Axios API client
  pages/               # Page components (auth, items, oauth)
  stores/              # Zustand state stores (authStore, itemsStore)
```

### Key Patterns

1. **Authentication**: Dual-token JWT system (access + refresh tokens) stored in HttpOnly cookies. CSRF protection via double-submit cookie pattern.

2. **Repository Pattern**: Services depend on repository interfaces, not concrete implementations. This enables testing with mocks.

3. **API Proxy**: In development, Vite proxies `/api/*` requests to the Go backend. The frontend runs on port 5173, backend on 8080.

4. **OAuth Flow**: GitHub OAuth callback is handled by frontend (`OAuthCallbackPage`) which calls backend with `format=json` to get JSON response instead of 302 redirect.

5. **State Management**: Zustand stores in `web/src/stores/` manage client-side state (auth, items). TanStack Query handles server state caching and revalidation via custom hooks in `web/src/hooks/`.

6. **API Client**: Centralized Axios instance in `web/src/lib/api/axios.ts` handles request/response interceptors, auth token injection, and error transformation.

7. **Background Worker**: `internal/worker/` implements feed refresh as a background goroutine with configurable interval (`REFRESH_INTERVAL` env var, default 15m).


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
