import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { cn } from '@/lib/utils'
import React from 'react'

interface MarkdownRendererProps {
  content: string
  className?: string
}

/**
 * MarkdownRenderer renders Markdown content with:
 * - GitHub Flavored Markdown support (tables, strikethrough, etc.)
 * - LaTeX math rendering with KaTeX (inline `$...$` and block `$$...$$`)
 * - Syntax highlighting for code blocks
 * - Lazy loading for images
 * - Secure external links (target="_blank", rel="noopener noreferrer")
 */
export function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  return (
    <div className={cn('prose prose-slate dark:prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
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

            // Create a wrapper to fix the ref type issue
            const CodeBlock: React.FC<any> = ({ children }) => {
              return (
                <SyntaxHighlighter
                  style={oneDark as any}
                  language={language}
                  PreTag="div"
                >
                  {String(children).replace(/\n$/, '')}
                </SyntaxHighlighter>
              )
            }

            return <CodeBlock {...props}>{children}</CodeBlock>
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
