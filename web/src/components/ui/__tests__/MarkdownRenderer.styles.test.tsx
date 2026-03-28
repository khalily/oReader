import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MarkdownRenderer } from '../MarkdownRenderer'

// Mock @shikijs/rehype — 与 MarkdownRenderer.test.tsx 保持一致
vi.mock('@shikijs/rehype', () => ({
  default: () => async (tree: any) => tree,
}))

describe('MarkdownRenderer CSS Styles', () => {
  describe('Inline Code', () => {
    it('should render inline code without visible backticks', async () => {
      const content = 'Use the `log(a)` function to compute logarithm'
      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        const codeElement = document.querySelector('code')
        expect(codeElement).toBeInTheDocument()
        expect(codeElement?.textContent).toBe('log(a)')
      }, { timeout: 10000 })
    })

    it('should render code with brackets correctly', async () => {
      const content = 'The token list `[BOS, e, m, m, a, BOS]` is processed'
      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        const codeElements = document.querySelectorAll('code')
        expect(codeElements.length).toBeGreaterThanOrEqual(1)
        const target = Array.from(codeElements).find(el => el.textContent === '[BOS, e, m, m, a, BOS]')
        expect(target).toBeInTheDocument()
      }, { timeout: 10000 })
    })

    it('should not have node attribute in rendered code element', async () => {
      const content = 'Use `code` here'
      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        const codeElement = document.querySelector('code')
        expect(codeElement).toBeInTheDocument()
        expect(codeElement?.hasAttribute('node')).toBe(false)
      }, { timeout: 10000 })
    })
  })

  describe('Code in Tables', () => {
    it('should render code in table cells correctly', async () => {
      const content = `
| Operation | Forward |
| --- | --- |
| \`log(a)\` | $\\ln(a)$ |
      `.trim()

      render(<MarkdownRenderer content={content} />)

      expect(await screen.findByRole('table')).toBeInTheDocument()

      await waitFor(() => {
        const codeInTable = document.querySelector('table code')
        expect(codeInTable).toBeInTheDocument()
      })
    })
  })

  describe('LaTeX Math', () => {
    it('should render inline math with $ delimiters', async () => {
      const content = 'The formula $E = mc^2$ is famous'
      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        expect(document.querySelector('.katex')).toBeInTheDocument()
      })
    })

    it('should render block math with $$ delimiters', async () => {
      const content = `
Block formula:
$$
\\frac{\\partial L}{\\partial w} = \\nabla
$$
      `.trim()

      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        expect(document.querySelector('.katex-display')).toBeInTheDocument()
      })
    })

    it('should render LaTeX commands like \\ln and \\max', async () => {
      const content = 'Use $\\ln(a)$ and $\\max(0, a)$ functions'
      render(<MarkdownRenderer content={content} />)

      await waitFor(() => {
        expect(document.querySelectorAll('.katex').length).toBeGreaterThanOrEqual(2)
      })
    })
  })

  describe('Combined Code and Math', () => {
    it('should correctly render table with code and math together', async () => {
      const content = `
| Operation | Forward | Local gradients |
| --- | --- | --- |
| \`a + b\` | $a + b$ | $\\frac{\\partial}{\\partial a} = 1$ |
| \`log(a)\` | $\\ln(a)$ | $\\frac{\\partial}{\\partial a} = \\frac{1}{a}$ |
      `.trim()

      render(<MarkdownRenderer content={content} />)

      expect(await screen.findByRole('table')).toBeInTheDocument()

      await waitFor(() => {
        expect(document.querySelectorAll('table code').length).toBe(2)
        expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
      })
    })
  })
})
