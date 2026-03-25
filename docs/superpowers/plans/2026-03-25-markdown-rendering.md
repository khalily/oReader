# Markdown 渲染优化实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 RSS 文章内容从 HTML 存储和渲染改为 Markdown，提供更一致的阅读体验。

**Architecture:** 后端使用 html-to-markdown 库在 RSS 抓取时转换内容；前端使用 react-markdown 渲染，配合 react-syntax-highlighter 做代码高亮。

**Tech Stack:** Go (html-to-markdown), React (react-markdown, remark-gfm, react-syntax-highlighter)

---

## 文件结构

```
internal/infra/markdown/
├── converter.go          # HTML → Markdown 转换器
└── converter_test.go     # 单元测试

internal/infra/rss/
└── rss.go                # 修改: 调用 Markdown 转换

web/src/components/ui/
└── MarkdownRenderer.tsx  # 新建: Markdown 渲染组件

web/src/components/items/
└── ArticlePanel.tsx      # 修改: 使用 MarkdownRenderer

web/src/pages/items/
└── ItemViewPage.tsx      # 修改: 使用 MarkdownRenderer

web/src/lib/
└── syntax-highlight.ts   # 删除或保留备用

cmd/migrate-to-markdown/
└── main.go               # 数据迁移脚本
```

---

### Task 1: 后端 Markdown 转换模块

**Files:**
- Create: `internal/infra/markdown/converter.go`
- Create: `internal/infra/markdown/converter_test.go`

- [ ] **Step 1: 添加后端依赖**

```bash
cd /data00/home/wangyang.backend/work/oReader
go get github.com/JohannesKaufmann/html-to-markdown
```

Expected: go.mod 更新，添加 html-to-markdown 依赖

- [ ] **Step 2: 编写 converter 测试**

创建 `internal/infra/markdown/converter_test.go`:

