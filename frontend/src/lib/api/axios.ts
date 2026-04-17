import axios, { type AxiosError, type AxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/authStore'

// Create axios instance
export const apiClient = axios.create({
  baseURL: '/api/v1',
  withCredentials: true, // Send cookies
  headers: {
    'Content-Type': 'application/json',
  },
})

// CSRF token management
let currentCsrfToken: string | null = null

export function setCsrfToken(token: string | null): void {
  currentCsrfToken = token

  if (token) {
    apiClient.defaults.headers.common['X-CSRF-Token'] = token
  } else {
    delete apiClient.defaults.headers.common['X-CSRF-Token']
  }
}

// Request interceptor - Add CSRF token
apiClient.interceptors.request.use(
  (config) => {
    // Get CSRF token from store or use current token
    const storeToken = useAuthStore.getState().getCsrfToken()
    const token = storeToken || currentCsrfToken

    if (token) {
      config.headers['X-CSRF-Token'] = token
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Silent Refresh: Token refresh state management
let isRefreshing = false
let failedQueue: Array<{
  resolve: (value?: unknown) => void
  reject: (error?: unknown) => void
}> = []

// Process queued requests after refresh attempt
const processQueue = (error: AxiosError | null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve()
    }
  })
  failedQueue = []
}

// Response interceptor - Handle auth errors with Silent Refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }
    const errorData = error.response?.data as { error?: { code?: string; message?: string } } | undefined

    // Handle TOKEN_EXPIRED with Silent Refresh
    if (
      error.response?.status === 401 &&
      errorData?.error?.code === 'TOKEN_EXPIRED' &&
      !originalRequest._retry
    ) {
      // If already refreshing, queue this request
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then(() => apiClient(originalRequest))
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        // Call refresh endpoint - cookies are sent automatically via withCredentials
        const response = await axios.post(
          '/api/v1/auth/refresh',
          {},
          { withCredentials: true }
        )

        const { csrf_token } = response.data

        // Update CSRF token in both local storage and auth store
        if (csrf_token) {
          setCsrfToken(csrf_token)
          useAuthStore.setState({ csrfToken: csrf_token })
        }

        // Process all queued requests - they will now succeed with new token
        processQueue(null)

        // Retry the original request
        return apiClient(originalRequest)
      } catch (refreshError) {
        // Refresh failed - clear auth state and reject all queued requests
        console.warn('[auth] Reactive token refresh failed:', refreshError)
        processQueue(refreshError as AxiosError)
        useAuthStore.getState().clearUser()
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    // Other 401 errors (invalid token, no token, etc.) - clear auth state
    if (error.response?.status === 401) {
      useAuthStore.getState().clearUser()
    }

    return Promise.reject(error)
  }
)

export default apiClient
