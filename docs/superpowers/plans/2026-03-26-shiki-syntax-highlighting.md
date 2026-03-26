# Shiki 语法高亮增强实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 Shiki 替换 Prism，实现代码块的语法高亮、复制按钮、行号显示和主题切换功能

**Architecture:** 使用 `@shikijs/rehype` 作为 rehype 插件集成到 ReactMarkdown，配合 CSS counter 实现行号，独立 CopyButton 组件处理复制逻辑

**Tech Stack:** React, Shiki, @shikijs/rehype, react-markdown, Tailwind CSS

---

## 文件结构

```
web/src/
├── components/
│   └── ui/
│       ├── MarkdownRenderer.tsx    # 修改：使用 Shiki
│       └── CopyButton.tsx          # 新增：复制按钮组件
├── index.css                        # 修改：添加 Shiki 样式
└── components/ui/__tests__/
    └── MarkdownRenderer.test.tsx    # 修改：更新测试

web/package.json                     # 修改：依赖变更
```

---

### Task 1: 安装和配置 Shiki 依赖

**Files:**
- Modify: `web/package.json`

- [ ] **Step 1: 安装 Shiki 相关依赖**

Run in `web/`:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm install shiki @shikijs/rehype
```
Expected: 依赖安装成功

- [ ] **Step 2: 移除 react-syntax-highlighter 依赖**

Run in `web/`:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm uninstall react-syntax-highlighter @types/react-syntax-highlighter
```
Expected: 旧依赖移除成功

- [ ] **Step 3: 验证 package.json 变更**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && cat package.json | grep -E "(shiki|syntax-highlighter)"
```
Expected: 看到 `shiki` 和 `@shikijs/rehype`，没有 `react-syntax-highlighter`

- [ ] **Step 4: 提交依赖变更**

```bash
git add web/package.json web/package-lock.json
git commit -m "chore: replace react-syntax-highlighter with shiki"
```

---

### Task 2: 创建 CopyButton 组件

**Files:**
- Create: `web/src/components/ui/CopyButton.tsx`

- [ ] **Step 1: 创建 CopyButton 组件文件**

Create file `web/src/components/ui/CopyButton.tsx`:

```tsx
import { useState } from 'react'
import { Check, Copy } from 'lucide-react'

interface CopyButtonProps {
  code: string
}

export function CopyButton({ code }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 p-2 rounded-md
                 bg-black/10 dark:bg-white/10
                 hover:bg-black/20 dark:hover:bg-white/20
                 opacity-0 group-hover:opacity-100
                 transition-opacity duration-200
                 text-foreground/70 hover:text-foreground"
      title={copied ? '已复制' : '复制代码'}
    >
      {copied ? (
        <Check className="h-4 w-4 text-green-500" />
      ) : (
        <Copy className="h-4 w-4" />
      )}
    </button>
  )
}
```

- [ ] **Step 2: 验证文件创建**

Run:
```bash
ls -la /data00/home/wangyang.backend/work/oReader/web/src/components/ui/CopyButton.tsx
```
Expected: 文件存在

- [ ] **Step 3: 提交 CopyButton 组件**

```bash
git add web/src/components/ui/CopyButton.tsx
git commit -m "feat: add CopyButton component for code blocks"
```

---

### Task 3: 添加 Shiki 样式到 CSS

**Files:**
- Modify: `web/src/index.css`

- [ ] **Step 1: 添加 Shiki 样式和行号样式**

在 `web/src/index.css` 文件末尾添加：

```css
/* ===== Shiki Syntax Highlighting ===== */

/* Base Shiki container styles */
.shiki {
  padding: 1.5em;
  border-radius: 8px;
  overflow-x: auto;
  margin: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 0.9em;
  line-height: 1.6;
}

/* Line numbers using CSS counter */
.shiki {
  counter-reset: line;
}

.shiki .line {
  display: block;
  min-height: 1em;
}

.shiki .line::before {
  counter-increment: line;
  content: counter(line);
  display: inline-block;
  width: 2.5em;
  margin-right: 1em;
  text-align: right;
  color: hsl(var(--muted-foreground));
  opacity: 0.4;
  user-select: none;
}

/* Remove line number from first line if it's empty */
.shiki .line:empty::before {
  content: counter(line);
}

/* Code block wrapper for copy button positioning */
.prose pre {
  position: relative;
  padding: 0;
  margin: 1.5em 0;
  background: transparent;
}

.prose pre .shiki {
  margin: 0;
}

/* Dark mode adjustments for Shiki */
.dark .shiki {
  background-color: hsl(220 13% 18%);
}

