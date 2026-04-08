import { MarkdownHooks, defaultUrlTransform } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import rehypeShiki from '@shikijs/rehype'
import { visit, CONTINUE } from 'unist-util-visit'
import { cn } from '@/lib/utils'
import React from 'react'
import { CopyButton } from './CopyButton'

interface MarkdownRendererProps {
  content: string
  className?: string
}

// ─── Shiki 配置 ────────────────────────────────────────────────────
const shikiOptions = {
  themes: {
    light: 'github-light',
    dark: 'github-dark',
  },
  defaultColor: false, // 使用 CSS 变量实现双主题切换
}

// ─── remark-footnote-refs ────────────────────────────────────────────
// Converts overview ↔ detail cross-references in RSS Markdown content.
//
// Markdown source (overview):
//   - Anthropic 发布 Claude Mythos 模型 [↗](https://example.com) `#1`
//
// Markdown source (detail):
//   ## [Anthropic 发布 Claude Mythos 模型](https://example.com) `\#1`
//
// After parsing, `\#1` inside inline code becomes literal `#1`
// (the backslash escape is consumed by the Markdown parser).
//
// Pass 1: headings with inlineCode `#N` → add id="ref-N", remove marker
// Pass 2: non-heading inlineCode `#N` → replace with <a href="#ref-N">#N</a>
// ────────────────────────────────────────────────────────────────────
const REF_PATTERN = /^\\?#(\d+)$/

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function remarkFootnoteRefs() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // Pass 1: headings → add id, strip the marker
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'heading', (node: any) => {
      if (!Array.isArray(node.children)) return CONTINUE

      let refId: string | null = null

      for (let i = node.children.length - 1; i >= 0; i--) {
        const child = node.children[i]
        if (child.type === 'inlineCode') {
          const match = REF_PATTERN.exec(child.value)
          if (match) {
            refId = `ref-${match[1]}`
            node.children.splice(i, 1)
            break
          }
        }
      }

      if (refId) {
        node.data ??= {}
        node.data.hProperties = { id: refId }
      }

      return CONTINUE
    })

    // Pass 2: non-heading inlineCode → replace with link
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    visit(tree, 'inlineCode', (node: any, index: number | undefined, parent: any) => {
      if (index === undefined || !parent || !Array.isArray(parent.children)) {
        return CONTINUE
      }
      if (parent.type === 'heading') return CONTINUE

      const match = REF_PATTERN.exec(node.value)
      if (!match) return CONTINUE

      parent.children[index] = {
        type: 'link',
        url: `#ref-${match[1]}`,
        title: null,
        children: [{ type: 'text', value: `#${match[1]}` }],
      }

      return CONTINUE
    })
  }
}

// ─── rehype-add-default-lang ────────────────────────────────────────
// 后端 HTML→Markdown 转换后，代码块可能不带语言标识（如 ```python），
// 导致 remark-rehype 生成的 <code> 没有 className="language-xxx"。
// Shiki 只处理带 language-xxx class 的 code 元素，无标识的直接跳过。
// 此插件在 Shiki 之前运行，为无语言标识的代码块添加 language-text。
// ────────────────────────────────────────────────────────────────────
function rehypeAddDefaultLang() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (tree: any) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function visit(node: any) {
      if (
        node?.tagName === 'pre' &&
        node.children?.[0]?.tagName === 'code'
      ) {
        const code = node.children[0]
        const classes: string[] = code.properties?.className ?? []
        const hasLang = classes.some(
          (c: string) => typeof c === 'string' && c.startsWith('language-')
        )
        if (!hasLang) {
          code.properties ??= {}
          code.properties.className = [...classes, 'language-text']
        }
      }
      for (const child of node?.children ?? []) visit(child)
    }
    visit(tree)
  }
}

// ─── Module-level 常量 ──────────────────────────────────────────────
// MarkdownHooks 在 useEffect 中比较 rehypePlugins 引用，
// 每次 render 创建新数组会导致不必要的重新处理。
// ────────────────────────────────────────────────────────────────────
const remarkPlugins = [remarkGfm, remarkMath, remarkFootnoteRefs]

/**
 * URL transform: allow data: URIs for inline images (base64 embedded by MinerU).
 * react-markdown v10's defaultUrlTransform blocks data: URIs for security,
 * returning empty strings that strip src from <img> elements.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const urlTransform = (url: string, key: string): string => {
  if (url.startsWith('data:')) return url
  return defaultUrlTransform(url)
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const rehypePlugins: any[] = [
  rehypeKatex,
  rehypeAddDefaultLang,
  [rehypeShiki, shikiOptions],
]

/**
 * MarkdownRenderer renders Markdown content with:
 * - GitHub Flavored Markdown support (tables, strikethrough, etc.)
 * - LaTeX math rendering with KaTeX (inline `$...$` and block `$$...$$`)
 * - Syntax highlighting with Shiki (dual theme: light + dark)
 * - Copy button for code blocks
 * - Lazy loading for images
 * - Secure external links (target="_blank", rel="noopener noreferrer")
 */
export function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  const extractCodeFromChildren = (children: React.ReactNode): string => {
    if (typeof children === 'string') return children
    if (Array.isArray(children)) {
      return children.map(extractCodeFromChildren).join('')
    }
    if (React.isValidElement(children)) {
      const childProps = children.props as { children?: React.ReactNode }
      if (childProps && typeof childProps.children !== 'undefined') {
        return extractCodeFromChildren(childProps.children)
      }
    }
    return ''
  }

  return (
    <div className={cn('prose prose-slate dark:prose-invert max-w-none', className)}>
      <MarkdownHooks
        remarkPlugins={remarkPlugins}
        rehypePlugins={rehypePlugins}
        fallback={<div>{content}</div>}
        urlTransform={urlTransform}
        components={{
          pre({ children, node: _node, ...props }) { // eslint-disable-line @typescript-eslint/no-unused-vars
            const code = extractCodeFromChildren(children)
            return (
              <div className="relative group">
                <pre {...props}>{children}</pre>
                {code && <CopyButton code={code} />}
              </div>
            )
          },
          img({ src, alt, title, ...restProps }) {
            // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
            const { node: _node, ...props } = restProps as any
            // data: URIs must not use loading="lazy": Chromium defers decoding
            // until the element enters the viewport, but with height:auto and
            // naturalHeight=0 (not yet decoded) the computed height is 0px.
            // The image never "enters the viewport", creating a dead-lock.
            const isDataUri = src?.startsWith('data:')
            return (
              <img
                src={src}
                alt={alt || ''}
                title={title}
                loading={isDataUri ? undefined : 'lazy'}
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
            // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
            const { node: _node, ...props } = restProps as any
            const isAnchor = href?.startsWith('#')
            return (
              <a
                href={href}
                {...(isAnchor ? {} : { target: '_blank', rel: 'noopener noreferrer' })}
                className={isAnchor ? undefined : 'text-primary hover:underline'}
                {...props}
              >
                {children}
              </a>
            )
          },
          table({ children, ...restProps }) {
            // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any
            const { node: _node, ...props } = restProps as any
            return (
              <div className="overflow-x-auto -mx-1 px-1">
                <table {...props}>{children}</table>
              </div>
            )
          },
        }}
      >
        {content}
      </MarkdownHooks>
    </div>
  )
}
