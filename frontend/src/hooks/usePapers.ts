import { useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  ListPapersOptions,
  ListPapersResponse,
  Paper,
  PaperStatusResponse,
  UpdatePaperRequest,
  UpdateTagsRequest,
  UploadPaperResponse,
} from '@/types/paper'

// API functions
async function uploadPaper(file: File): Promise<UploadPaperResponse> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await apiClient.post<UploadPaperResponse>('/papers/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return response.data
}

async function listPapers(options?: ListPapersOptions): Promise<ListPapersResponse> {
  const params = new URLSearchParams()
  if (options?.page) params.append('page', options.page.toString())
  if (options?.per_page) params.append('per_page', options.per_page.toString())
  if (options?.q) params.append('q', options.q)
  if (options?.year) params.append('year', options.year)
  if (options?.tag) params.append('tag', options.tag)
  if (options?.status) params.append('status', options.status)
  if (options?.sort) params.append('sort', options.sort)
  if (options?.order) params.append('order', options.order)

  const url = params.toString() ? `/papers?${params.toString()}` : '/papers'
  const response = await apiClient.get<ListPapersResponse>(url)
  return response.data
}

async function getPaper(paperId: string): Promise<Paper> {
  const response = await apiClient.get<{ paper: Paper }>(`/papers/${paperId}`)
  return response.data.paper
}

async function getPaperStatus(paperId: string): Promise<PaperStatusResponse> {
  const response = await apiClient.get<PaperStatusResponse>(`/papers/${paperId}/status`)
  return response.data
}

async function updatePaper(paperId: string, data: UpdatePaperRequest): Promise<Paper> {
  const response = await apiClient.put<{ paper: Paper }>(`/papers/${paperId}`, data)
  return response.data.paper
}

async function updateTags(paperId: string, data: UpdateTagsRequest): Promise<void> {
  await apiClient.put(`/papers/${paperId}/tags`, data)
}

async function deletePaper(paperId: string): Promise<void> {
  await apiClient.delete(`/papers/${paperId}`)
}

async function retryPaper(paperId: string): Promise<Paper> {
  const response = await apiClient.post<{ paper: Paper }>(`/papers/${paperId}/retry`)
  return response.data.paper
}

async function listTags(): Promise<{ tags: string[] }> {
  const response = await apiClient.get<{ tags: string[] }>('/papers/tags')
  return response.data
}

async function downloadPaper(paperId: string): Promise<Blob> {
  const response = await apiClient.get(`/papers/${paperId}/download`, {
    responseType: 'blob',
  })
  return response.data as Blob
}

// React Query hooks
export function usePapers() {
  const useUploadPaper = () =>
    useMutation({
      mutationFn: uploadPaper,
    })

  const useListPapers = (options?: ListPapersOptions) =>
    useQuery({
      queryKey: ['papers', 'list', options],
      queryFn: () => listPapers(options),
      staleTime: 2 * 60 * 1000,
    })

  const useGetPaper = (paperId: string | null) =>
    useQuery({
      queryKey: ['papers', paperId],
      queryFn: () => getPaper(paperId!),
      enabled: !!paperId,
      staleTime: 5 * 60 * 1000,
    })

  const useGetPaperStatus = (paperId: string | null) =>
    useQuery({
      queryKey: ['papers', 'status', paperId],
      queryFn: () => getPaperStatus(paperId!),
      enabled: !!paperId,
      refetchInterval: (query) => {
        const data = query.state.data
        if (!data) return false
        return data.status === 'pending' || data.status === 'processing' ? 3000 : false
      },
    })

  const useUpdatePaper = () =>
    useMutation({
      mutationFn: ({ paperId, data }: { paperId: string; data: UpdatePaperRequest }) =>
        updatePaper(paperId, data),
    })

  const useUpdateTags = () =>
    useMutation({
      mutationFn: ({ paperId, tags }: { paperId: string; tags: string[] }) =>
        updateTags(paperId, { tags }),
    })

  const useDeletePaper = () =>
    useMutation({
      mutationFn: deletePaper,
    })

  const useRetryPaper = () =>
    useMutation({
      mutationFn: retryPaper,
    })

  const useListTags = () =>
    useQuery({
      queryKey: ['papers', 'tags'],
      queryFn: listTags,
      staleTime: 5 * 60 * 1000,
    })

  const useDownloadPaper = () =>
    useMutation({
      mutationFn: downloadPaper,
    })

  return {
    useUploadPaper,
    useListPapers,
    useGetPaper,
    useGetPaperStatus,
    useUpdatePaper,
    useUpdateTags,
    useDeletePaper,
    useRetryPaper,
    useListTags,
    useDownloadPaper,
  }
}
