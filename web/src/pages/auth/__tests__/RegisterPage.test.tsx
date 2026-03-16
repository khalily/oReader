import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import RegisterPage from '../RegisterPage'

// Mock handlers
const handlers = [
  http.post('/api/v1/auth/register', async ({ request }) => {
    const body = await request.json() as { email: string; password: string }

    if (body.email === 'existing@example.com') {
      return HttpResponse.json(
        { error: { code: 'CONFLICT', message: 'Email already exists' } },
        { status: 409 }
      )
    }

    return HttpResponse.json({
      user: {
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: body.email,
        nickname: null,
        avatar_url: null,
        auth_provider: 'email' as const,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
      csrf_token: 'mock-csrf-token',
    })
  }),
]

const server = setupServer(...handlers)

describe('RegisterPage', () => {
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

  it('should render register form', () => {
    render(<RegisterPage />, { wrapper })

    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  it('should have link to login page', () => {
    render(<RegisterPage />, { wrapper })

    const loginLink = screen.getByRole('link', { name: /sign in/i })
    expect(loginLink).toBeInTheDocument()
    expect(loginLink).toHaveAttribute('href', '/login')
  })

  it('should have nickname input', () => {
    render(<RegisterPage />, { wrapper })

    expect(screen.getByLabelText(/nickname/i)).toBeInTheDocument()
  })

  it('should have email input with correct attributes', () => {
    render(<RegisterPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    expect(emailInput).toHaveAttribute('type', 'email')
    expect(emailInput).toHaveAttribute('placeholder', 'your@email.com')
  })

  it('should have password input with correct attributes', () => {
    render(<RegisterPage />, { wrapper })

    const passwordInput = screen.getByLabelText(/^password$/i)
    expect(passwordInput).toHaveAttribute('type', 'password')
    expect(passwordInput).toHaveAttribute('placeholder', '••••••••')
  })

  it('should have confirm password input with correct attributes', () => {
    render(<RegisterPage />, { wrapper })

    const confirmPasswordInput = screen.getByLabelText(/confirm password/i)
    expect(confirmPasswordInput).toHaveAttribute('type', 'password')
    expect(confirmPasswordInput).toHaveAttribute('placeholder', '••••••••')
  })

  it('should show error message on failed registration', async () => {
    const user = userEvent.setup()
    render(<RegisterPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    const passwordInput = screen.getByLabelText(/^password$/i)
    const confirmPasswordInput = screen.getByLabelText(/confirm password/i)
    const submitButton = screen.getByRole('button', { name: /create account/i })

    await user.type(emailInput, 'existing@example.com')
    await user.type(passwordInput, 'password123')
    await user.type(confirmPasswordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText(/email already exists/i)).toBeInTheDocument()
    })
  })

  it('should have card with correct title', () => {
    render(<RegisterPage />, { wrapper })

    expect(screen.getByText('Sign up to get started with oReader')).toBeInTheDocument()
  })

  it('should show loading state during registration', async () => {
    const user = userEvent.setup()
    render(<RegisterPage />, { wrapper })

    const emailInput = screen.getByLabelText(/email/i)
    const passwordInput = screen.getByLabelText(/^password$/i)
    const confirmPasswordInput = screen.getByLabelText(/confirm password/i)
    const submitButton = screen.getByRole('button', { name: /create account/i })

    await user.type(emailInput, 'new@example.com')
    await user.type(passwordInput, 'password123')
    await user.type(confirmPasswordInput, 'password123')

    // Before click, button is enabled
    expect(submitButton).toBeEnabled()

    await user.click(submitButton)

    // Check that loading text appears
    await waitFor(() => {
      expect(screen.getByText(/Creating account/i)).toBeInTheDocument()
    })
  })
})
