import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { MarkdownRenderer } from '../MarkdownRenderer'

// Mock @shikijs/rehype — 默认导出是 async transformer，与真实行为一致
vi.mock('@shikijs/rehype', () => ({
  default: () => async (tree: any) => tree,
}))

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

  it('renders plain text', async () => {
    const { container } = render(<MarkdownRenderer content="Hello World" />)
    // MarkdownHooks renders async; wait until text appears
    await waitFor(() => {
      expect(container.textContent).toContain('Hello World')
    })
  })

  it('renders headings', async () => {
    render(<MarkdownRenderer content="# Title" />)
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Title' })
    ).toBeInTheDocument()
  })

  it('renders links with target blank', async () => {
    render(<MarkdownRenderer content="[Link](https://example.com)" />)
    const link = await screen.findByRole('link', { name: 'Link' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('renders code blocks with pre element', async () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    await waitFor(() => {
      expect(document.querySelector('pre')).toBeInTheDocument()
      expect(document.querySelector('pre code')).toBeInTheDocument()
    })
  })

  it('renders code blocks with language class', async () => {
    render(<MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />)
    await waitFor(() => {
      const codeElement = document.querySelector('code')
      expect(codeElement).toBeInTheDocument()
      expect(codeElement?.className).toMatch(/language-go/)
    })
  })

  it('renders multiple code blocks', async () => {
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
    await waitFor(() => {
      expect(container.querySelectorAll('pre code').length).toBe(3)
    })
  })

  it('renders images with lazy loading', async () => {
    render(<MarkdownRenderer content="![Alt text](https://example.com/img.png)" />)
    const img = await screen.findByRole('img', { name: 'Alt text' })
    expect(img).toHaveAttribute('loading', 'lazy')
  })

  it('applies custom className', () => {
    const { container } = render(<MarkdownRenderer content="test" className="custom-class" />)
    // className on outer div is synchronous, no need to wait
    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('renders GFM tables', async () => {
    render(
      <MarkdownRenderer
        content={`| A | B |
|---|---|
| 1 | 2 |`}
      />
    )
    expect(await screen.findByRole('table')).toBeInTheDocument()
  })

  it('renders inline LaTeX math with $...$', async () => {
    render(<MarkdownRenderer content="The formula $E = mc^2$ is famous." />)
    await waitFor(() => {
      expect(document.querySelector('.katex')).toBeInTheDocument()
    })
    expect(screen.getByText('The formula', { exact: false })).toBeInTheDocument()
    expect(screen.getByText('is famous.', { exact: false })).toBeInTheDocument()
  })

  it('renders block LaTeX math with $$...$$', async () => {
    render(<MarkdownRenderer content={'$$\\frac{\\partial L}{\\partial w} = \\nabla$$'} />)
    await waitFor(() => {
      expect(document.querySelector('.katex')).toBeInTheDocument()
    })
  })

  it('renders complex partial derivative formulas', async () => {
    const formula = '$$\\frac{\\partial(a \\cdot b)}{\\partial a} = b$$'
    render(<MarkdownRenderer content={formula} />)
    await waitFor(() => {
      expect(document.querySelector('.katex')).toBeInTheDocument()
    })
  })

  it('renders LaTeX formulas inside GFM tables', async () => {
    const tableWithMath = `| Operation | Forward | Local gradients |
|-----------|---------|----------------|
| \`a + b\` | $$a + b$$ | $$\\frac{\\partial}{\\partial a} = 1$$ |`
    render(<MarkdownRenderer content={tableWithMath} />)
    expect(await screen.findByRole('table')).toBeInTheDocument()
    await waitFor(() => {
      expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
    })
  })

  it('shows copy button on code blocks', async () => {
    const { container } = render(
      <MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />
    )
    await waitFor(() => {
      expect(container.querySelector('button')).toBeInTheDocument()
    })
  })

  it('copies code to clipboard when copy button is clicked', async () => {
    const { container } = render(
      <MarkdownRenderer content={'```go\nfmt.Println("Hello")\n```'} />
    )
    const copyButton = await waitFor(() => {
      const btn = container.querySelector('button')
      expect(btn).toBeInTheDocument()
      return btn!
    })

    fireEvent.click(copyButton)

    await waitFor(() => {
      expect(mockClipboardWrite).toHaveBeenCalled()
    })
  })
})
