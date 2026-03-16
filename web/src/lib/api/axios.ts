import axios, { type AxiosError } from 'axios'
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

// Response interceptor - Handle auth errors
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    const errorData = error.response?.data as { error?: { code?: string; message?: string } } | undefined

    // Handle token expired error
    if (error.response?.status === 401) {
      const errorCode = errorData?.error?.code

      if (errorCode === 'TOKEN_EXPIRED') {
        // Token expired - clear auth state
        // The refresh logic should be handled by a separate interceptor
        // or by checking the error code in the components
        useAuthStore.getState().clearUser()
      }
    }

    return Promise.reject(error)
  }
)

export default apiClient
