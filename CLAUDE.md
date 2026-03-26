# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is a modern RSS reader with Go backend and React frontend. It supports multiple users, OAuth (GitHub), feed subscriptions, and article management.

### Full Development Environment

```bash
# Run both backend and frontend with one command
./dev.sh
```

## Architecture

### Key Patterns

1. **Authentication**: Dual-token JWT system (access + refresh tokens) stored in HttpOnly cookies. CSRF protection via double-submit cookie pattern.

2. **Repository Pattern**: Services depend on repository interfaces, not concrete implementations. This enables testing with mocks.

3. **API Proxy**: In development, Vite proxies `/api/*` requests to the Go backend. The frontend runs on port 5173, backend on 8080.

4. **OAuth Flow**: GitHub OAuth callback is handled by frontend (`OAuthCallbackPage`) which calls backend with `format=json` to get JSON response instead of 302 redirect.


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

### Frontend (React) - `web/src/components/ui/MarkdownRenderer.tsx`

The frontend renders Markdown to React components:

1. **Must exclude `node` prop** from ReactMarkdown component callbacks:
   ```tsx
   // Wrong - node prop leaks to DOM as [object Object]
   code({ children, ...props }) {
     return <code {...props}>{children}</code>
   }

   // Correct - exclude node prop
   code({ children, node, ...restProps }) {
     const { node: _node, ...props } = restProps as any
     return <code {...props}>{children}</code>
   }
   ```

2. **CSS override for Tailwind Typography**:
   - Tailwind Typography plugin adds backtick decorators to `<code>` elements
   - Override in `index.css`:
   ```css
   .prose :not(pre) > code::before,
   .prose :not(pre) > code::after {
     content: none !important;
   }
   ```

### Testing

- Unit tests: `converter_test.go` (backend), `MarkdownRenderer.styles.test.tsx` (frontend)
- CSS pseudo-elements (`::before`/`::after`) cannot be tested in jsdom - require E2E tests
