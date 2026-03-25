# 文章 Markdown 渲染优化设计

## 背景

当前 oReader 的文章展示流程是：
1. 后端抓取 RSS，保存 HTML 内容
2. 前端使用 `dangerouslySetInnerHTML` 直接渲染 HTML
3. 使用 Prism.js 堆代码高亮

**问题：**
- 不同 RSS 源的 HTML 结构不一致，导致渲染效果参差不齐
- HTML 内容可能包含不规范的标签，难以维护
- 代码高亮需要后处理，效率较低

## 目标

将抓取的文章内容转换为 Markdown 格式存储和渲染，提供更一致、更干净的阅读体验。

## 技术方案

### 后端改动

#### 1. 新增 HTML → Markdown 转换模块

**文件：** `internal/infra/markdown/converter.go`

```go
package markdown

import (
    "github.com/JohannesKaufmann/html-to-markdown"
)

// Converter 封装 HTML 到 Markdown 的转换逻辑
type Converter struct {
    conv *md.Converter
}

// NewConverter 创建新的转换器实例
func NewConverter() *Converter {
    conv := md.NewConverter("", true, &md.Options{
        HeadingStyle:             "atx",           // 使用 # 风格标题
        HorizontalRule:           "---",          // 水平分隔线
        BulletListMarker:         "-",            // 无序列表标记
        CodeBlockStyle:           "fenced",       // 使用 ``` 代码块
        EmDelimiter:              "*",            // 斜体
        StrongDelimiter:          "**",           // 粗体
        LinkStyle:                "inlined",      // 内联链接
        LinkReferenceStyle:       "full",         // 完整链接引用
    })

    return &Converter{conv: conv}
}

// Convert 将 HTML 转换为 Markdown
func (c *Converter) Convert(html string) (string, error) {
    return c.conv.ConvertString(html)
}
```

#### 2. 修改 RSS 处理流程

**文件：** `internal/infra/rss/rss.go`

修改 `SanitizeFeed` 函数：

```go
// SanitizeFeed 将所有内容转换为 Markdown
func SanitizeFeed(feed *ParsedFeed) {
    converter := markdown.NewConverter()

    feed.Title = sanitize.SanitizeFeedTitle(feed.Title)
    feed.Description = sanitize.SanitizeFeedTitle(feed.Description)

    for _, item := range feed.Items {
        // 1. 修复相对图片 URL
        if item.Content != "" && feed.Link != "" {
            item.Content = FixRelativeImageURLs(item.Content, feed.Link)
        }

        // 2. 转换为 Markdown
        if item.Content != "" {
            markdownContent, err := converter.Convert(item.Content)
            if err != nil {
                // 转换失败时保留原始内容（做基本安全处理）
                item.Content = sanitize.SanitizeArticleContent(item.Content)
            } else {
                item.Content = markdownContent
            }
        }

        // 3. 生成描述（从 Markdown 提取）
        item.Description = generateMarkdownDescription(item.Content)
        item.Title = sanitize.SanitizeFeedTitle(item.Title)
    }
}

// generateMarkdownDescription 从 Markdown 内容生成摘要
func generateMarkdownDescription(mdContent string) string {
    // 移除 Markdown 语法，提取纯文本
    text := stripMarkdownSyntax(mdContent)
    return truncateText(text, 200)
}
```

### 前端改动

#### 1. 安装依赖

```bash
cd web
npm install react-markdown remark-gfm react-syntax-highlighter
```

#### 2. 创建 Markdown 渲染组件

**文件：** `web/src/components/ui/MarkdownRenderer.tsx`

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

export function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  return (
    <div className={cn('prose prose-slate dark:prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          code({ node, inline, className, children, ...props }) {
            const match = /language-(\w+)/.exec(className || '')
            const language = match ? match[1] : 'text'

            return !inline ? (
              <SyntaxHighlighter
                style={oneDark}
                language={language}
                PreTag="div"
                {...props}
              >
                {String(children).replace(/\n$/, '')}
              </SyntaxHighlighter>
            ) : (
              <code className={className} {...props}>
                {children}
              </code>
            )
          },
          img({ node, src, alt, title, ...props }) {
            return (
              <img
                src={src}
                alt={alt}
                title={title}
                loading="lazy"
                className="rounded-lg"
                onError={(e) => {
                  e.currentTarget.src = '/image-placeholder.svg'
                  e.currentTarget.style.opacity = '0.5'
                }}
                {...props}
              />
            )
          },
          a({ node, href, children, ...props }) {
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

#### 3. 修改文章展示组件

**文件：** `web/src/components/items/ArticlePanel.tsx`

主要改动：
- 移除 `dangerouslySetInnerHTML` 和 `createSafeHTML`
- 移除 `processArticleContent` 后处理逻辑
- 使用 `<MarkdownRenderer content={item.content} />` 替代原有的 article 标签

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

同样修改 `web/src/pages/items/ItemViewPage.tsx`

### 数据迁移

提供一次性迁移脚本，将现有数据库中的 HTML 内容转换为 Markdown：

**文件：** `cmd/migrate-to-markdown.go`

```go
package main

import (
    "log"
    "oreader/internal/config"
    "oreader/internal/infra/markdown"
    "gorm.io/gorm"
)

func main() {
    db := config.GetDB()
    converter := markdown.NewConverter()

    // 批量更新 items 表的 content 字段
    var items []struct {
        ID      string
        Content string
    }

    if err := db.Table("items").Select("id, content").Find(&items).Error; err != nil {
        log.Fatal(err)
    }

    for _, item := range items {
        if item.Content == "" {
            continue
        }

        mdContent, err := converter.Convert(item.Content)
        if err != nil {
            log.Printf("转换失败 [id=%s]: %v", item.ID, err)
            continue
        }

        if err := db.Table("items").Where("id = ?", item.ID).Update("content", mdContent); err != nil {
            log.Printf("更新失败 [id=%s]: %v", item.ID, err)
        }
    }

    log.Println("迁移完成")
}
```

## 依赖清单

### 后端
- `github.com/JohannesKaufmann/html-to-markdown` - HTML 转 Markdown

### 前端
- `react-markdown` - Markdown 渲染
- `remark-gfm` - GitHub Flavored Markdown 支持（表格、删除线等）
- `react-syntax-highlighter` - 代码语法高亮

## 实施顺序

1. 后端添加 Markdown 转换模块
2. 修改 RSS 处理流程
3. 前端添加 Markdown 渲染组件
4. 修改文章展示组件
5. 编写数据迁移脚本
6. 测试和验证

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| HTML 转 Markdown 可能丢失格式 | 使用成熟的转换库，对复杂 HTML 保留 fallback |
| 旧数据迁移失败 | 提供 rollback 方案，迁移前备份 |
| 代码高亮样式变化 | 使用 oneDark 主题，与原有 Prism 主题接近 |

## 测试计划

1. **单元测试：** HTML → Markdown 转换函数
2. **集成测试：** 完整 RSS 抓取 → Markdown 渲染流程
3. **视觉测试：** 对比迁移前后的文章展示效果
