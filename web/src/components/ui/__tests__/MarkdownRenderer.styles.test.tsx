import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MarkdownRenderer } from '../MarkdownRenderer'

describe('MarkdownRenderer CSS Styles', () => {
  describe('Inline Code', () => {
    it('should render inline code without visible backticks', () => {
      const content = 'Use the `log(a)` function to compute logarithm'
      render(<MarkdownRenderer content={content} />)

      const codeElement = screen.getByText('log(a)')
      expect(codeElement.tagName).toBe('CODE')

      // Verify the code element exists and has correct content
      expect(codeElement.textContent).toBe('log(a)')
    })

    it('should render code with brackets correctly', () => {
      const content = 'The token list `[BOS, e, m, m, a, BOS]` is processed'
      render(<MarkdownRenderer content={content} />)

      const codeElement = screen.getByText('[BOS, e, m, m, a, BOS]')
      expect(codeElement.tagName).toBe('CODE')
      expect(codeElement.textContent).toBe('[BOS, e, m, m, a, BOS]')
    })

    it('should not have node attribute in rendered code element', () => {
      const content = 'Use `code` here'
      render(<MarkdownRenderer content={content} />)

      const codeElement = screen.getByText('code')
      // The node attribute should not be present in the DOM
      expect(codeElement.hasAttribute('node')).toBe(false)
    })
  })

  describe('Code in Tables', () => {
    it('should render code in table cells correctly', () => {
      const content = `
| Operation | Forward |
| --- | --- |
| \`log(a)\` | $\\ln(a)$ |
      `.trim()

      render(<MarkdownRenderer content={content} />)

      // Check that code is rendered in table
      const codeElement = screen.getByText('log(a)')
      expect(codeElement.tagName).toBe('CODE')

      // Check table structure exists
      const table = document.querySelector('table')
      expect(table).toBeInTheDocument()
    })
  })

  describe('LaTeX Math', () => {
    it('should render inline math with $ delimiters', () => {
      const content = 'The formula $E = mc^2$ is famous'
      render(<MarkdownRenderer content={content} />)

      // KaTeX renders math in span elements with class katex
      const katexElement = document.querySelector('.katex')
      expect(katexElement).toBeInTheDocument()
    })

    it('should render block math with $$ delimiters', () => {
      const content = `
Block formula:
$$
\\frac{\\partial L}{\\partial w} = \\nabla
$$
      `.trim()

      render(<MarkdownRenderer content={content} />)

      const katexElement = document.querySelector('.katex-display')
      expect(katexElement).toBeInTheDocument()
    })

    it('should render LaTeX commands like \\ln and \\max', () => {
      const content = 'Use $\\ln(a)$ and $\\max(0, a)$ functions'
      render(<MarkdownRenderer content={content} />)

      // Both math expressions should be rendered
      const katexElements = document.querySelectorAll('.katex')
      expect(katexElements.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Combined Code and Math', () => {
    it('should correctly render table with code and math together', () => {
      const content = `
| Operation | Forward | Local gradients |
| --- | --- | --- |
| \`a + b\` | $a + b$ | $\\frac{\\partial}{\\partial a} = 1$ |
| \`log(a)\` | $\\ln(a)$ | $\\frac{\\partial}{\\partial a} = \\frac{1}{a}$ |
      `.trim()

      render(<MarkdownRenderer content={content} />)

      // Verify code elements using more specific selectors
      const codeElements = document.querySelectorAll('table code')
      expect(codeElements.length).toBe(2)

      // Verify table exists
      expect(document.querySelector('table')).toBeInTheDocument()

      // Verify math is rendered
      expect(document.querySelectorAll('.katex').length).toBeGreaterThan(0)
    })
  })
})
