import { useState, useCallback } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { Upload, Search, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PaperList } from '@/components/papers/PaperList'
import { PaperUpload } from '@/components/papers/PaperUpload'
import { usePapers } from '@/hooks/usePapers'
import { useToast } from '@/components/ui/toast'
import type { ListPapersOptions } from '@/types/paper'

export function PapersPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const toast = useToast()
  const [isUploadOpen, setIsUploadOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [selectedTag, setSelectedTag] = useState(searchParams.get('tag') || '')

  const { useListPapers, useDeletePaper, useRetryPaper, useListTags } = usePapers()

  const listOptions: ListPapersOptions = {
    page: parseInt(searchParams.get('page') || '1'),
    per_page: 20,
    q: searchQuery || undefined,
    tag: selectedTag || undefined,
    sort: 'created_at',
    order: 'desc',
  }

  const { data: papersData, isLoading, refetch } = useListPapers(listOptions)
  const { data: tagsData } = useListTags()
  const deletePaper = useDeletePaper()
  const retryPaper = useRetryPaper()

  const papers = papersData?.papers ?? []
  const total = papersData?.total ?? 0

  const handleSearch = useCallback((query: string) => {
    setSearchQuery(query)
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (query) next.set('q', query)
      else next.delete('q')
      next.set('page', '1')
      return next
    })
  }, [setSearchParams])

  const handlePaperClick = useCallback((paperId: string) => {
    navigate(`/papers/${paperId}`)
  }, [navigate])

  const handleRetry = useCallback((paperId: string) => {
    retryPaper.mutate(paperId, {
      onSuccess: () => {
        toast.showSuccess('Retry started')
        refetch()
      },
      onError: () => {
        toast.showError('Retry failed')
      },
    })
  }, [retryPaper, refetch, toast])

  const handleDelete = useCallback((paperId: string) => {
    if (!confirm('Delete this paper?')) return
    deletePaper.mutate(paperId, {
      onSuccess: () => {
        toast.showSuccess('Paper deleted')
        refetch()
      },
      onError: () => {
        toast.showError('Failed to delete paper')
      },
    })
  }, [deletePaper, refetch, toast])

  const handlePageChange = (page: number) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('page', page.toString())
      return next
    })
  }

  const currentPage = parseInt(searchParams.get('page') || '1')
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="max-w-5xl mx-auto py-6 px-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" onClick={() => navigate('/items')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <h1 className="text-2xl font-bold">Papers</h1>
          <span className="text-sm text-muted-foreground">({total})</span>
        </div>
        <Button onClick={() => setIsUploadOpen(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Upload PDF
        </Button>
      </div>

      {/* Search & Filters */}
      <div className="flex gap-3 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search papers..."
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        {tagsData?.tags && tagsData.tags.length > 0 && (
          <div className="flex gap-2 overflow-x-auto">
            {tagsData.tags.slice(0, 5).map((tag) => (
              <Button
                key={tag}
                variant={selectedTag === tag ? 'secondary' : 'outline'}
                size="sm"
                onClick={() => {
                  const newTag = selectedTag === tag ? '' : tag
                  setSelectedTag(newTag)
                  setSearchParams((prev) => {
                    const next = new URLSearchParams(prev)
                    if (newTag) next.set('tag', newTag)
                    else next.delete('tag')
                    next.set('page', '1')
                    return next
                  })
                }}
              >
                {tag}
              </Button>
            ))}
          </div>
        )}
      </div>

      {/* Paper List */}
      <PaperList
        papers={papers}
        isLoading={isLoading}
        onPaperClick={handlePaperClick}
        onRetry={handleRetry}
        onDelete={handleDelete}
      />

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 mt-6">
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage <= 1}
            onClick={() => handlePageChange(currentPage - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {currentPage} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={currentPage >= totalPages}
            onClick={() => handlePageChange(currentPage + 1)}
          >
            Next
          </Button>
        </div>
      )}

      {/* Upload Modal */}
      <PaperUpload
        open={isUploadOpen}
        onOpenChange={setIsUploadOpen}
        onSuccess={() => refetch()}
      />
    </div>
  )
}

export default PapersPage
