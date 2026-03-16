import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import LoadingSpinner from '../LoadingSpinner'

describe('LoadingSpinner', () => {
  it('should render when visible', () => {
    render(<LoadingSpinner visible />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('should not render when not visible', () => {
    const { container } = render(<LoadingSpinner visible={false} />)
    expect(container.firstChild).toBeNull()
  })

  it('should render default visible (true)', () => {
    render(<LoadingSpinner />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('should have accessible label', () => {
    render(<LoadingSpinner />)
    const statusElement = screen.getByRole('status')
    expect(statusElement).toHaveAttribute('aria-label', 'loading')
  })

  it('should apply correct classes for animation', () => {
    const { container } = render(<LoadingSpinner />)
    const spinner = container.querySelector('.animate-spin')
    expect(spinner).toBeInTheDocument()
  })

  it('should apply custom className when provided', () => {
    render(<LoadingSpinner className="custom-spinner-class" />)
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('custom-spinner-class')
  })

  it('should apply custom size', () => {
    render(<LoadingSpinner size="lg" />)
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('h-8')
    expect(spinner).toHaveClass('w-8')
  })

  it('should apply small size', () => {
    render(<LoadingSpinner size="sm" />)
    const spinner = screen.getByRole('status')
    expect(spinner).toHaveClass('h-4')
    expect(spinner).toHaveClass('w-4')
  })

  it('should have accessible text for screen readers', () => {
    render(<LoadingSpinner />)
    const srText = screen.getByText('loading')
    expect(srText).toHaveClass('sr-only')
  })
})
