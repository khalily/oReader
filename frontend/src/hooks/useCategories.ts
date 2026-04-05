import { useMutation, useQuery } from '@tanstack/react-query'
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
export function useCategories() {
  const useListCategories = (type: 'feed' | 'paper') =>
    useQuery({
      queryKey: ['categories', type],
      queryFn: () => listCategories(type),
      staleTime: 60 * 1000, // 1 minute
    })

  const useCreateCategory = () =>
    useMutation({
      mutationFn: (data: CreateCategoryRequest) => createCategory(data),
    })

  const useRenameCategory = () =>
    useMutation({
      mutationFn: ({ categoryId, data }: { categoryId: string; data: RenameCategoryRequest }) =>
        renameCategory(categoryId, data),
    })

  const useDeleteCategory = () =>
    useMutation({
      mutationFn: (categoryId: string) => deleteCategory(categoryId),
    })

  const useMoveFeedToCategory = () =>
    useMutation({
      mutationFn: ({ feedId, data }: { feedId: string; data: MoveToCategoryRequest }) =>
        moveFeedToCategory(feedId, data),
    })

  const useMovePaperToCategory = () =>
    useMutation({
      mutationFn: ({ paperId, data }: { paperId: string; data: MoveToCategoryRequest }) =>
        movePaperToCategory(paperId, data),
    })

  return {
    useListCategories,
    useCreateCategory,
    useRenameCategory,
    useDeleteCategory,
    useMoveFeedToCategory,
    useMovePaperToCategory,
  }
}
