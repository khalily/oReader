import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { useAuth } from '../useAuth'

// Mock API base URL
const API_BASE_URL = '/api/v1'

// Mock handlers
const handlers = [
  // Login
  http.post(`${API_BASE_URL}/auth/login`, async ({ request }) => {
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

  // Register
  http.post(`${API_BASE_URL}/auth/register`, async ({ request }) => {
    const body = await request.json() as { email: string; password: string }

    if (body.email === 'existing@example.com') {
      return HttpResponse.json(
        { error: { code: 'CONFLICT', message: 'Email already exists' } },
        { status: 409 }
      )
    }

    if (!body.email || !body.email.includes('@')) {
      return HttpResponse.json(
        { error: { code: 'VALIDATION_ERROR', message: 'Invalid email', details: { field: 'email' } } },
        { status: 400 }
      )
    }

    return HttpResponse.json({
      user: {
        id: '550e8400-e29b-41d4-a716-446655440001',
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

  // Logout
  http.post(`${API_BASE_URL}/auth/logout`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Get current user (me)
  http.get(`${API_BASE_URL}/auth/me`, () => {
    return HttpResponse.json({
      id: '550e8400-e29b-41d4-a716-446655440000',
      email: 'test@example.com',
      nickname: null,
      avatar_url: null,
      auth_provider: 'email' as const,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    })
  }),
]

const server = setupServer(...handlers)

describe('useAuth hook', () => {
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
      {children}
    </QueryClientProvider>
  )

  describe('useLogin', () => {
    it('should successfully login with valid credentials', async () => {
      const { result } = renderHook(() => useAuth().useLogin(), { wrapper })

      result.current.mutate({
        email: 'test@example.com',
        password: 'password123',
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.user).toEqual({
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: null,
        avatar_url: null,
        auth_provider: 'email',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      })
      expect(result.current.data?.csrf_token).toBe('mock-csrf-token')
    })

    it('should fail login with invalid credentials', async () => {
      const { result } = renderHook(() => useAuth().useLogin(), { wrapper })

      result.current.mutate({
        email: 'test@example.com',
        password: 'wrongpassword',
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'UNAUTHORIZED')
    })

    it('should have loading state during login', async () => {
      const { result } = renderHook(() => useAuth().useLogin(), { wrapper })

      // Initially idle
      expect(result.current.isPending).toBe(false)

      result.current.mutate({
        email: 'test@example.com',
        password: 'password123',
      })

      // Loading state
      expect(result.current.isPending).toBe(true)

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })
    })
  })

  describe('useRegister', () => {
    it('should successfully register with valid data', async () => {
      const { result } = renderHook(() => useAuth().useRegister(), { wrapper })

      result.current.mutate({
        email: 'newuser@example.com',
        password: 'password123',
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.user.email).toBe('newuser@example.com')
      expect(result.current.data?.user.auth_provider).toBe('email')
    })

    it('should fail registration with existing email', async () => {
      const { result } = renderHook(() => useAuth().useRegister(), { wrapper })

      result.current.mutate({
        email: 'existing@example.com',
        password: 'password123',
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'CONFLICT')
    })

    it('should fail registration with invalid email format', async () => {
      const { result } = renderHook(() => useAuth().useRegister(), { wrapper })

      result.current.mutate({
        email: 'invalid-email',
        password: 'password123',
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toHaveProperty('response.data.error.code', 'VALIDATION_ERROR')
    })

    it('should allow registration with nickname', async () => {
      const { result } = renderHook(() => useAuth().useRegister(), { wrapper })

      result.current.mutate({
        email: 'withnick@example.com',
        password: 'password123',
        nickname: 'TestUser',
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data?.user.email).toBe('withnick@example.com')
    })
  })

  describe('useLogout', () => {
    it('should successfully logout', async () => {
      const { result } = renderHook(() => useAuth().useLogout(), { wrapper })

      result.current.mutate()

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })
    })
  })

  describe('useMe', () => {
    it('should fetch current user', async () => {
      const { result } = renderHook(() => useAuth().useMe(), { wrapper })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual({
        id: '550e8400-e29b-41d4-a716-446655440000',
        email: 'test@example.com',
        nickname: null,
        avatar_url: null,
        auth_provider: 'email',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      })
    })

    it('should have loading state initially', () => {
      const { result } = renderHook(() => useAuth().useMe(), { wrapper })

      expect(result.current.isLoading).toBe(true)
    })
  })
})
