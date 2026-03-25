# Article Rendering Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize article rendering in oReader to provide comfortable reading experience with syntax highlighting, proper image display, and clean typography.

**Architecture:** Backend fixes relative image URLs and enhances HTML sanitization to preserve class attributes. Frontend adds Tailwind Typography plugin for base prose styling, Prism.js for code highlighting, and custom CSS for reading comfort.

**Tech Stack:** Go (bluemonday, goquery), React, Tailwind CSS, Prism.js

---

## File Structure

### Backend Files
- **Modify**: `internal/infra/sanitize/sanitize.go` - Enhanced HTML sanitization with class whitelist
- **Modify**: `internal/infra/sanitize/sanitize_test.go` - Add tests for class attribute handling
- **Modify**: `internal/infra/rss/rss.go` - Add relative URL fixing in SanitizeFeed
- **Create**: `internal/infra/rss/image_url_fixer.go` - Relative image URL fixing logic
- **Create**: `internal/infra/rss/image_url_fixer_test.go` - Unit tests for URL fixing

### Frontend Files
- **Modify**: `web/tailwind.config.js` - Add typography plugin
- **Modify**: `web/src/index.css` - Add custom prose styles
- **Modify**: `web/src/components/items/ArticlePanel.tsx` - Add Prism.js, image handling
- **Modify**: `web/src/pages/items/ItemViewPage.tsx` - Mobile article rendering
- **Create**: `web/src/lib/syntax-highlight.ts` - Prism.js initialization helper
- **Create**: `web/public/image-placeholder.svg` - Placeholder image for failed loads

### Dependencies
- **Backend**: `github.com/PuerkitoBio/goquery`
- **Frontend**: `@tailwindcss/typography`, `prismjs@1.29.0`

---

## Task 1: Backend - Add goquery Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add goquery dependency**

```bash
cd /data00/home/wangyang.backend/work/oReader
go get github.com/PuerkitoBio/goquery
```

Run: `go mod tidy`
Expected: Dependencies resolved successfully

- [ ] **Step 2: Verify dependency**

Run: `go list -m github.com/PuerkitoBio/goquery`
Expected: Version printed (e.g., `github.com/PuerkitoBio/goquery v1.8.1`)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add goquery for HTML parsing"
```

---

## Task 2: Backend - Write Tests for Relative URL Fixing

**Files:**
- Create: `internal/infra/rss/image_url_fixer_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package rss

import (
	"strings"
	"testing"
)

