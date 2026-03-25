# 文章渲染优化设计文档

## 概述

优化 oReader 的文章渲染效果，解决当前排版问题，提升阅读体验。

## 问题分析

### 当前问题

| 问题 | 原因 | 影响 |
|------|------|------|
| 代码块无样式 | 缺少 `@tailwindcss/typography` 插件 | 技术文章难以阅读 |
| 图片无法显示 | RSS 中的相对路径未转换为绝对路径 | 图片区域空白 |
| 段落间距不清 | `prose` 类未生效 | 阅读体验差 |
| 标题样式异常 | Typography 插件缺失 | 层级不明确 |
| 无语法高亮 | 未引入代码高亮库 | 代码可读性差 |

### 根本原因

1. **Typography 插件未安装**：代码使用了 `prose` 类，但 `tailwind.config.js` 中未配置 `@tailwindcss/typography` 插件
2. **后端净化过于严格**：`bluemonday` 不保留 `class` 等属性，导致样式信息丢失
3. **相对路径未处理**：RSS 图片使用相对路径时，未基于源网站 URL 转换

## 解决方案

### 技术选型

- **排版基础**：`@tailwindcss/typography` 插件（prose 类）
- **代码高亮**：Prism.js（支持多种语言，主题可定制）
- **图片处理**：后端修复相对路径 + 前端懒加载 + 错误处理
- **净化策略**：放宽后端净化，保留必要属性

### 架构设计

```
RSS Feed (HTML 内容)
       │
       ▼
┌──────────────────────────────────────┐
│  后端处理 (Go)                        │
│  1. 修复相对路径图片 URL              │
│  2. 增强 HTML 净化（保留 class 等）   │
└──────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│  前端渲染 (React)                     │
│  1. Typography 插件提供基础排版       │
│  2. 自定义 CSS 优化行高/间距          │
│  3. Prism.js 代码语法高亮             │
│  4. 图片懒加载 + 错误处理             │
└──────────────────────────────────────┘
```

## 详细设计

### 1. 前端改动

#### 1.1 依赖安装

```bash
cd web
npm install @tailwindcss/typography
npm install prismjs
npm install -D @types/prismjs
```

#### 1.2 Tailwind 配置

```js
// web/tailwind.config.js
module.exports = {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // 现有配置保持不变
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
```

#### 1.3 ArticlePanel 组件改造

**文件**：`web/src/components/items/ArticlePanel.tsx`

**改动点**：

1. 引入 Prism.js
2. 移除前端二次净化（后端已增强处理）
3. 添加 useEffect 处理代码高亮和图片
4. 优化 prose 类配置

```tsx
// 新增 import
import Prism from 'prismjs'
import 'prismjs/themes/prism-tomorrow.min.css'
// 按需加载语言
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-css'

// 新增 ref
const articleRef = useRef<HTMLElement>(null)

// 新增 useEffect - 内容渲染后处理
useEffect(() => {
  if (!articleRef.current || !item?.content) return

  // 代码高亮
  Prism.highlightAllUnder(articleRef.current)

  // 图片懒加载和错误处理
  articleRef.current.querySelectorAll('img').forEach(img => {
    img.loading = 'lazy'
    img.onerror = () => {
      img.style.display = 'none'
      // 可选：显示占位图
    }
  })
}, [item?.content])

// 简化 createSafeHTML 或直接移除
// 后端已做净化，前端无需二次处理
```

#### 1.4 自定义 Prose 样式

**文件**：`web/src/index.css`

```css
/* 文章阅读体验优化 - 舒适风格 */
@layer components {
  .prose {
    --tw-prose-body: 1.75rem;        /* 行高 */
    --tw-prose-spacing: 1.5em;        /* 段落间距 */
    font-size: 1.0625rem;             /* 基础字号 17px */
    line-height: 1.75;
  }

  .prose p {
    margin-bottom: 1.5em;
  }

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

  .prose img {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 2em auto;
    display: block;
  }

  .prose pre {
    padding: 1.5em;
    border-radius: 8px;
    overflow-x: auto;
    margin: 1.5em 0;
  }

  .prose code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.9em;
  }

  .prose :not(pre) > code {
    background-color: hsl(var(--muted));
    padding: 0.2em 0.4em;
    border-radius: 4px;
  }

  .prose blockquote {
    border-left: 4px solid hsl(var(--border));
    padding-left: 1em;
    margin-left: 0;
    color: hsl(var(--muted-foreground));
    font-style: italic;
  }

  .prose a {
    color: hsl(var(--primary));
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .prose a:hover {
    text-decoration-thickness: 2px;
  }

  .prose ul, .prose ol {
    padding-left: 1.5em;
    margin-bottom: 1.25em;
  }

  .prose li {
    margin-bottom: 0.5em;
  }

  .prose hr {
    margin: 2em 0;
    border-color: hsl(var(--border));
  }

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
```

### 2. 后端改动

#### 2.1 增强 HTML 净化策略

**文件**：`internal/infra/sanitize/sanitize.go`

