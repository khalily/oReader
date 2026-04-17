import { useEffect, useRef } from 'react'
import axios from 'axios'
import { setCsrfToken } from '@/lib/api/axios'
import { useAuthStore } from '@/stores/authStore'

// Refresh 3 minutes before the access token expires (15 min TTL)
const REFRESH_INTERVAL_MS = 12 * 60 * 1000

/**
 * Proactive token refresh hook.
 *
 * Starts a timer when the user is authenticated to silently refresh
 * the access token before it expires. Failures are silently ignored
 * so the reactive (401 interceptor) refresh remains as fallback.
 */
export function useTokenRefresh(): void {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (!isAuthenticated) {
      // Clear any existing timer when logged out
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }

    intervalRef.current = setInterval(async () => {
      try {
        const response = await axios.post(
          '/api/v1/auth/refresh',
          {},
          { withCredentials: true },
        )
        const { csrf_token } = response.data
        if (csrf_token) {
          setCsrfToken(csrf_token)
          useAuthStore.setState({ csrfToken: csrf_token })
        }
      } catch {
        // Intentionally not clearing auth state on proactive refresh failure.
        // The reactive refresh (axios 401 interceptor) will handle it later.
      }
    }, REFRESH_INTERVAL_MS)

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [isAuthenticated])
}