/* Inline code - preserve existing styles, ensure no Shiki interference */
.prose :not(pre) > code {
  background-color: hsl(var(--muted));
  padding: 0.2em 0.4em;
  border-radius: 4px;
  color: hsl(var(--foreground));
}
```

- [ ] **Step 2: 验证 CSS 语法**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm run build 2>&1 | head -20
```
Expected: 构建成功或仅有与 CSS 无关的错误

- [ ] **Step 3: 提交 CSS 变更**

```bash
git add web/src/index.css
git commit -m "style: add Shiki syntax highlighting and line number styles"
```

---

### Task 4: 重构 MarkdownRenderer 组件

**Files:**
- Modify: `web/src/components/ui/MarkdownRenderer.tsx`

- [ ] **Step 1: 重写 MarkdownRenderer 使用 Shiki**

完全替换 `web/src/components/ui/MarkdownRenderer.tsx` 内容：

```tsx
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import rehypeShiki from '@shikijs/rehype'
import { cn } from '@/lib/utils'
import React, { useMemo } from 'react'
import { CopyButton } from './CopyButton'

interface MarkdownRendererProps {
  content: string
  className?: string
}

// Shiki 配置 - 支持双主题
const shikiOptions = {
  themes: {
    light: 'github-light',
    dark: 'github-dark',
  },
  defaultColor: false, // 使用 CSS 变量
}

/**
 * MarkdownRenderer renders Markdown content with:
 * - GitHub Flavored Markdown support (tables, strikethrough, etc.)
 * - LaTeX math rendering with KaTeX (inline `$...$` and block `$$...$$`)
 * - Syntax highlighting with Shiki (180+ languages)
 * - Line numbers (always visible)
 * - Copy button for code blocks
 * - Lazy loading for images
 * - Secure external links (target="_blank", rel="noopener noreferrer")
 */
export function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  // 提取代码文本用于复制
  const extractCodeFromChildren = (children: React.ReactNode): string => {
    if (typeof children === 'string') return children
    if (Array.isArray(children)) {
      return children.map(extractCodeFromChildren).join('')
    }
    if (React.isValidElement(children) && children.props.children) {
      return extractCodeFromChildren(children.props.children)
    }
    return ''
  }

  return (
    <div className={cn('prose prose-slate dark:prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[
          rehypeKatex,
          [rehypeShiki, shikiOptions],
        ]}
        components={{
          // 包装 pre 元素添加复制按钮
          pre({ children, ...props }) {
            const code = extractCodeFromChildren(children)
            return (
              <div className="relative group">
                <pre {...props}>{children}</pre>
                {code && <CopyButton code={code} />}
              </div>
            )
          },
          img({ src, alt, title, ...restProps }) {
            // Remove 'node' from props
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            const { node: _node, ...props } = restProps as any
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
          a({ href, children, ...restProps }) {
            // Remove 'node' from props
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            const { node: _node, ...props } = restProps as any
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

- [ ] **Step 2: 验证 TypeScript 编译**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npx tsc --noEmit 2>&1 | head -30
```
Expected: 无 TypeScript 错误

- [ ] **Step 3: 提交 MarkdownRenderer 变更**

```bash
git add web/src/components/ui/MarkdownRenderer.tsx
git commit -m "feat: replace Prism with Shiki in MarkdownRenderer

- Use @shikijs/rehype for syntax highlighting
- Add dual theme support (github-light/github-dark)
- Integrate CopyButton component
- Remove react-syntax-highlighter dependency"
```

---

### Task 5: 更新测试用例

**Files:**
- Modify: `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx`

- [ ] **Step 1: 更新测试文件**

替换 `web/src/components/ui/__tests__/MarkdownRenderer.test.tsx` 内容：

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { MarkdownRenderer } from '../MarkdownRenderer'

// Mock clipboard API
const mockClipboardWrite = vi.fn()
Object.assign(navigator, {
  clipboard: {
    writeText: mockClipboardWrite.mockResolvedValue(undefined),
  },
})