func TestFixRelativeImageURLs(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		baseURL        string
		wantContains   string
		wantNotContains string
	}{
		{
			name:         "fix relative path",
			html:         `<img src="/images/photo.jpg">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="https://example.com/images/photo.jpg"`,
		},
		{
			name:         "preserve absolute URL",
			html:         `<img src="https://other.com/img.png">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="https://other.com/img.png"`,
		},
		{
			name:         "preserve data URI",
			html:         `<img src="data:image/png;base64,abc">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="data:image/png;base64,abc"`,
		},
		{
			name:         "fix relative with subdirectory",
			html:         `<img src="../images/photo.jpg">`,
			baseURL:      "https://example.com/blog/article",
			wantContains: `src="https://example.com/images/photo.jpg"`,
		},
		{
			name:         "empty content returns unchanged",
			html:         "",
			baseURL:      "https://example.com",
			wantContains: "",
		},
		{
			name:         "empty baseURL returns unchanged",
			html:         `<img src="/img.jpg">`,
			baseURL:      "",
			wantContains: `<img src="/img.jpg">`,
		},
		{
			name:            "multiple images - all fixed",
			html:            `<img src="/a.jpg"><img src="https://other.com/b.jpg"><img src="/c.jpg">`,
			baseURL:         "https://example.com",
			wantContains:    `src="https://example.com/a.jpg"`,
			wantNotContains: `src="/a.jpg"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FixRelativeImageURLs(tt.html, tt.baseURL)

			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("FixRelativeImageURLs() = %q, want to contain %q", got, tt.wantContains)
			}
			if tt.wantNotContains != "" && strings.Contains(got, tt.wantNotContains) {
				t.Errorf("FixRelativeImageURLs() = %q, should NOT contain %q", got, tt.wantNotContains)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infra/rss -run TestFixRelativeImageURLs -v`
Expected: FAIL - function not defined

- [ ] **Step 3: Commit test file**

```bash
git add internal/infra/rss/image_url_fixer_test.go
git commit -m "test(rss): add tests for relative image URL fixing"
```

---

## Task 3: Backend - Implement Relative URL Fixing

**Files:**
- Create: `internal/infra/rss/image_url_fixer.go`

- [ ] **Step 1: Implement FixRelativeImageURLs function**

```go
package rss

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"oreader/internal/infra/logger"
)

// FixRelativeImageURLs fixes relative image URLs in HTML content by converting them to absolute URLs.
// This ensures images from RSS feeds display correctly regardless of their original path format.
func FixRelativeImageURLs(htmlContent string, baseURL string) string {
	if htmlContent == "" || baseURL == "" {
		return htmlContent
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		logger.Debug().Err(err).Str("base_url", baseURL).Msg("Failed to parse base URL")
		return htmlContent
	}

	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to parse HTML for image URL fixing")
		return htmlContent
	}

	// Track if any changes were made
	modified := false

	// Find all images and fix relative paths
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		// Skip absolute URLs and data URIs
		if strings.HasPrefix(src, "http://") ||
			strings.HasPrefix(src, "https://") ||
			strings.HasPrefix(src, "data:") {
			return
		}

		// Parse relative URL
		relURL, err := url.Parse(src)
		if err != nil {
			return
		}

		// Resolve to absolute URL
		absURL := base.ResolveReference(relURL)
		s.SetAttr("src", absURL.String())
		modified = true
	})

	// If no changes, return original content
	if !modified {
		return htmlContent
	}

	// Return modified HTML
	html, err := doc.Html()
	if err != nil {
		return htmlContent
	}
	return html
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/infra/rss -run TestFixRelativeImageURLs -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/infra/rss/image_url_fixer.go
git commit -m "feat(rss): implement relative image URL fixing"
```

---

## Task 4: Backend - Update SanitizeFeed to Use URL Fixer

**Files:**
- Modify: `internal/infra/rss/rss.go`

- [ ] **Step 1: Update SanitizeFeed function**

Find the `SanitizeFeed` function (around line 189-199) and modify it:

```go
// SanitizeFeed sanitizes all feed and item content
func SanitizeFeed(feed *ParsedFeed) {
	feed.Title = sanitize.SanitizeFeedTitle(feed.Title)
	feed.Description = sanitize.SanitizeFeedTitle(feed.Description)

	for _, item := range feed.Items {
		// 1. Fix relative image URLs before sanitization (using feed.Link as baseURL)
		if item.Content != "" && feed.Link != "" {
			item.Content = FixRelativeImageURLs(item.Content, feed.Link)
		}

		// 2. Generate description from content
		item.Description = sanitize.GenerateDescription(item.Content)

		// 3. Sanitize content
		item.Title = sanitize.SanitizeFeedTitle(item.Title)
		item.Content = sanitize.SanitizeArticleContent(item.Content)
	}
}
```

- [ ] **Step 2: Run all RSS tests**

Run: `go test ./internal/infra/rss -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/infra/rss/rss.go
git commit -m "feat(rss): integrate image URL fixing into SanitizeFeed"
```

---

## Task 5: Backend - Write Sanitization Enhancement Tests

**Files:**
- Modify: `internal/infra/sanitize/sanitize_test.go` (EXISTS - append tests)

- [ ] **Step 1: Read existing test file to find insertion point**

Run: `cat internal/infra/sanitize/sanitize_test.go | tail -20`
Note: Append new tests at the end of the existing file

- [ ] **Step 2: Add tests for class attribute handling**

Append to `internal/infra/sanitize/sanitize_test.go`:

```go

// ===== Tests for enhanced sanitization (class attributes) =====

func TestSanitizeArticleContent_ClassAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeep string
		wantDrop string
	}{
		{
			name:     "preserve language class on pre",
			input:    `<pre class="language-go">code</pre>`,
			wantKeep: `class="language-go"`,
		},
		{
			name:     "preserve token class on span",
			input:    `<span class="token keyword">func</span>`,
			wantKeep: `class="token keyword"`,
		},
		{
			name:     "drop malicious class",
			input:    `<div class="onclick-alert">text</div>`,
			wantDrop: `onclick`,
		},
		{
			name:     "preserve prose class",
			input:    `<p class="prose-lg">text</p>`,
			wantKeep: `prose`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeArticleContent(tt.input)

			if tt.wantKeep != "" && !strings.Contains(got, tt.wantKeep) {
				t.Errorf("Expected to keep %q, got %q", tt.wantKeep, got)
			}
			if tt.wantDrop != "" && strings.Contains(got, tt.wantDrop) {
				t.Errorf("Expected to drop %q, got %q", tt.wantDrop, got)
			}
		})
	}
}

func TestSanitizeArticleContent_ImageAttributes(t *testing.T) {
	input := `<img src="https://example.com/img.jpg" alt="test" loading="lazy" referrerpolicy="no-referrer">`
	got := SanitizeArticleContent(input)

	if !strings.Contains(got, `loading="lazy"`) {
		t.Error("Expected loading attribute to be preserved")
	}
	if !strings.Contains(got, `referrerpolicy="no-referrer"`) {
		t.Error("Expected referrerpolicy attribute to be preserved")
	}
}

func TestSanitizeArticleContent_DataAttributes(t *testing.T) {
	input := `<pre data-language="go" data-line="5">code</pre>`
	got := SanitizeArticleContent(input)

	if !strings.Contains(got, `data-language="go"`) {
		t.Error("Expected data-language attribute to be preserved")
	}
}
```

- [ ] **Step 3: Add strings import if not present**

Check if `import "strings"` exists in the file. If not, add it to the imports.

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/infra/sanitize -run TestSanitizeArticleContent_Class -v`
Expected: FAIL - class attributes being stripped

- [ ] **Step 5: Commit**

```bash
git add internal/infra/sanitize/sanitize_test.go
git commit -m "test(sanitize): add tests for enhanced class attribute handling"
```

---

## Task 6: Backend - Enhance HTML Sanitization Policy

**Files:**
- Modify: `internal/infra/sanitize/sanitize.go`

- [ ] **Step 1: Add regex variables for class/data attributes**

Add new regex variables after `regexpSafeRel` (around line 19):

```go
// regexpSafeRel is a simple pattern for safe rel attribute values
var regexpSafeRel = regexp.MustCompile(`(?i)^nofollow\s*(noopener\s*noreferrer?|noopener|noreferrer?)?$|^noopener\s*(noreferrer?|nofollow)?$|^noreferrer?$`)

// regexpSafeClass matches safe CSS class names for syntax highlighting and prose styling
// Security: Only allows whitelisted patterns to prevent CSS injection attacks
var regexpSafeClass = regexp.MustCompile(`^(language-[a-z0-9-]+|token|keyword|string|comment|number|operator|punctuation|function|class-name|builtin|variable|constant|property|tag|attr-name|attr-value|selector|regex|important|bold|italic|underline|highlight-[a-z]+|prose[-\w]*)$`)

// regexpDataAttr matches safe data attribute names
var regexpDataAttr = regexp.MustCompile(`^data-[a-z\-]+$`)
```

- [ ] **Step 2: Add class attribute allowance in init function**

In the `init()` function, after the existing image attributes (around line 40-41), add:

```go
	// Allow images with additional attributes
	ugcpolicy.AllowAttrs("src").OnElements("img")
	ugcpolicy.AllowAttrs("alt").OnElements("img")
	ugcpolicy.AllowAttrs("width", "height").OnElements("img")
	ugcpolicy.AllowAttrs("loading").OnElements("img")           // Lazy loading
	ugcpolicy.AllowAttrs("referrerpolicy").OnElements("img")   // Privacy protection
	ugcpolicy.AllowAttrs("class").OnElements("img")            // Styling

	// ===== NEW: Allow class attributes for code highlighting =====
	// Security: Only allow whitelisted class names via regex
	ugcpolicy.AllowAttrs("class").Matching(regexpSafeClass).OnElements(
		"pre", "code", "span", "div",
		"p", "h1", "h2", "h3", "h4", "h5", "h6",
		"table", "thead", "tbody", "tr", "th", "td",
		"blockquote", "ul", "ol", "li",
	)

	// Allow data-* attributes (used by some syntax highlighters)
	ugcpolicy.AllowAttrsMatching(regexpDataAttr).OnElements("pre", "code", "span")
```

- [ ] **Step 3: Run all sanitize tests**

Run: `go test ./internal/infra/sanitize -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/infra/sanitize/sanitize.go
git commit -m "feat(sanitize): allow class attributes for code highlighting"
```

---

## Task 7: Frontend - Install Dependencies

**Files:**
- Modify: `web/package.json`

- [ ] **Step 1: Install frontend dependencies**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm install @tailwindcss/typography
npm install prismjs@1.29.0
npm install -D @types/prismjs
```

- [ ] **Step 2: Verify installation**

Run: `npm list @tailwindcss/typography prismjs`
Expected: Versions displayed

- [ ] **Step 3: Commit**

```bash
git add web/package.json web/package-lock.json
git commit -m "chore(deps): add typography plugin and prismjs for article rendering"
```

---

## Task 8: Frontend - Configure Tailwind and Add Prose Styles

**Files:**
- Modify: `web/tailwind.config.js`
- Modify: `web/src/index.css`

- [ ] **Step 1: Add typography plugin to tailwind.config.js**

Update `web/tailwind.config.js` - add the plugins array at the end:

```js
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // ... existing color config ...
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
```

- [ ] **Step 2: Add custom prose styles to index.css**

Append to the end of `web/src/index.css`:

```css
/* ===== Article Rendering Optimization ===== */
/* Comfortable reading experience styles */

@layer components {
  /* Base prose configuration */
  .prose {
    --tw-prose-body: 1.75;
    --tw-prose-spacing: 1.5em;
    font-size: 1.0625rem;
    line-height: 1.75;
  }

  /* Paragraph spacing */
  .prose p {
    margin-bottom: 1.5em;
  }

  /* Headings */
  .prose h1, .prose h2, .prose h3,
  .prose h4, .prose h5, .prose h6 {
    margin-top: 2em;
    margin-bottom: 0.75em;
    font-weight: 600;
    line-height: 1.3;
  }

  .prose h1 { font-size: 1.75em; }
  .prose h2 { font-size: 1.5em; }
  .prose h3 { font-size: 1.25em; }

  /* Images */
  .prose img {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 2em auto;
    display: block;
  }

  /* Code blocks */
  .prose pre {
    padding: 1.5em;
    border-radius: 8px;
    overflow-x: auto;
    margin: 1.5em 0;
  }

  .prose code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    font-size: 0.9em;
  }

  /* Inline code */
  .prose :not(pre) > code {
    background-color: hsl(var(--muted));
    padding: 0.2em 0.4em;
    border-radius: 4px;
    color: hsl(var(--foreground));
  }

  /* Blockquotes */
  .prose blockquote {
    border-left: 4px solid hsl(var(--border));
    padding-left: 1em;
    margin-left: 0;
    color: hsl(var(--muted-foreground));
    font-style: italic;
  }

  /* Links */
  .prose a {
    color: hsl(var(--primary));
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .prose a:hover {
    text-decoration-thickness: 2px;
  }

  /* Lists */
  .prose ul, .prose ol {
    padding-left: 1.5em;
    margin-bottom: 1.25em;
  }

  .prose li {
    margin-bottom: 0.5em;
  }

  /* Horizontal rule */
  .prose hr {
    margin: 2em 0;
    border-color: hsl(var(--border));
  }

  /* Tables */
  .prose table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5em 0;
  }

  .prose th, .prose td {
    border: 1px solid hsl(var(--border));
    padding: 0.75em;
    text-align: left;
  }

  .prose th {
    background-color: hsl(var(--muted));
    font-weight: 600;
  }
}

/* Prism.js theme overrides for dark mode */
.dark .prose pre {
  background-color: hsl(220 13% 18%);
}

.dark .prose :not(pre) > code {
  background-color: hsl(220 13% 18%);
}
```

- [ ] **Step 3: Verify Tailwind config builds**

Run: `cd web && npm run build 2>&1 | head -20`
Expected: Build succeeds without errors

- [ ] **Step 4: Commit**

```bash
git add web/tailwind.config.js web/src/index.css
git commit -m "feat(frontend): add typography plugin and custom prose styles"
```

---

## Task 9: Frontend - Create Placeholder Image

**Files:**
- Create: `web/public/image-placeholder.svg`

- [ ] **Step 1: Create placeholder SVG**

```xml
<svg width="200" height="150" viewBox="0 0 200 150" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect width="200" height="150" fill="#E5E7EB"/>
  <rect x="70" y="45" width="60" height="60" rx="4" fill="#9CA3AF"/>
  <circle cx="85" cy="60" r="8" fill="#D1D5DB"/>
  <path d="M75 95 L95 75 L115 85 L125 70" stroke="#D1D5DB" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
  <text x="100" y="125" text-anchor="middle" font-family="system-ui" font-size="10" fill="#6B7280">Image unavailable</text>
</svg>
```

- [ ] **Step 2: Commit**

```bash
git add web/public/image-placeholder.svg
git commit -m "feat(assets): add image placeholder for failed loads"
```

---

## Task 10: Frontend - Create Syntax Highlight Helper

**Files:**
- Create: `web/src/lib/syntax-highlight.ts`

- [ ] **Step 1: Create syntax highlight helper module**

```typescript
// web/src/lib/syntax-highlight.ts
import Prism from 'prismjs'

// Import commonly used language support
// Note: These add ~50KB to bundle. Consider code-splitting if bundle size becomes an issue.
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-jsx'
import 'prismjs/components/prism-tsx'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-markdown'
import 'prismjs/components/prism-css'
import 'prismjs/components/prism-sql'
import 'prismjs/components/prism-diff'

// Import default theme (light)
import 'prismjs/themes/prism.min.css'

/**
 * Highlight all code blocks within a container element
 */
export function highlightCode(container: HTMLElement): void {
  Prism.highlightAllUnder(container)
}

/**
 * Setup lazy image loading and error handling within a container
 */
export function setupImages(container: HTMLElement): void {
  container.querySelectorAll('img').forEach((img) => {
    // Enable native lazy loading
    img.loading = 'lazy'

    // Handle image load errors gracefully
    img.onerror = () => {
      img.src = '/image-placeholder.svg'
      img.alt = 'Image failed to load'
      img.style.opacity = '0.5'
    }
  })
}

/**
 * Process article content after render
 * Handles code highlighting and image optimization
 */
export function processArticleContent(container: HTMLElement): void {
  if (!container) return

  // Apply code highlighting
  highlightCode(container)

  // Setup image handling
  setupImages(container)
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/syntax-highlight.ts
git commit -m "feat(frontend): add syntax highlight helper with Prism.js"
```

---

## Task 11: Frontend - Update ArticlePanel Component

**Files:**
- Modify: `web/src/components/items/ArticlePanel.tsx`

- [ ] **Step 1: Add import for syntax-highlight**

Add to the imports section (around line 6):

```tsx
import { processArticleContent } from '@/lib/syntax-highlight'
```

- [ ] **Step 2: Add article ref**

Find the state declarations (around line 32) and add after them:

```tsx
  // Article content ref for post-processing
  const articleRef = useRef<HTMLElement>(null)
```

Note: `useRef` is already imported from 'react' on line 6.

- [ ] **Step 3: Add content processing useEffect**

Add after the existing useEffects (around line 78):

```tsx
  // Process article content after render (code highlighting, image handling)
  useEffect(() => {
    if (!articleRef.current || !item?.content) return

    // Process content: syntax highlighting + image handling
    processArticleContent(articleRef.current)
  }, [item?.content])
```

- [ ] **Step 4: Simplify createSafeHTML function**

Replace the existing `createSafeHTML` function (around line 146-154) with:

```tsx
  // Simplified frontend sanitization (defense-in-depth backup)
  // Backend performs primary sanitization; this is a safety net
  const createSafeHTML = (html: string | null) => {
    if (!html) return { __html: '' }
    // Only remove the most dangerous patterns as backup
    const sanitized = html
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/on\w+="[^"]*"/gi, '')
      .replace(/javascript:/gi, '')
    return { __html: sanitized }
  }
```

- [ ] **Step 5: Update article element to use ref**

Find the article element (around line 313-316) and add the ref:

```tsx
              <article
                ref={articleRef}
                className="prose prose-slate max-w-none dark:prose-invert"
                dangerouslySetInnerHTML={createSafeHTML(item.content)}
              />
```

- [ ] **Step 6: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add web/src/components/items/ArticlePanel.tsx
git commit -m "feat(ArticlePanel): add code highlighting and image handling"
```

---

## Task 12: Frontend - Update ItemViewPage Component

**Files:**
- Modify: `web/src/pages/items/ItemViewPage.tsx`

- [ ] **Step 1: Add import for syntax-highlight**

Add to the imports section:

```tsx
import { processArticleContent } from '@/lib/syntax-highlight'
```

Note: Check if `useRef` and `useEffect` are already imported. If not, add them to the react import.

- [ ] **Step 2: Add article ref**

Add after existing hooks/state:

```tsx
  // Article content ref for post-processing
  const articleRef = useRef<HTMLElement>(null)
```

- [ ] **Step 3: Add content processing useEffect**

```tsx
  // Process article content after render
  useEffect(() => {
    if (!articleRef.current || !item?.content) return
    processArticleContent(articleRef.current)
  }, [item?.content])
```

- [ ] **Step 4: Update article element with ref**

Find the article element and add the ref:

```tsx
              <article
                ref={articleRef}
                className="prose prose-slate max-w-none dark:prose-invert"
                dangerouslySetInnerHTML={createSafeHTML(item.content)}
              />
```

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/items/ItemViewPage.tsx
git commit -m "feat(ItemViewPage): add code highlighting and image handling"
```

---

## Task 13: Integration Test

- [ ] **Step 1: Run all backend tests**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./... -v
```

Expected: All tests PASS

- [ ] **Step 2: Run frontend build**

```bash
cd web
npm run build
```

Expected: Build succeeds

- [ ] **Step 3: Run frontend type check**

```bash
cd web
npx tsc --noEmit
```

Expected: No errors

- [ ] **Step 4: Manual smoke test**

Start dev server and verify:
1. Code blocks show syntax highlighting
2. Prose styling applied (paragraph spacing, headings)
3. Images load correctly
4. Dark mode works

```bash
make dev
```

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete article rendering optimization

- Add @tailwindcss/typography plugin for prose styling
- Add Prism.js for code syntax highlighting
- Fix relative image URLs in RSS content
- Enhance backend sanitization to preserve class attributes
- Add custom prose styles for comfortable reading"
```

---

## Verification Checklist

After implementation, verify:

- [ ] Code blocks display syntax highlighting for common languages
- [ ] Images load correctly (including relative URLs)
- [ ] Paragraph spacing is clear and comfortable
- [ ] Headings have proper hierarchy
- [ ] Dark mode code theme works
- [ ] Mobile rendering is correct
- [ ] No XSS vulnerabilities (sanitization works)
- [ ] Performance is acceptable (no major lag)

---

## Rollback Instructions

If issues occur:

1. **Quick revert all changes:**
   ```bash
   # Find the first commit of this feature
   git log --oneline -15
   # Revert to the commit before this feature
   git revert --no-commit <first-feature-commit>..HEAD
   git commit -m "revert: article rendering optimization"
   ```

2. **Disable features individually:**
   - Remove Prism imports from components
   - Remove typography plugin from tailwind.config.js
   - Comment out FixRelativeImageURLs call in SanitizeFeed
