import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ErrorBoundary, AppErrorBoundary, ErrorWrapper } from '../ErrorBoundary'

// Suppress console.error during tests
const originalError = console.error
beforeEach(() => {
  console.error = vi.fn()
})
afterEach(() => {
  console.error = originalError
})

// Component that throws an error when triggered
const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Test error')
  }
  return <div>No error</div>
}

describe('ErrorBoundary', () => {
  it('should render children when no error', () => {
    render(
      <ErrorBoundary>
        <div>Test content</div>
      </ErrorBoundary>
    )

    expect(screen.getByText('Test content')).toBeInTheDocument()
  })

  it('should catch errors and display fallback UI', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.queryByText('No error')).not.toBeInTheDocument()
  })

  it('should call onError callback when error occurs', () => {
    const onError = vi.fn()

    render(
      <ErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(onError).toHaveBeenCalled()
    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({ componentStack: expect.any(String) })
    )
  })

  it('should display custom fallback when provided', () => {
    render(
      <ErrorBoundary fallback={<div>Custom fallback</div>}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Custom fallback')).toBeInTheDocument()
  })

  it('should display feature name in error message', () => {
    render(
      <ErrorBoundary featureName="Test Feature">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Test Feature Error')).toBeInTheDocument()
  })

  it('should reset error state when Try Again is clicked', () => {
    // Use key to force remount after reset
    const { rerender } = render(
      <ErrorBoundary key="boundary-1">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    // Click Try Again button - this resets the internal state
    fireEvent.click(screen.getByText('Try Again'))

    // Rerender with a new key (fresh ErrorBoundary) and non-throwing children
    rerender(
      <ErrorBoundary key="boundary-2">
        <ThrowError shouldThrow={false} />
      </ErrorBoundary>
    )

    expect(screen.getByText('No error')).toBeInTheDocument()
  })

  it('should show error details in development mode', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Error details')).toBeInTheDocument()
  })
})

describe('AppErrorBoundary', () => {
  it('should render children when no error', () => {
    render(
      <AppErrorBoundary>
        <div>App content</div>
      </AppErrorBoundary>
    )

    expect(screen.getByText('App content')).toBeInTheDocument()
  })

  it('should display full-screen error UI on error', () => {
    render(
      <AppErrorBoundary>
        <ThrowError shouldThrow={true} />
      </AppErrorBoundary>
    )

    expect(screen.getByText('Application Error')).toBeInTheDocument()
    expect(screen.getByText('Refresh Page')).toBeInTheDocument()
    expect(screen.getByText('Dismiss')).toBeInTheDocument()
  })

  it('should call onError callback on error', () => {
    const onError = vi.fn()

    render(
      <AppErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </AppErrorBoundary>
    )

    expect(onError).toHaveBeenCalled()
  })

  it('should dismiss error when Dismiss is clicked', () => {
    // Use key to force remount after dismiss
    const { rerender } = render(
      <AppErrorBoundary key="app-boundary-1">
        <ThrowError shouldThrow={true} />
      </AppErrorBoundary>
    )

    expect(screen.getByText('Application Error')).toBeInTheDocument()

    // Click Dismiss button - this resets the internal state
    fireEvent.click(screen.getByText('Dismiss'))

    // Rerender with a new key (fresh ErrorBoundary) and non-throwing children
    rerender(
      <AppErrorBoundary key="app-boundary-2">
        <ThrowError shouldThrow={false} />
      </AppErrorBoundary>
    )

    expect(screen.getByText('No error')).toBeInTheDocument()
  })
})

describe('ErrorWrapper', () => {
  it('should render children when no error', () => {
    render(
      <ErrorWrapper>
        <div>Wrapper content</div>
      </ErrorWrapper>
    )

    expect(screen.getByText('Wrapper content')).toBeInTheDocument()
  })

  it('should catch errors and call onError', () => {
    const onError = vi.fn()

    render(
      <ErrorWrapper onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorWrapper>
    )

    expect(onError).toHaveBeenCalledWith(expect.any(Error))
  })

  it('should display custom fallback on error', () => {
    render(
      <ErrorWrapper fallback={<div>Wrapper fallback</div>}>
        <ThrowError shouldThrow={true} />
      </ErrorWrapper>
    )

    expect(screen.getByText('Wrapper fallback')).toBeInTheDocument()
  })
})
