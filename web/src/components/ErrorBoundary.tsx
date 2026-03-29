import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface ErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  featureName?: string
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
}

/**
 * Feature-level Error Boundary for graceful degradation.
 * Wraps individual features to isolate errors.
 */
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    }
  }

  static getDerivedStateFromError(_error: Error): Partial<ErrorBoundaryState> { // eslint-disable-line @typescript-eslint/no-unused-vars
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log to console in development
    if (import.meta.env.DEV) {
      console.error('ErrorBoundary caught an error:', error, errorInfo)
    }

    // Call custom error handler if provided
    this.props.onError?.(error, errorInfo)

    // Update state with error details
    this.setState({
      error,
      errorInfo,
    })
  }

  handleReset = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    })
  }

  render(): ReactNode {
    const { hasError, error } = this.state
    const { children, fallback, featureName } = this.props

    if (!hasError) {
      return children
    }

    // Use custom fallback if provided
    if (fallback) {
      return fallback
    }

    // Default error UI for feature-level boundary
    return (
      <Card className="border-destructive/50 bg-destructive/5">
        <CardHeader>
          <CardTitle className="text-destructive flex items-center gap-2">
            <span>⚠️</span>
            {featureName ? `${featureName} Error` : 'Something went wrong'}
          </CardTitle>
          <CardDescription>
            {featureName
              ? `The ${featureName} feature encountered an error. You can try again, or navigate to a different section.`
              : 'This part of the application encountered an error.'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <details className="text-sm text-muted-foreground">
              <summary className="cursor-pointer hover:text-foreground">
                Error details
              </summary>
              <pre className="mt-2 p-3 bg-muted rounded-md overflow-x-auto">
                {error.toString()}
                {this.state.errorInfo?.componentStack}
              </pre>
            </details>
          )}
          <div className="flex gap-2">
            <Button onClick={this.handleReset} variant="default">
              Try Again
            </Button>
            <Button
              onClick={() => window.location.href = '/items'}
              variant="outline"
            >
              Go to Home
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }
}

/**
 * App-level Error Boundary for catastrophic errors.
 * Displays a full-screen error page when the entire app fails.
 */
export class AppErrorBoundary extends Component<
  Omit<ErrorBoundaryProps, 'fallback'>,
  ErrorBoundaryState
> {
  constructor(props: Omit<ErrorBoundaryProps, 'fallback'>) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    }
  }

  static getDerivedStateFromError(_error: Error): Partial<ErrorBoundaryState> { // eslint-disable-line @typescript-eslint/no-unused-vars
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    if (import.meta.env.DEV) {
      console.error('AppErrorBoundary caught an error:', error, errorInfo)
    }

    this.props.onError?.(error, errorInfo)

    this.setState({
      error,
      errorInfo,
    })
  }

  render(): ReactNode {
    const { hasError } = this.state
    const { children } = this.props

    if (!hasError) {
      return children
    }

    return (
      <div className="min-h-screen flex items-center justify-center bg-background p-4">
        <Card className="max-w-lg w-full border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive text-2xl flex items-center gap-2">
              <span>💥</span>
              Application Error
            </CardTitle>
            <CardDescription>
              The application encountered a critical error. Please refresh the
              page to try again.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {this.state.error && import.meta.env.DEV && (
              <details className="text-sm text-muted-foreground">
                <summary className="cursor-pointer hover:text-foreground">
                  Technical Details
                </summary>
                <pre className="mt-2 p-3 bg-muted rounded-md overflow-x-auto text-xs">
                  {this.state.error.toString()}
                  {this.state.errorInfo?.componentStack}
                </pre>
              </details>
            )}
            <div className="flex gap-2">
              <Button
                onClick={() => window.location.reload()}
                variant="default"
                className="flex-1"
              >
                Refresh Page
              </Button>
              <Button
                onClick={() => {
                  this.setState({ hasError: false, error: null, errorInfo: null })
                }}
                variant="outline"
                className="flex-1"
              >
                Dismiss
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }
}

/**
 * Hook-style Error Boundary wrapper for function components.
 * Use this for simpler integration with modern React patterns.
 */
interface ErrorWrapperProps {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error) => void
}

export function ErrorWrapper({
  children,
  fallback,
  onError,
}: ErrorWrapperProps): ReactNode {
  return (
    <ErrorBoundary fallback={fallback} onError={onError && ((e) => onError(e))}>
      {children}
    </ErrorBoundary>
  )
}