```go
package markdown

import (
	"strings"
	"testing"
)

func TestConverter_Convert(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "converts paragraph",
			input:    "<p>Hello World</p>",
			contains: "Hello World",
		},
		{
			name:     "converts heading",
			input:    "<h1>Title</h1>",
			contains: "# Title",
		},
		{
			name:     "converts link",
			input:    `<a href="https://example.com">Link</a>`,
			contains: "[Link](https://example.com)",
		},
		{
			name:     "converts code block",
			input:    "<pre><code>func main() {}</code></pre>",
			contains: "```",
		},
		{
			name:     "converts image",
			input:    `<img src="https://example.com/img.png" alt="test" />`,
			contains: "![test](https://example.com/img.png)",
		},
		{
			name:     "converts list",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: "- Item",
		},
		{
			name:     "converts bold and italic",
			input:    "<strong>bold</strong> <em>italic</em>",
			contains: "**bold**",
		},
		{
			name:     "handles empty input",
			input:    "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Errorf("Convert() error = %v", err)
				return
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Convert() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestConverter_Convert_ComplexHTML(t *testing.T) {
	converter := NewConverter()

	html := `
	<article>
		<h2>Article Title</h2>
		<p>This is a <strong>bold</strong> paragraph with a <a href="https://example.com">link</a>.</p>
		<pre><code class="language-go">fmt.Println("Hello")</code></pre>
		<img src="https://example.com/image.png" alt="An image" />
	</article>
	`

	result, err := converter.Convert(html)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify key conversions
	if !strings.Contains(result, "## Article Title") {
		t.Error("Expected heading to be converted")
	}
	if !strings.Contains(result, "**bold**") {
		t.Error("Expected bold to be converted")
	}
	if !strings.Contains(result, "[link](https://example.com)") {
		t.Error("Expected link to be converted")
	}
	if !strings.Contains(result, "![An image]") {
		t.Error("Expected image to be converted")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./internal/infra/markdown -v
```

Expected: FAIL - package markdown does not exist

- [ ] **Step 4: 实现 converter.go**

创建 `internal/infra/markdown/converter.go`:

```go
// Package markdown provides HTML to Markdown conversion
package markdown

import (
	"github.com/JohannesKaufmann/html-to-markdown"
)

// Converter wraps the html-to-markdown converter
type Converter struct {
	conv *md.Converter
}

// NewConverter creates a new Converter instance with configured options
func NewConverter() *Converter {
	conv := md.NewConverter("", true, &md.Options{
		HeadingStyle:         "atx",      // Use # style headings
		HorizontalRule:       "---",      // Horizontal rule style
		BulletListMarker:     "-",        // Unordered list marker
		CodeBlockStyle:       "fenced",   // Use ``` code blocks
		FencedCodeBlockStyle: "backticks", // Use backticks for code blocks
		EmDelimiter:          "*",        // Italic delimiter
		StrongDelimiter:      "**",       // Bold delimiter
		LinkStyle:            "inlined",  // Inline links
	})

	return &Converter{conv: conv}
}

// Convert converts HTML content to Markdown format
func (c *Converter) Convert(html string) (string, error) {
	if html == "" {
		return "", nil
	}
	return c.conv.ConvertString(html)
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./internal/infra/markdown -v
```

Expected: PASS - all tests green

- [ ] **Step 6: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add go.mod go.sum internal/infra/markdown/
git commit -m "feat(markdown): add HTML to Markdown converter module"
```

---

### Task 2: 修改 RSS 处理流程

**Files:**
- Modify: `internal/infra/rss/rss.go`
- Modify: `internal/infra/rss/rss_test.go`

- [ ] **Step 1: 编写测试 - 验证 Markdown 转换被调用**

修改 `internal/infra/rss/rss_test.go`，添加测试用例:

```go
func TestSanitizeFeed_ConvertsToMarkdown(t *testing.T) {
	feed := &ParsedFeed{
		Title: "Test Feed",
		Link:  "https://example.com",
		Items: []*ParsedItem{
			{
				Title:   "<script>alert('xss')</script>Test Item",
				Content: "<p>This is <strong>bold</strong> content.</p>",
			},
		},
	}

	SanitizeFeed(feed)

	// Verify content was converted to markdown
	if !strings.Contains(feed.Items[0].Content, "**bold**") {
		t.Errorf("Expected content to be converted to markdown, got: %s", feed.Items[0].Content)
	}

	// Verify title was sanitized (script tag removed)
	if strings.Contains(feed.Items[0].Title, "<script>") {
		t.Errorf("Expected script tag to be removed from title, got: %s", feed.Items[0].Title)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./internal/infra/rss -v -run TestSanitizeFeed_ConvertsToMarkdown
```

Expected: FAIL - content still contains HTML tags

- [ ] **Step 3: 修改 rss.go 的 SanitizeFeed 函数**

修改 `internal/infra/rss/rss.go`:

```go
import (
	// ... existing imports
	"oreader/internal/infra/markdown"
)

// SanitizeFeed sanitizes all feed and item content, converting HTML to Markdown
func SanitizeFeed(feed *ParsedFeed) {
	// Create markdown converter
	converter := markdown.NewConverter()

	// Sanitize feed metadata
	feed.Title = sanitize.SanitizeFeedTitle(feed.Title)
	feed.Description = sanitize.SanitizeFeedTitle(feed.Description)

	for _, item := range feed.Items {
		// 1. Fix relative image URLs before conversion
		if item.Content != "" && feed.Link != "" {
			item.Content = FixRelativeImageURLs(item.Content, feed.Link)
		}

		// 2. Convert HTML content to Markdown
		if item.Content != "" {
			mdContent, err := converter.Convert(item.Content)
			if err != nil {
				// On conversion error, keep original content with basic sanitization
				item.Content = sanitize.SanitizeArticleContent(item.Content)
			} else {
				item.Content = mdContent
			}
		}

		// 3. Generate description from Markdown content
		item.Description = generateMarkdownDescription(item.Content)
		item.Title = sanitize.SanitizeFeedTitle(item.Title)
	}
}

// generateMarkdownDescription extracts plain text from Markdown for description
func generateMarkdownDescription(mdContent string) string {
	// Strip Markdown syntax to get plain text
	text := stripMarkdownSyntax(mdContent)
	return truncateContent(text, 200)
}

// stripMarkdownSyntax removes common Markdown syntax characters
func stripMarkdownSyntax(md string) string {
	// Remove headings (#)
	md = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(md, "")
	// Remove emphasis
	md = regexp.MustCompile(`[*_]{1,2}([^*_]+)[*_]{1,2}`).ReplaceAllString(md, "$1")
	// Remove links but keep text
	md = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(md, "$1")
	// Remove images
	md = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`).ReplaceAllString(md, "")
	// Remove code blocks
	md = regexp.MustCompile("```[\\s\\S]*?```").ReplaceAllString(md, "")
	// Remove inline code
	md = regexp.MustCompile("`([^`]+)`").ReplaceAllString(md, "$1")
	// Remove blockquotes
	md = regexp.MustCompile(`(?m)^>\s+`).ReplaceAllString(md, "")
	// Remove list markers
	md = regexp.MustCompile(`(?m)^[-*+]\s+`).ReplaceAllString(md, "")
	md = regexp.MustCompile(`(?m)^\d+\.\s+`).ReplaceAllString(md, "")
	// Remove horizontal rules
	md = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`).ReplaceAllString(md, "")

	return strings.TrimSpace(md)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./internal/infra/rss -v
```

Expected: PASS - all tests green

- [ ] **Step 5: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add internal/infra/rss/
git commit -m "feat(rss): convert article content to Markdown in SanitizeFeed"
```

---

### Task 3: 前端 Markdown 渲染组件

**Files:**
- Create: `web/src/components/ui/MarkdownRenderer.tsx`
- Create: `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx`

- [ ] **Step 1: 安装前端依赖**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm install react-markdown remark-gfm react-syntax-highlighter
npm install -D @types/react-syntax-highlighter
```

Expected: package.json updated with new dependencies

- [ ] **Step 2: 编写 MarkdownRenderer 测试**

创建 `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MarkdownRenderer } from '../MarkdownRenderer'

describe('MarkdownRenderer', () => {
  it('renders plain text', () => {
    render(<MarkdownRenderer content="Hello World" />)
    expect(screen.getByText('Hello World')).toBeInTheDocument()
  })

  it('renders headings', () => {
    render(<MarkdownRenderer content="# Title" />)
    expect(screen.getByRole('heading', { level: 1, name: 'Title' })).toBeInTheDocument()
  })

  it('renders bold text', () => {
    render(<MarkdownRenderer content="**bold text**" />)
    expect(screen.getByText('bold text')).toHaveStyle({ fontWeight: 'bold' })
  })

  it('renders links with target blank', () => {
    render(<MarkdownRenderer content="[Link](https://example.com)" />)
    const link = screen.getByRole('link', { name: 'Link' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('renders code blocks with syntax highlighting', () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    expect(screen.getByText(/fmt\.Println/)).toBeInTheDocument()
  })

  it('renders images with lazy loading', () => {
    render(<MarkdownRenderer content="![Alt text](https://example.com/img.png)" />)
    const img = screen.getByRole('img', { name: 'Alt text' })
    expect(img).toHaveAttribute('loading', 'lazy')
  })

  it('applies custom className', () => {
    const { container } = render(<MarkdownRenderer content="test" className="custom-class" />)
    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('renders GFM tables', () => {
    render(
      <MarkdownRenderer
        content={`| A | B |
|---|---|
| 1 | 2 |`}
      />
    )
    expect(screen.getByRole('table')).toBeInTheDocument()
  })
})
```

- [ ] **Step 3: 运行测试确认失败**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run components/ui/__tests__/MarkdownRenderer.test.tsx
```

Expected: FAIL - MarkdownRenderer component does not exist

- [ ] **Step 4: 实现 MarkdownRenderer 组件**

创建 `web/src/components/ui/MarkdownRenderer.tsx`:

```tsx
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { cn } from '@/lib/utils'

interface MarkdownRendererProps {
  content: string
  className?: string
}

/**
 * MarkdownRenderer renders Markdown content with:
 * - GitHub Flavored Markdown support (tables, strikethrough, etc.)
 * - Syntax highlighting for code blocks
 * - Lazy loading for images
 * - Secure external links (target="_blank", rel="noopener noreferrer")
 */
export function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  return (
    <div className={cn('prose prose-slate dark:prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          code({ className: codeClassName, children, ...props }) {
            const match = /language-(\w+)/.exec(codeClassName || '')
            const language = match ? match[1] : 'text'
            const isInline = !match

            if (isInline) {
              return (
                <code className={codeClassName} {...props}>
                  {children}
                </code>
              )
            }

            return (
              <SyntaxHighlighter
                style={oneDark}
                language={language}
                PreTag="div"
                {...props}
              >
                {String(children).replace(/\n$/, '')}
              </SyntaxHighlighter>
            )
          },
          img({ src, alt, title, ...props }) {
            return (
              <img
                src={src}
                alt={alt || ''}
                title={title}
                loading="lazy"
                className="rounded-lg max-w-full h-auto"
                onError={(e) => {
                  e.currentTarget.src = '/image-placeholder.svg'
                  e.currentTarget.style.opacity = '0.5'
                }}
                {...props}
              />
            )
          },
          a({ href, children, ...props }) {
            return (
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
                {...props}
              >
                {children}
              </a>
            )
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run components/ui/__tests__/MarkdownRenderer.test.tsx
```

Expected: PASS - all tests green

- [ ] **Step 6: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add web/package.json web/package-lock.json web/src/components/ui/MarkdownRenderer.tsx web/src/components/ui/__tests__/MarkdownRenderer.test.tsx
git commit -m "feat(frontend): add MarkdownRenderer component with syntax highlighting"
```

---

### Task 4: 修改 ArticlePanel 组件

**Files:**
- Modify: `web/src/components/items/ArticlePanel.tsx`
- Modify: `web/src/components/items/__tests__/ArticlePanel.test.tsx`

- [ ] **Step 1: 更新 ArticlePanel 测试**

修改 `web/src/components/items/__tests__/ArticlePanel.test.tsx`，将 HTML 内容测试改为 Markdown 内容:

```tsx
// 更新 mock 数据中的 content 为 Markdown 格式
const mockItem = {
  // ... other fields
  content: "# Test Article\n\nThis is **bold** text.",  // Markdown instead of HTML
}

// 更新测试用例
it('renders markdown content', async () => {
  // ... existing test setup
  expect(screen.getByRole('heading', { name: 'Test Article' })).toBeInTheDocument()
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run components/items/__tests__/ArticlePanel.test.tsx
```

Expected: FAIL - component still uses dangerouslySetInnerHTML

- [ ] **Step 3: 修改 ArticlePanel.tsx**

修改 `web/src/components/items/ArticlePanel.tsx`:

1. 添加 import:
```tsx
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
```

2. 移除 `createSafeHTML` 函数（第157-165行）

3. 移除 `articleRef` 和 `processArticleContent` 相关代码（第35行、第83-89行）

4. 替换文章内容渲染（第324-329行）:

```tsx
// 修改前
<article
  ref={articleRef}
  className="prose prose-slate max-w-none dark:prose-invert prose-sm"
  dangerouslySetInnerHTML={createSafeHTML(item.content)}
/>

// 修改后
<MarkdownRenderer content={item.content} className="prose-sm" />
```

5. 移除 syntax-highlight 的 import（第9行）

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run components/items/__tests__/ArticlePanel.test.tsx
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add web/src/components/items/ArticlePanel.tsx web/src/components/items/__tests__/ArticlePanel.test.tsx
git commit -m "refactor(ArticlePanel): use MarkdownRenderer instead of dangerouslySetInnerHTML"
```

---

### Task 5: 修改 ItemViewPage 组件

**Files:**
- Modify: `web/src/pages/items/ItemViewPage.tsx`
- Modify: `web/src/pages/items/__tests__/ItemViewPage.test.tsx`

- [ ] **Step 1: 更新 ItemViewPage 测试**

与 Task 4 类似，更新测试用例使用 Markdown 格式内容。

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run pages/items/__tests__/ItemViewPage.test.tsx
```

- [ ] **Step 3: 修改 ItemViewPage.tsx**

修改 `web/src/pages/items/ItemViewPage.tsx`:

1. 添加 import:
```tsx
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
```

2. 移除 `createSafeHTML` 函数（第127-138行）

3. 移除 `articleRef` 和 `processArticleContent` 相关代码（第29行、第51-55行）

4. 替换文章内容渲染（第258-262行）:

```tsx
// 修改前
<article
  ref={articleRef}
  className="prose prose-slate max-w-none dark:prose-invert"
  dangerouslySetInnerHTML={createSafeHTML(item.content)}
/>

// 修改后
<MarkdownRenderer content={item.content} />
```

5. 移除 syntax-highlight 的 import（第10行）

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run pages/items/__tests__/ItemViewPage.test.tsx
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add web/src/pages/items/ItemViewPage.tsx web/src/pages/items/__tests__/ItemViewPage.test.tsx
git commit -m "refactor(ItemViewPage): use MarkdownRenderer instead of dangerouslySetInnerHTML"
```

---

### Task 6: 清理旧代码

**Files:**
- Delete: `web/src/lib/syntax-highlight.ts` (如果不再需要)

- [ ] **Step 1: 检查 syntax-highlight.ts 是否还有其他引用**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
grep -r "syntax-highlight" src/ --include="*.tsx" --include="*.ts"
```

Expected: No references found (already removed in previous tasks)

- [ ] **Step 2: 删除 syntax-highlight.ts**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
rm src/lib/syntax-highlight.ts
```

- [ ] **Step 3: 运行所有测试确认无破坏**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run
```

Expected: PASS - all tests green

- [ ] **Step 4: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add -A
git commit -m "chore: remove unused syntax-highlight.ts"
```

---

### Task 7: 数据迁移脚本

**Files:**
- Create: `cmd/migrate-to-markdown/main.go`

- [ ] **Step 1: 创建迁移脚本**

创建 `cmd/migrate-to-markdown/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"oreader/internal/config"
	"oreader/internal/infra/markdown"
)

// Item represents a minimal item model for migration
type Item struct {
	ID      string
	Content string
}

func main() {
	// Parse flags
	dryRun := flag.Bool("dry-run", false, "Preview changes without modifying database")
	flag.Parse()

	// Load configuration
	cfg := config.Load()
	db := config.GetDB(cfg)

	converter := markdown.NewConverter()

	log.Println("Starting HTML to Markdown migration...")
	if *dryRun {
		log.Println("DRY RUN MODE: No changes will be made to the database.")
	} else {
		log.Println("WARNING: This will modify all item content in the database.")
		log.Print("Continue? (y/N): ")

		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			log.Println("Migration cancelled.")
			os.Exit(0)
		}
	}

	// Count items to migrate
	var total int64
	db.Table("items").Count(&total)
	log.Printf("Found %d items to process", total)

	// Process in batches
	batchSize := 100
	offset := 0
	processed := 0
	skipped := 0
	failed := 0

	for {
		var items []Item
		result := db.Table("items").
			Select("id, content").
			Where("content IS NOT NULL AND content != ''").
			Offset(offset).
			Limit(batchSize).
			Find(&items)

		if result.Error != nil {
			log.Printf("Error fetching items: %v", result.Error)
			break
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			// Check if already Markdown (simple heuristic)
			if strings.Contains(item.Content, "```") ||
				strings.HasPrefix(item.Content, "#") {
				skipped++
				continue
			}

			mdContent, err := converter.Convert(item.Content)
			if err != nil {
				log.Printf("Failed to convert item %s: %v", item.ID, err)
				failed++
				continue
			}

			if *dryRun {
				log.Printf("[DRY-RUN] Would update item %s (content length: %d -> %d)",
					item.ID, len(item.Content), len(mdContent))
				processed++
			} else {
				result := db.Table("items").Where("id = ?", item.ID).Update("content", mdContent)
				if result.Error != nil {
					log.Printf("Failed to update item %s: %v", item.ID, result.Error)
					failed++
					continue
				}
				processed++
			}
		}

		offset += batchSize
		log.Printf("Processed %d, Skipped %d, Failed %d / %d items...",
			processed, skipped, failed, total)
	}

	log.Printf("Migration complete. Processed: %d, Skipped: %d, Failed: %d",
		processed, skipped, failed)
}
```

- [ ] **Step 2: 提交**

```bash
cd /data00/home/wangyang.backend/work/oReader
git add cmd/migrate-to-markdown/
git commit -m "feat(cmd): add migrate-to-markdown tool for existing data"
```

---

### Task 8: 集成测试和验证

**Files:**
- Modify: `internal/infra/rss/rss_test.go` (添加集成测试)

- [ ] **Step 1: 运行完整后端测试**

```bash
cd /data00/home/wangyang.backend/work/oReader
go test ./... -v
```

Expected: PASS - all tests green

- [ ] **Step 2: 运行完整前端测试**

```bash
cd /data00/home/wangyang.backend/work/oReader/web
npm run test -- --run
```

Expected: PASS - all tests green

- [ ] **Step 3: 手动验证 - 启动开发服务器**

```bash
# Terminal 1: Backend
cd /data00/home/wangyang.backend/work/oReader
make dev

# Terminal 2: Frontend
cd /data00/home/wangyang.backend/work/oReader/web
npm run dev
```

Expected: App runs without errors

- [ ] **Step 4: 手动验证 - 添加新 Feed 并检查内容格式**

1. 访问 http://localhost:5173
2. 添加一个测试 Feed（如 https://feeds.bbci.co.uk/news/rss.xml）
3. 检查文章内容是否以 Markdown 格式渲染

- [ ] **Step 5: 最终提交（如有遗漏）**

```bash
cd /data00/home/wangyang.backend/work/oReader
git status
# Fix any remaining changes
git add -A
git commit -m "test: verify markdown rendering integration"
```

---

## 测试清单

- [ ] 后端: `go test ./internal/infra/markdown -v` - Markdown 转换单元测试
- [ ] 后端: `go test ./internal/infra/rss -v` - RSS 处理测试
- [ ] 后端: `go test ./...` - 全量测试
- [ ] 前端: `npm run test` - 全量测试
- [ ] 集成: 手动验证新 Feed 内容以 Markdown 渲染

## 回滚方案

如果出现问题，可以：

1. **代码回滚**: `git revert HEAD~N` 回退最近的提交
2. **数据库回滚**: 在迁移前备份数据库文件（SQLite）或使用事务（MySQL）
3. **前端回滚**: 恢复 `dangerouslySetInnerHTML` 方式

## 注意事项

1. **react-markdown v9 注意**: `node` 属性已废弃，不要使用
2. **迁移脚本需谨慎**: 在生产环境运行前先在测试环境验证
3. **CSS 样式**: 确保 Tailwind Typography 的 `prose` 类仍然生效
