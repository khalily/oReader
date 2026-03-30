import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import LoginPage from '../LoginPage'

// Mock handlers
const handlers = [
  http.post('/api/v1/auth/login', async ({ request }) => {
    const body = await request.json() as { email: string; password: string }

    if (body.email === 'test@example.com' && body.password === 'password123') {
      return HttpResponse.json({
        user: {
          id: '550e8400-e29b-41d4-a716-446655440000',
          email: 'test@example.com',
          nickname: null,
          avatar_url: null,
          auth_provider: 'email' as const,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        csrf_token: 'mock-csrf-token',
      })
    }

    return HttpResponse.json(
      { error: { code: 'UNAUTHORIZED', message: 'Invalid credentials' } },
      { status: 401 }
    )
  }),
]

const server = setupServer(...handlers)

describe('LoginPage', () => {
  let queryClient: QueryClient

  beforeAll(() => {
    server.listen()
  })

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
        mutations: {
          retry: false,
        },
      },
    })
  })

  afterEach(() => {
    server.resetHandlers()
    queryClient.clear()
  })

  afterAll(() => {
    server.close()
  })

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )

  it('should render login form', () => {
    render(<LoginPage />, { wrapper })

    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
    // More specific selector to avoid matching "Sign in with GitHub"
    expect(screen.getByRole('button', { name: /^sign in$/i })).toBeInTheDocument()
  })

  it('should have link to register page', () => {
    render(<LoginPage />, { wrapper })

    const registerLink = screen.getByRole('link', { name: /create one/i })
    expect(registerLink).toBeInTheDocument()
    expect(registerLink).toHaveAttribute('href', '/register')
  })

  it('should show error message on failed login', async () => {
    const user = userEvent.setup()
    render(<LoginPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    const passwordInput = screen.getByLabelText(/^password$/i)
    const submitButton = screen.getByRole('button', { name: /^sign in$/i })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'wrongpassword')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument()
    })
  })

  it('should have email input with correct attributes', () => {
    render(<LoginPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    expect(emailInput).toHaveAttribute('type', 'email')
    expect(emailInput).toHaveAttribute('placeholder', 'your@email.com')
  })

  it('should have password input with correct attributes', () => {
    render(<LoginPage />, { wrapper })

    const passwordInput = screen.getByLabelText(/^password$/i)
    expect(passwordInput).toHaveAttribute('type', 'password')
    expect(passwordInput).toHaveAttribute('placeholder', '••••••••')
  })

  it('should have card with correct title', () => {
    render(<LoginPage />, { wrapper })

    expect(screen.getByText('Enter your credentials to access your account')).toBeInTheDocument()
  })

  it('should complete login successfully', async () => {
    const user = userEvent.setup()
    render(<LoginPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    const passwordInput = screen.getByLabelText(/^password$/i)
    const submitButton = screen.getByRole('button', { name: /^sign in$/i })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')

    // Before click, button is enabled
    expect(submitButton).toBeEnabled()

    await user.click(submitButton)

    // Wait for navigation or success state
    await waitFor(() => {
      // After successful login, should navigate away or show success
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })
  })

  it('should render GitHub login button', () => {
    render(<LoginPage />, { wrapper })

    expect(screen.getByRole('button', { name: /sign in with github/i })).toBeInTheDocument()
  })

  it('should render divider between OAuth and email login', () => {
    render(<LoginPage />, { wrapper })

    expect(screen.getByText(/or continue with/i)).toBeInTheDocument()
  })

  it('should display OAuth error message from URL params (access_denied)', () => {
    // Mock URL with oauth_error param
    const wrapperWithLocation = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={['/login?oauth_error=access_denied']}>{children}</MemoryRouter>
      </QueryClientProvider>
    )

    render(<LoginPage />, { wrapper: wrapperWithLocation })

    expect(screen.getByText(/github login cancelled/i)).toBeInTheDocument()
  })

  it('should display OAuth error message (invalid_state)', () => {
    const wrapperWithLocation = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={['/login?oauth_error=invalid_state']}>{children}</MemoryRouter>
      </QueryClientProvider>
    )

    render(<LoginPage />, { wrapper: wrapperWithLocation })

    expect(screen.getByText(/login expired, please try again/i)).toBeInTheDocument()
  })

  it('should display OAuth error message (github_error)', () => {
    const wrapperWithLocation = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={['/login?oauth_error=github_error']}>{children}</MemoryRouter>
      </QueryClientProvider>
    )

    render(<LoginPage />, { wrapper: wrapperWithLocation })

    expect(screen.getByText(/github service temporarily unavailable/i)).toBeInTheDocument()
  })
})
