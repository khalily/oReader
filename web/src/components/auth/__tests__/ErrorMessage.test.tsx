import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ErrorMessage from '../ErrorMessage'

describe('ErrorMessage', () => {
  it('should not render when message is null', () => {
    const { container } = render(<ErrorMessage message={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('should not render when message is undefined', () => {
    const { container } = render(<ErrorMessage message={undefined} />)
    expect(container.firstChild).toBeNull()
  })

  it('should not render when message is empty string', () => {
    const { container } = render(<ErrorMessage message="" />)
    expect(container.firstChild).toBeNull()
  })

  it('should render error message when provided', () => {
    render(<ErrorMessage message="Something went wrong" />)
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('should display correct styling for error', () => {
    render(<ErrorMessage message="Error occurred" />)
    const errorElement = screen.getByText('Error occurred')
    expect(errorElement).toHaveClass('text-destructive')
  })

  it('should handle API error object', () => {
    const apiError = {
      response: {
        data: {
          error: {
            code: 'VALIDATION_ERROR',
            message: 'Invalid email format',
          },
        },
      },
    } as any

    render(<ErrorMessage message={apiError} />)
    expect(screen.getByText('Invalid email format')).toBeInTheDocument()
  })

  it('should handle API error with details', () => {
    const apiError = {
      response: {
        data: {
          error: {
            code: 'VALIDATION_ERROR',
            message: 'Validation failed',
            details: {
              field: 'email',
              value: 'invalid-email',
            },
          },
        },
      },
    } as any

    render(<ErrorMessage message={apiError} />)
    expect(screen.getByText('Validation failed')).toBeInTheDocument()
  })

  it('should handle error as string', () => {
    render(<ErrorMessage message="Invalid credentials" />)
    expect(screen.getByText('Invalid credentials')).toBeInTheDocument()
  })

  it('should handle unknown error format gracefully', () => {
    render(<ErrorMessage message={{ unknown: 'format' } as any} />)
    // Should show a generic error or nothing
    const container = screen.queryByText(/error/i)
    expect(container).toBeTruthy()
  })

  it('should be dismissible with close button', async () => {
    const onDismiss = vi.fn()
    render(<ErrorMessage message="Error" onDismiss={onDismiss} />)

    const closeButton = screen.getByRole('button', { name: /close/i })
    await userEvent.click(closeButton)

    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('should not show close button when onDismiss is not provided', () => {
    render(<ErrorMessage message="Error" />)
    const closeButton = screen.queryByRole('button', { name: /close/i })
    expect(closeButton).not.toBeInTheDocument()
  })

  it('should handle different error codes', () => {
    const errorCodes = [
      'UNAUTHORIZED',
      'TOKEN_EXPIRED',
      'VALIDATION_ERROR',
      'NOT_FOUND',
      'CONFLICT',
      'RATE_LIMIT_EXCEEDED',
      'INTERNAL_ERROR',
    ]

    errorCodes.forEach((code) => {
      const apiError = {
        response: {
          data: {
            error: {
              code,
              message: `${code} message`,
            },
          },
        },
      } as any

      const { unmount } = render(<ErrorMessage message={apiError} />)
      expect(screen.getByText(`${code} message`)).toBeInTheDocument()
      unmount()
    })
  })

  it('should apply custom className when provided', () => {
    render(<ErrorMessage message="Error" className="custom-class" />)
    const errorElement = screen.getByText('Error')
    expect(errorElement).toHaveClass('custom-class')
  })
})
