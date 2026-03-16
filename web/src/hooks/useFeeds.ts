import { useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  CreateFeedRequest,
  CreateFeedResponse,
  Feed,
  GetFeedResponse,
  ListFeedsResponse,
  RefreshFeedResponse,
  UserFeed,
} from '@/types/feed'

// API functions
async function listFeeds(params?: { limit?: number; offset?: number }): Promise<ListFeedsResponse> {
  const queryParams = new URLSearchParams()
  if (params?.limit) queryParams.append('limit', params.limit.toString())
  if (params?.offset) queryParams.append('offset', params.offset.toString())

  const url = queryParams.toString() ? `/feeds?${queryParams.toString()}` : '/feeds'
  const response = await apiClient.get<ListFeedsResponse>(url)
  return response.data
}

async function getFeed(feedId: string): Promise<GetFeedResponse> {
  const response = await apiClient.get<GetFeedResponse>(`/feeds/${feedId}`)
  return response.data
}

async function createFeed(data: CreateFeedRequest): Promise<CreateFeedResponse> {
  const response = await apiClient.post<CreateFeedResponse>('/feeds', data)
  return response.data
}

async function deleteFeed(feedId: string): Promise<void> {
  await apiClient.delete(`/feeds/${feedId}`)
}

async function refreshFeed(feedId: string): Promise<RefreshFeedResponse> {
  const response = await apiClient.post<RefreshFeedResponse>(`/feeds/${feedId}/refresh`)
  return response.data
}

// React Query hooks
export function useFeeds() {
  const useListFeeds = (params?: { limit?: number; offset?: number }) =>
    useQuery({
      queryKey: ['feeds', 'list', params],
      queryFn: () => listFeeds(params),
      staleTime: 5 * 60 * 1000, // 5 minutes
    })

  const useGetFeed = (feedId: string | null) =>
    useQuery({
      queryKey: ['feeds', feedId],
      queryFn: () => getFeed(feedId!),
      enabled: !!feedId,
      staleTime: 5 * 60 * 1000,
    })

  const useCreateFeed = () =>
    useMutation({
      mutationFn: createFeed,
    })

  const useDeleteFeed = () =>
    useMutation({
      mutationFn: deleteFeed,
    })

  const useRefreshFeed = () =>
    useMutation({
      mutationFn: refreshFeed,
    })

  return {
    useListFeeds,
    useGetFeed,
    useCreateFeed,
    useDeleteFeed,
    useRefreshFeed,
  }
}