```go
func init() {
    // 创建 UGC policy
    ugcpolicy = bluemonday.UGCPolicy()

    // 基础元素（保持不变）
    ugcpolicy.AllowElements("p", "br", "hr")
    ugcpolicy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
    ugcpolicy.AllowElements("strong", "b", "em", "i", "u", "s", "strike")
    ugcpolicy.AllowElements("ul", "ol", "li")
    ugcpolicy.AllowElements("blockquote", "pre", "code")
    ugcpolicy.AllowElements("table", "thead", "tbody", "tr", "th", "td")

    // ===== 新增：允许 class 属性（支持代码高亮等） =====
    ugcpolicy.AllowAttrs("class").OnElements(
        "pre", "code", "span", "div",
        "p", "h1", "h2", "h3", "h4", "h5", "h6",
        "table", "thead", "tbody", "tr", "th", "td",
        "blockquote", "ul", "ol", "li",
    )

    // 允许 data-* 属性（某些代码高亮库使用）
    ugcpolicy.AllowAttrsMatching(regexp.MustCompile(`^data-[a-z\-]+$`)).OnElements("pre", "code", "span")

    // 链接属性（保持不变）
    ugcpolicy.AllowAttrs("href").OnElements("a")
    ugcpolicy.AllowAttrs("rel").Matching(regexpSafeRel).OnElements("a")
    ugcpolicy.AllowAttrs("target").Matching(regexp.MustCompile(`^_blank$`)).OnElements("a")

    // ===== 新增：图片属性增强 =====
    ugcpolicy.AllowAttrs("src").OnElements("img")
    ugcpolicy.AllowAttrs("alt").OnElements("img")
    ugcpolicy.AllowAttrs("width", "height").OnElements("img")
    ugcpolicy.AllowAttrs("loading").OnElements("img")           // 懒加载
    ugcpolicy.AllowAttrs("referrerpolicy").OnElements("img")   // 隐私保护
    ugcpolicy.AllowAttrs("class").OnElements("img")            // 样式类

    // ... 其余保持不变
}
```

#### 2.2 相对路径图片修复

**文件**：`internal/infra/rss/rss.go`

```go
import (
    "strings"
    "github.com/PuerkitoBio/goquery"
)

// FixRelativeImageURLs 修复 HTML 内容中的相对路径图片 URL
func FixRelativeImageURLs(htmlContent string, baseURL string) string {
    if htmlContent == "" || baseURL == "" {
        return htmlContent
    }

    // 解析 base URL
    base, err := url.Parse(baseURL)
    if err != nil {
        return htmlContent
    }

    // 使用 goquery 解析 HTML
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
    if err != nil {
        return htmlContent
    }

    // 遍历所有图片，修复相对路径
    doc.Find("img").Each(func(i int, s *goquery.Selection) {
        src, exists := s.Attr("src")
        if !exists || src == "" {
            return
        }

        // 检查是否为相对路径
        if !strings.HasPrefix(src, "http://") &&
           !strings.HasPrefix(src, "https://") &&
           !strings.HasPrefix(src, "data:") {
            // 解析相对路径
            relURL, err := url.Parse(src)
            if err == nil {
                // 拼接为绝对路径
                absURL := base.ResolveReference(relURL)
                s.SetAttr("src", absURL.String())
            }
        }
    })

    // 返回修改后的 HTML
    html, err := doc.Html()
    if err != nil {
        return htmlContent
    }
    return html
}
```

#### 2.3 调用位置

**文件**：`internal/infra/rss/rss.go`

在 `Parse` 函数中，内容净化**之前**调用修复函数：

```go
func (p *Parser) Parse(feedURL string) (*ParsedFeed, error) {
    // ... 现有解析逻辑 ...

    for i, item := range feed.Items {
        content := item.Content
        if content == "" {
            content = item.Description
        }

        // 1. 先修复相对路径图片（使用 feed.Link 作为 baseURL）
        if content != "" && feed.Link != "" {
            content = FixRelativeImageURLs(content, feed.Link)
        }

        // 2. 再进行 HTML 净化
        content = sanitize.SanitizeArticleContent(content)

        // 保存处理后的内容
        parsedItems[i].Content = content
        // ...
    }

    // ...
}
```

#### 2.4 新增依赖

```bash
# 添加 goquery 依赖（用于 HTML 解析）
go get github.com/PuerkitoBio/goquery
```

## 实现清单

### 前端任务

- [ ] 安装 `@tailwindcss/typography` 插件
- [ ] 安装 `prismjs` 及语言包
- [ ] 更新 `tailwind.config.js` 配置
- [ ] 在 `index.css` 中添加自定义 prose 样式
- [ ] 改造 `ArticlePanel.tsx` 组件
- [ ] 改造 `ItemViewPage.tsx` 组件（移动端）

### 后端任务

- [ ] 添加 `goquery` 依赖
- [ ] 实现 `FixRelativeImageURLs` 函数
- [ ] 更新 `sanitize.go` 增强净化策略
- [ ] 在 RSS 解析流程中调用修复函数
- [ ] 添加单元测试

## 测试计划

### 单元测试

1. `sanitize.go` - 验证净化策略允许必要属性
2. `rss.go` - 验证相对路径修复逻辑

### 集成测试

1. 抓取包含相对路径图片的 RSS，验证图片 URL 转换正确
2. 抓取包含代码块的 RSS，验证前端渲染正常
3. 明暗主题切换，验证代码块样式适配

### 手动测试

1. 订阅技术博客 RSS（如 GitHub Blog），验证代码高亮
2. 订阅新闻类 RSS，验证图片显示
3. 检查各类元素（表格、列表、引用）的渲染效果
4. 验证移动端响应式布局

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Prism.js 体积较大 | 首屏加载变慢 | 按需加载语言包，考虑异步加载 |
| 放宽净化策略 | XSS 风险 | 仅允许特定元素的 class 属性，禁止 style 属性 |
| goquery 性能 | RSS 解析变慢 | 仅处理有图片的内容，添加缓存 |

## 验收标准

1. ✅ 代码块显示语法高亮
2. ✅ 图片正确加载（包括相对路径）
3. ✅ 段落间距清晰，阅读舒适
4. ✅ 标题层级明确
5. ✅ 明暗主题正常切换
6. ✅ 移动端渲染正常