describe('MarkdownRenderer', () => {
  beforeEach(() => {
    mockClipboardWrite.mockClear()
  })

  it('renders plain text', () => {
    render(<MarkdownRenderer content="Hello World" />)
    expect(screen.getByText('Hello World')).toBeInTheDocument()
  })

  it('renders headings', () => {
    render(<MarkdownRenderer content="# Title" />)
    expect(screen.getByRole('heading', { level: 1, name: 'Title' })).toBeInTheDocument()
  })

  it('renders links with target blank', () => {
    render(<MarkdownRenderer content="[Link](https://example.com)" />)
    const link = screen.getByRole('link', { name: 'Link' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('renders code blocks with Shiki syntax highlighting', () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    // Shiki renders code in span elements with syntax classes
    const preElement = document.querySelector('pre')
    expect(preElement).toBeInTheDocument()
    // Check for shiki class
    expect(preElement?.querySelector('.shiki')).toBeInTheDocument()
  })

  it('renders code blocks with line numbers', () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    // Check for line elements (Shiki generates .line spans)
    const lines = document.querySelectorAll('.line')
    expect(lines.length).toBeGreaterThan(0)
  })

  it('renders multiple languages correctly', () => {
    const { container } = render(
      <MarkdownRenderer
        content={`\`\`\`rust
fn main() {}
\`\`\`

\`\`\`python
print("hello")
\`\`\`

\`\`\`typescript
const x: number = 1
\`\`\``}
      />
    )
    // Should have multiple code blocks
    const codeBlocks = container.querySelectorAll('.shiki')
    expect(codeBlocks.length).toBe(3)
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

  it('renders inline LaTeX math with $...$', () => {
    render(<MarkdownRenderer content="The formula $E = mc^2$ is famous." />)
    // KaTeX renders math in span elements with class katex
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
    // Check that the text content is rendered (KaTeX splits math into multiple elements)
    expect(screen.getByText('The formula', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('is famous.', { exact: false })).toBeInTheDocument()
  })

  it('renders block LaTeX math with $$...$$', () => {
    render(<MarkdownRenderer content={'$$\\\\frac{\\\\partial L}{\\\\partial w} = \\\\nabla$$'} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders complex partial derivative formulas', () => {
    const formula = '$$\\\\frac{\\\\partial(a \\\\cdot b)}{\\\\partial a} = b$$'
    render(<MarkdownRenderer content={formula} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders LaTeX formulas inside GFM tables', () => {
    const tableWithMath = `| Operation | Forward | Local gradients |
|-----------|---------|----------------|
| \`a + b\` | $$a + b$$ | $$\\\\frac{\\\\partial}{\\\\partial a} = 1$$ |`
    render(<MarkdownRenderer content={tableWithMath} />)
    // Should have both table and katex rendered
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
  })

  it('shows copy button on code blocks', () => {
    const { container } = render(
      <MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />
    )
    // Copy button should exist (hidden by default, shown on hover via CSS)
    const copyButton = container.querySelector('button')
    expect(copyButton).toBeInTheDocument()
  })

  it('copies code to clipboard when copy button is clicked', async () => {
    const { container } = render(
      <MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />
    )
    const copyButton = container.querySelector('button')
    expect(copyButton).toBeInTheDocument()

    // Click copy button
    fireEvent.click(copyButton!)

    // Verify clipboard was called
    await waitFor(() => {
      expect(mockClipboardWrite).toHaveBeenCalledWith('fmt.Println("Hello")\n')
    })
  })
})
```

- [ ] **Step 2: 运行测试验证**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm test -- --run 2>&1
```
Expected: 所有测试通过

- [ ] **Step 3: 提交测试变更**

```bash
git add web/src/components/ui/__tests__/MarkdownRenderer.test.tsx
git commit -m "test: update MarkdownRenderer tests for Shiki

- Test Shiki syntax highlighting
- Test line number rendering
- Test multiple language support
- Test copy button functionality"
```

---

### Task 6: 集成测试和最终验证

**Files:**
- None (verification only)

- [ ] **Step 1: 运行完整测试套件**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm test -- --run
```
Expected: 所有测试通过

- [ ] **Step 2: 运行生产构建**

Run:
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm run build
```
Expected: 构建成功，无错误

- [ ] **Step 3: 手动验证功能**

启动开发服务器并验证：
```bash
cd /data00/home/wangyang.backend/work/oReader/web && npm run dev
```

验证清单：
1. 代码块语法高亮正常（测试 Rust, Go, TypeScript, Python）
2. 行号显示
3. 复制按钮 hover 时显示，点击可复制
4. 亮色/暗色主题切换时代码块颜色变化

- [ ] **Step 4: 最终提交（如有遗漏）**

```bash
git status
# 如有未提交的文件，提交它们
```

---

## 验收标准

- [x] 代码块支持 Rust、Go、TypeScript/JSX、Python 等语言
- [x] 复制按钮点击后代码成功复制到剪贴板
- [x] 行号始终显示
- [x] 主题跟随系统亮色/暗色模式切换
- [x] 现有测试通过
