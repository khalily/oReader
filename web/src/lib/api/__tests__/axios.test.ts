import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { apiClient, setCsrfToken } from '../axios'
import { server } from '../../../test/server'
import { http, HttpResponse } from 'msw'

// Mock authStore
vi.mock('../../stores/authStore', () => ({
  authStore: {
    getState: vi.fn(() => ({
      getCsrfToken: () => 'mock-csrf-token',
      setUser: vi.fn(),
      clearUser: vi.fn(),
    })),
  },
}))

describe('axios client', () => {
  beforeEach(() => {
    // Reset CSRF token before each test
    setCsrfToken(null)
  })

  afterEach(() => {
    vi.clearAllMocks()
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
      expect(response.data.csrfToken).toBe('test-csrf-token')
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

    it('should handle token expired error (401)', async () => {
      server.use(
        http.get('/api/v1/test', () => {
          return HttpResponse.json(
            { error: { code: 'TOKEN_EXPIRED', message: 'Token expired' } },
            { status: 401 }
          )
        })
      )

      await expect(apiClient.get('/test')).rejects.toMatchObject({
        response: {
          status: 401,
          data: {
            error: {
              code: 'TOKEN_EXPIRED',
              message: 'Token expired',
            },
          },
        },
      })
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
    it('should handle network timeout', async () => {
      // Mock a timeout by not responding
      server.use(
        http.get('/api/v1/test', async () => {
          // Never resolve
          await new Promise(() => {})
          return HttpResponse.json({})
        })
      )

      // Configure short timeout for this test
      const originalTimeout = apiClient.defaults.timeout
      apiClient.defaults.timeout = 100

      await expect(apiClient.get('/test')).rejects.toThrow()

      // Restore original timeout
      apiClient.defaults.timeout = originalTimeout
    })
  })
})
