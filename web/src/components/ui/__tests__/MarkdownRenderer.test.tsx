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
    render(<MarkdownRenderer content={'$$\\frac{\\partial L}{\\partial w} = \\nabla$$'} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders complex partial derivative formulas', () => {
    const formula = '$$\\frac{\\partial(a \\cdot b)}{\\partial a} = b$$'
    render(<MarkdownRenderer content={formula} />)
    const katexElement = document.querySelector('.katex')
    expect(katexElement).toBeInTheDocument()
  })

  it('renders LaTeX formulas inside GFM tables', () => {
    const tableWithMath = `| Operation | Forward | Local gradients |
|-----------|---------|----------------|
| \`a + b\` | $$a + b$$ | $$\\frac{\\partial}{\\partial a} = 1$$ |`
    render(<MarkdownRenderer content={tableWithMath} />)
    // Should have both table and katex rendered
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
  })
})
