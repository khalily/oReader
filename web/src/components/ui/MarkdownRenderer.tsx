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
