import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import axios from 'axios'
import { useTokenRefresh } from '../useTokenRefresh'

// Mock axios at the module level
vi.mock('axios', () => ({
  default: {
    post: vi.fn(),
  },
}))

const mockedAxiosPost = vi.mocked(axios.post)

const { mockSetCsrfToken, mockSetState } = vi.hoisted(() => ({
  mockSetCsrfToken: vi.fn(),
  mockSetState: vi.fn(),
}))

vi.mock('@/lib/api/axios', () => ({
  setCsrfToken: mockSetCsrfToken,
}))

vi.mock('@/stores/authStore', () => ({
  useAuthStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        isAuthenticated: true,
        setState: mockSetState,
      }),
    { setState: mockSetState },
  ),
}))

describe('useTokenRefresh', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should call refresh endpoint every 12 minutes', async () => {
    mockedAxiosPost.mockResolvedValue({
      data: { csrf_token: 'new-csrf-token' },
    })

    renderHook(() => useTokenRefresh())

    // Fast-forward 12 minutes
    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })

    expect(mockedAxiosPost).toHaveBeenCalledWith(
      '/api/v1/auth/refresh',
      {},
      { withCredentials: true },
    )
    expect(mockSetCsrfToken).toHaveBeenCalledWith('new-csrf-token')
    expect(mockSetState).toHaveBeenCalledWith({ csrfToken: 'new-csrf-token' })
  })

  it('should update CSRF token on successful refresh', async () => {
    mockedAxiosPost.mockResolvedValue({
      data: { csrf_token: 'updated-csrf' },
    })

    renderHook(() => useTokenRefresh())

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })

    expect(mockSetCsrfToken).toHaveBeenCalledWith('updated-csrf')
  })

  it('should not clear auth state on refresh failure', async () => {
    mockedAxiosPost.mockRejectedValue(new Error('Network error'))

    // Should not throw even when refresh fails
    const { result } = renderHook(() => useTokenRefresh())

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })

    // Hook should not have thrown — component still mounted
    expect(result.error).toBeUndefined()
    expect(mockSetCsrfToken).not.toHaveBeenCalled()
  })

  it('should clear interval on unmount', async () => {
    mockedAxiosPost.mockResolvedValue({
      data: { csrf_token: 'csrf-1' },
    })

    const { unmount } = renderHook(() => useTokenRefresh())

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })
    expect(mockedAxiosPost).toHaveBeenCalledTimes(1)

    unmount()

    // Advance past another interval — no additional call
    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })
    expect(mockedAxiosPost).toHaveBeenCalledTimes(1)
  })

  it('should refresh multiple times at correct intervals', async () => {
    mockedAxiosPost.mockResolvedValue({
      data: { csrf_token: 'csrf-token' },
    })

    renderHook(() => useTokenRefresh())

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })
    expect(mockedAxiosPost).toHaveBeenCalledTimes(1)

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })
    expect(mockedAxiosPost).toHaveBeenCalledTimes(2)

    await act(async () => {
      vi.advanceTimersByTime(12 * 60 * 1000)
    })
    expect(mockedAxiosPost).toHaveBeenCalledTimes(3)
  })
})
