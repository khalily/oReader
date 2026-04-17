import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  Category,
  CreateCategoryRequest,
  ListCategoriesResponse,
  MoveToCategoryRequest,
  RenameCategoryRequest,
} from '@/types/category'

// API functions
async function listCategories(type: 'feed' | 'paper'): Promise<ListCategoriesResponse> {
  const response = await apiClient.get<ListCategoriesResponse>(`/categories?type=${type}`)
  return response.data
}

async function createCategory(data: CreateCategoryRequest): Promise<{ category: Category }> {
  const response = await apiClient.post<{ category: Category }>('/categories', data)
  return response.data
}

async function renameCategory(categoryId: string, data: RenameCategoryRequest): Promise<{ category: Category }> {
  const response = await apiClient.put<{ category: Category }>(`/categories/${categoryId}/rename`, data)
  return response.data
}

async function deleteCategory(categoryId: string): Promise<void> {
  await apiClient.delete(`/categories/${categoryId}`)
}

async function moveFeedToCategory(feedId: string, data: MoveToCategoryRequest): Promise<void> {
  await apiClient.put(`/categories/feeds/${feedId}`, data)
}

async function movePaperToCategory(paperId: string, data: MoveToCategoryRequest): Promise<void> {
  await apiClient.put(`/categories/papers/${paperId}`, data)
}

// React Query hooks
export function useListCategories(type: 'feed' | 'paper') {
  return useQuery({
    queryKey: ['categories', type],
    queryFn: () => listCategories(type),
    staleTime: 60 * 1000, // 1 minute
  })
}

export function useCreateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateCategoryRequest) => createCategory(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
}

export function useRenameCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ categoryId, data }: { categoryId: string; data: RenameCategoryRequest }) =>
      renameCategory(categoryId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
}

export function useDeleteCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (categoryId: string) => deleteCategory(categoryId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
}

export function useMoveFeedToCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ feedId, data }: { feedId: string; data: MoveToCategoryRequest }) =>
      moveFeedToCategory(feedId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
}

export function useMovePaperToCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ paperId, data }: { paperId: string; data: MoveToCategoryRequest }) =>
      movePaperToCategory(paperId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
}
