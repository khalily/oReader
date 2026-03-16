import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient, { setCsrfToken } from '@/lib/api/axios'
import { useAuthStore } from '@/stores/authStore'
import type { LoginRequest, RegisterRequest, AuthResponse, User } from '@/types'

// API functions
async function login(data: LoginRequest): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/auth/login', data)
  return response.data
}

async function register(data: RegisterRequest): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/auth/register', data)
  return response.data
}

async function logout(): Promise<{ success: boolean }> {
  const response = await apiClient.post<{ success: boolean }>('/auth/logout')
  return response.data
}

async function getCurrentUser(): Promise<User> {
  const response = await apiClient.get<User>('/auth/me')
  return response.data
}

// React Query hooks
export function useAuth() {
  const queryClient = useQueryClient()
  const setUser = useAuthStore((state) => state.setUser)
  const clearUser = useAuthStore((state) => state.clearUser)

  const useLogin = () =>
    useMutation({
      mutationFn: login,
      onSuccess: (data) => {
        setUser(data.user, data.csrf_token)
        setCsrfToken(data.csrf_token)
        queryClient.invalidateQueries({ queryKey: ['currentUser'] })
      },
    })

  const useRegister = () =>
    useMutation({
      mutationFn: register,
      onSuccess: (data) => {
        setUser(data.user, data.csrf_token)
        setCsrfToken(data.csrf_token)
        queryClient.invalidateQueries({ queryKey: ['currentUser'] })
      },
    })

  const useLogout = () =>
    useMutation({
      mutationFn: logout,
      onSuccess: () => {
        clearUser()
        setCsrfToken(null)
        queryClient.clear()
      },
    })

  const useMe = () =>
    useQuery({
      queryKey: ['currentUser'],
      queryFn: getCurrentUser,
      retry: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    })

  return {
    useLogin,
    useRegister,
    useLogout,
    useMe,
  }
}
