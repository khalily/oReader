import { useInfiniteQuery, useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  Article,
  ListItemsOptions,
  ListItemsResponse,
  MarkAllReadResponse,
} from '@/types/feed'

// API functions
async function listItems(options?: ListItemsOptions): Promise<ListItemsResponse> {
  const queryParams = new URLSearchParams()
  if (options?.limit) queryParams.append('limit', options.limit.toString())
  if (options?.cursor) queryParams.append('cursor', options.cursor)
  if (options?.feed_id) queryParams.append('feed_id', options.feed_id)
  if (options?.starred !== undefined) queryParams.append('starred', options.starred.toString())
  if (options?.read !== undefined) queryParams.append('read', options.read.toString())
  if (options?.published_today !== undefined) queryParams.append('published_today', options.published_today.toString())

  const url = queryParams.toString() ? `/items?${queryParams.toString()}` : '/items'
  const response = await apiClient.get<ListItemsResponse>(url)
  return response.data
}

async function getItem(itemId: string): Promise<{ item: Article }> {
  const response = await apiClient.get<{ item: Article }>(`/items/${itemId}`)
  return response.data
}

async function toggleStar({ itemId, starred }: { itemId: string; starred: boolean }): Promise<{ item: Article }> {
  const response = await apiClient.put<{ item: Article }>(`/items/${itemId}/star`, { starred })
  return response.data
}

async function toggleRead({ itemId, read }: { itemId: string; read: boolean }): Promise<{ item: Article }> {
  const response = await apiClient.put<{ item: Article }>(`/items/${itemId}/read`, { read })
  return response.data
}

async function markAllRead(feedId: string): Promise<MarkAllReadResponse> {
  const response = await apiClient.post<MarkAllReadResponse>(`/feeds/${feedId}/mark-all-read`)
  return response.data
}

// React Query hooks
export function useItems() {
  const useListItems = (options?: ListItemsOptions) =>
    useQuery({
      queryKey: ['items', 'list', options],
      queryFn: () => listItems(options),
      staleTime: 2 * 60 * 1000, // 2 minutes
    })

  const useGetItem = (itemId: string | null) =>
    useQuery({
      queryKey: ['items', itemId],
      queryFn: () => getItem(itemId!),
      enabled: !!itemId,
      staleTime: 5 * 60 * 1000,
    })

  const useToggleStar = () =>
    useMutation({
      mutationFn: toggleStar,
    })

  const useToggleRead = () =>
    useMutation({
      mutationFn: toggleRead,
    })

  const useMarkAllRead = () =>
    useMutation({
      mutationFn: markAllRead,
    })

  const useListItemsInfinite = (options?: Omit<ListItemsOptions, 'cursor'>) =>
    useInfiniteQuery({
      queryKey: ['items', 'infinite', options],
      queryFn: ({ pageParam }) =>
        listItems({ ...options, cursor: pageParam as string | undefined }),
      initialPageParam: undefined as string | undefined,
      getNextPageParam: (lastPage: ListItemsResponse) =>
        lastPage.has_more ? lastPage.next_cursor : undefined,
      staleTime: 2 * 60 * 1000,
      maxPages: 10,
    })

  return {
    useListItems,
    useListItemsInfinite,
    useGetItem,
    useToggleStar,
    useToggleRead,
    useMarkAllRead,
  }
}
