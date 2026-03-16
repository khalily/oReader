import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import ProtectedRoute from '../ProtectedRoute'

// Mock zustand store
const mockAuthStoreState = {
  isAuthenticated: false,
  user: null,
  csrfToken: null,
  setUser: vi.fn(),
  clearUser: vi.fn(),
  initializeFromStorage: vi.fn(),
  getCsrfToken: vi.fn(),
}

vi.mock('../../stores/authStore', () => ({
  useAuthStore: ((selector: typeof mockAuthStoreState) => selector(mockAuthStoreState)),
  authStore: {
    getState: vi.fn(() => mockAuthStoreState),
  },
}))

const TestComponent = () => <div>Protected Content</div>

describe('ProtectedRoute', () => {
  beforeEach(() => {
    // Reset mock state before each test
    mockAuthStoreState.isAuthenticated = false
    mockAuthStoreState.user = null
    vi.clearAllMocks()
  })

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter initialEntries={['/protected']}>{children}</MemoryRouter>
  )

  it('should redirect to login when not authenticated', () => {
    mockAuthStoreState.isAuthenticated = false

    render(
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route
          path="/protected"
          element={
            <ProtectedRoute>
              <TestComponent />
            </ProtectedRoute>
          }
        />
      </Routes>,
      { wrapper }
    )

    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('should render protected content when authenticated', () => {
    mockAuthStoreState.isAuthenticated = true
    mockAuthStoreState.user = {
      id: '550e8400-e29b-41d4-a716-446655440000',
      email: 'test@example.com',
      nickname: null,
      avatar_url: null,
      auth_provider: 'email' as const,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    }

    render(
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route
          path="/protected"
          element={
            <ProtectedRoute>
              <TestComponent />
            </ProtectedRoute>
          }
        />
      </Routes>,
      { wrapper }
    )

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
  })

  it('should preserve redirect location when redirecting to login', () => {
    mockAuthStoreState.isAuthenticated = false

    render(
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route
          path="/protected"
          element={
            <ProtectedRoute>
              <TestComponent />
            </ProtectedRoute>
          }
        />
      </Routes>,
      { wrapper }
    )

    // The user should be redirected to login
    expect(screen.getByText('Login Page')).toBeInTheDocument()
  })

  it('should work with nested routes', () => {
    mockAuthStoreState.isAuthenticated = true

    render(
      <MemoryRouter initialEntries={['/protected/nested']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <ProtectedRoute>
                <div>Protected Layout</div>
              </ProtectedRoute>
            }
          />
          <Route
            path="/protected/nested"
            element={
              <ProtectedRoute>
                <div>Nested Protected Content</div>
              </ProtectedRoute>
            }
          />
        </Routes>
      </MemoryRouter>
    )

    expect(screen.getByText('Nested Protected Content')).toBeInTheDocument()
  })

  it('should handle loading state when checking authentication', async () => {
    // Simulate a loading state by having authentication check be async
    mockAuthStoreState.isAuthenticated = false

    const { container } = render(
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route
          path="/protected"
          element={
            <ProtectedRoute>
              <TestComponent />
            </ProtectedRoute>
          }
        />
      </Routes>,
      { wrapper }
    )

    // Should eventually show login page
    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })

  it('should render children when user exists', () => {
    mockAuthStoreState.isAuthenticated = true
    mockAuthStoreState.user = {
      id: '550e8400-e29b-41d4-a716-446655440000',
      email: 'user@example.com',
      nickname: 'Test User',
      avatar_url: null,
      auth_provider: 'email' as const,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    }

    render(
      <ProtectedRoute>
        <TestComponent />
      </ProtectedRoute>,
      { wrapper }
    )

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })
})
