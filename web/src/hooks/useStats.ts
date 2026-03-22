import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type { StatsResponse } from '@/types/feed'

// API function
async function getStats(): Promise<StatsResponse> {
  const response = await apiClient.get<StatsResponse>('/stats')
  return response.data
}

// React Query hook
export function useStats() {
  const useGetStats = () =>
    useQuery({
      queryKey: ['stats'],
      queryFn: getStats,
      staleTime: 30 * 1000, // 30 seconds
      refetchOnWindowFocus: true,
    })

  return {
    useGetStats,
  }
}
