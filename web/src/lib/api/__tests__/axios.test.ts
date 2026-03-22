import { describe, it, expect, vi, beforeEach, afterEach, beforeAll, afterAll } from 'vitest'
import { apiClient, setCsrfToken } from '../axios'
import { server } from '../../../test/server'
import { http, HttpResponse } from 'msw'

// Use vi.hoisted to define mock functions before vi.mock runs
const mockClearUser = vi.hoisted(() => vi.fn())
const mockGetCsrfToken = vi.hoisted(() => vi.fn(() => 'mock-csrf-token'))
const mockSetState = vi.hoisted(() => vi.fn())

vi.mock('../../../stores/authStore', () => ({
  useAuthStore: {
    getState: () => ({
      getCsrfToken: mockGetCsrfToken,
      clearUser: mockClearUser,
    }),
    setState: mockSetState,
  },
}))

describe('axios client', () => {
  // Start MSW server before all tests
  beforeAll(() => {
    server.listen({ onUnhandledRequest: 'error' })
  })

  // Reset handlers after each test
  afterEach(() => {
    server.resetHandlers()
    vi.clearAllMocks()
  })

  // Clean up after all tests
  afterAll(() => {
    server.close()
  })

  beforeEach(() => {
    // Reset CSRF token before each test
    setCsrfToken(null)
    mockClearUser.mockClear()
    mockSetState.mockClear()
    mockGetCsrfToken.mockClear()
    mockGetCsrfToken.mockReturnValue('mock-csrf-token')
  })

  describe('base configuration', () => {
    it('should have base URL configured', () => {
      expect(apiClient.defaults.baseURL).toBe('/api/v1')
    })

    it('should have withCredentials enabled', () => {
      expect(apiClient.defaults.withCredentials).toBe(true)
    })
  })

  describe('CSRF token handling', () => {
    it('should set and get CSRF token', () => {
      setCsrfToken('test-csrf-token')
      expect(apiClient.defaults.headers.common['X-CSRF-Token']).toBe('test-csrf-token')
    })

    it('should clear CSRF token when set to null', () => {
      setCsrfToken('test-token')
      expect(apiClient.defaults.headers.common['X-CSRF-Token']).toBe('test-token')

      setCsrfToken(null)
      expect(apiClient.defaults.headers.common['X-CSRF-Token']).toBeUndefined()
    })
  })

  describe('request interceptor', () => {
    it('should include CSRF token in request headers when available', async () => {
      setCsrfToken('test-csrf-token')

      // Mock a request handler
      server.use(
        http.get('/api/v1/test', ({ request }) => {
          const csrfToken = request.headers.get('X-CSRF-Token')
          return HttpResponse.json({ csrfToken })
        })
      )

      const response = await apiClient.get('/test')
      // The token from getCsrfToken() in the store is used first
      expect(response.data.csrfToken).toBe('mock-csrf-token')
    })

    it('should use setCsrfToken when store token is not available', async () => {
      // Make store return null
      mockGetCsrfToken.mockReturnValue(null)
      setCsrfToken('fallback-csrf-token')

      // Mock a request handler
      server.use(
        http.get('/api/v1/test', ({ request }) => {
          const csrfToken = request.headers.get('X-CSRF-Token')
          return HttpResponse.json({ csrfToken })
        })
      )

      const response = await apiClient.get('/test')
      expect(response.data.csrfToken).toBe('fallback-csrf-token')
    })
  })

  describe('response interceptor', () => {
    it('should return data on successful response', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json({ message: 'success' })
        })
      )

      const response = await apiClient.get('/test')
      expect(response.data).toEqual({ message: 'success' })
    })

    it('should handle token expired error (401) with refresh failure', async () => {
      server.use(
        // First request returns TOKEN_EXPIRED
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'TOKEN_EXPIRED', message: 'Token expired' } },
            { status: 401 }
          )
        }),
        // Refresh endpoint also fails
        http.post('/api/v1/auth/refresh', () => {
          return HttpResponse.json(
            { error: { code: 'REFRESH_FAILED', message: 'Refresh token expired' } },
            { status: 401 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 401,
        },
      })

      // Verify that clearUser was called when refresh failed
      expect(mockClearUser).toHaveBeenCalled()
    })

    it('should retry request after successful token refresh', async () => {
      let requestCount = 0

      server.use(
        // First request returns TOKEN_EXPIRED, second succeeds
        http.get('/api/v1/test', () => {
          requestCount++
          if (requestCount === 1) {
            return HttpResponse.json(
              { error: { code: 'TOKEN_EXPIRED', message: 'Token expired' } },
              { status: 401 }
            )
          }
          return HttpResponse.json({ message: 'success after refresh' })
        }),
        // Refresh endpoint succeeds
        http.post('/api/v1/auth/refresh', () => {
          return HttpResponse.json({ csrf_token: 'new-csrf-token' })
        })
      )

      const response = await apiClient.get('/test')
      expect(response.data).toEqual({ message: 'success after refresh' })
      expect(requestCount).toBe(2) // Initial request + retry after refresh

      // Verify CSRF token was updated
      expect(mockSetState).toHaveBeenCalledWith({ csrfToken: 'new-csrf-token' })
    })

    it('should handle unauthorized error (401)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } },
            { status: 401 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 401,
          data: {
            error: {
              code: 'UNAUTHORIZED',
              message: 'Unauthorized',
            },
          },
        },
      })
    })

    it('should handle validation error (400)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            {
              error: {
                code: 'VALIDATION_ERROR',
                message: 'Invalid input',
                details: { field: 'email' },
              },
            },
            { status: 400 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 400,
          data: {
            error: {
              code: 'VALIDATION_ERROR',
              message: 'Invalid input',
              details: { field: 'email' },
            },
          },
        },
      })
    })

    it('should handle conflict error (409)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'CONFLICT', message: 'Resource already exists' } },
            { status: 409 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 409,
          data: {
            error: {
              code: 'CONFLICT',
              message: 'Resource already exists',
            },
          },
        },
      })
    })

    it('should handle rate limit error (429)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'RATE_LIMIT_EXCEEDED', message: 'Too many requests' } },
            { status: 429 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 429,
          data: {
            error: {
              code: 'RATE_LIMIT_EXCEEDED',
              message: 'Too many requests',
            },
          },
        },
      })
    })

    it('should handle not found error (404)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'NOT_FOUND', message: 'Resource not found' } },
            { status: 404 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 404,
          data: {
            error: {
              code: 'NOT_FOUND',
              message: 'Resource not found',
            },
          },
        },
      })
    })

    it('should handle internal server error (500)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'INTERNAL_ERROR', message: 'Internal server error' } },
            { status: 500 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 500,
          data: {
            error: {
              code: 'INTERNAL_ERROR',
              message: 'Internal server error',
            },
          },
        },
      })
    })
  })

  describe('network errors', () => {
    it('should handle network error', async () => {
      // Mock a network error by returning an error response
      server.use(
        http.get('/api/v1/test', () => {
          // Simulate a network error
          return HttpResponse.error()
        })
      )

      await expect(apiClient.get('/test')).rejects.toThrow()
    })
  })
})
