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

  it('renders links with target blank', () => {
    render(<MarkdownRenderer content="[Link](https://example.com)" />)
    const link = screen.getByRole('link', { name: 'Link' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('renders code blocks with syntax highlighting', () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    // Syntax highlighter breaks text into spans, so check for parts
    expect(screen.getByText('fmt')).toBeInTheDocument()
    expect(screen.getByText('Println')).toBeInTheDocument()
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
