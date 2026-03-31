import { useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type { OpmlImportResponse, OpmlImportJobStatus } from '@/types/feed'

const MAX_OPML_SIZE = 1024 * 1024 // 1MB

async function importOpml(file: File): Promise<OpmlImportResponse> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await apiClient.post<OpmlImportResponse>('/opml/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return response.data
}

async function getImportStatus(jobId: string): Promise<OpmlImportJobStatus> {
  const response = await apiClient.get<OpmlImportJobStatus>(`/opml/import/${jobId}`)
  return response.data
}

async function exportOpml(): Promise<Blob> {
  const response = await apiClient.get('/opml/export', {
    responseType: 'blob',
  })
  return response.data as Blob
}

function triggerDownload(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.URL.revokeObjectURL(url)
}

export function useOPML() {
  const useImportOpml = () =>
    useMutation({
      mutationFn: (file: File) => {
        if (file.size > MAX_OPML_SIZE) {
          throw new Error('OPML file must be smaller than 1MB')
        }
        return importOpml(file)
      },
    })

  const useGetImportStatus = (jobId: string | null) =>
    useQuery({
      queryKey: ['opml', 'import', jobId],
      queryFn: () => getImportStatus(jobId!),
      enabled: !!jobId,
      refetchInterval: (query) => {
        const data = query.state.data
        if (!data) return false
        return data.status === 'pending' || data.status === 'processing' ? 2000 : false
      },
    })

  const useExportOpml = () =>
    useMutation({
      mutationFn: exportOpml,
    })

  return { useImportOpml, useGetImportStatus, useExportOpml, triggerDownload }
}
