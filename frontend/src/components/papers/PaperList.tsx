import { FileText, Loader2, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import type { Paper, PaperStatus } from '@/types/paper'
import { parseJsonArray } from '@/lib/utils'

interface PaperListProps {
  papers?: Paper[] | null
  isLoading?: boolean
  onPaperClick: (paperId: string) => void
  onRetry: (paperId: string) => void
  onDelete: (paperId: string) => void
}

function StatusBadge({ status }: { status: PaperStatus }) {
  switch (status) {
    case 'completed':
      return <Badge variant="secondary">Completed</Badge>
    case 'processing':
      return <Badge className="bg-blue-500 text-white">Processing</Badge>
    case 'pending':
      return <Badge className="bg-yellow-500 text-white">Pending</Badge>
    case 'failed':
      return <Badge variant="destructive">Failed</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export function PaperList({ papers = [], isLoading, onPaperClick, onRetry, onDelete }: PaperListProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!papers || papers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
        <FileText className="h-12 w-12 mb-4" />
        <p>No papers yet</p>
        <p className="text-sm mt-1">Upload a PDF to get started</p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {papers.map((paper) => (
        <div
          key={paper.id}
          className="p-4 rounded-lg border hover:bg-accent/50 cursor-pointer transition-colors"
          onClick={() => onPaperClick(paper.id)}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <h3 className="font-medium truncate">{paper.title || paper.original_filename}</h3>
              </div>
              <div className="text-sm text-muted-foreground">
                {parseJsonArray(paper.authors).length > 0 && (
                  <span className="mr-2">{parseJsonArray(paper.authors).slice(0, 3).join(', ')}</span>
                )}
                {paper.published_year && <span>{paper.published_year}</span>}
              </div>
              {paper.abstract && (
                <p className="text-sm text-muted-foreground mt-1 line-clamp-2">{paper.abstract}</p>
              )}
              {parseJsonArray(paper.keywords).length > 0 && (
                <div className="flex gap-1 mt-2 flex-wrap">
                  {parseJsonArray(paper.keywords).slice(0, 3).map((kw) => (
                    <Badge key={kw} variant="outline" className="text-xs">{kw}</Badge>
                  ))}
                </div>
              )}
            </div>
            <div className="flex flex-col items-end gap-2 flex-shrink-0">
              <StatusBadge status={paper.status} />
              {paper.status === 'failed' && (
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => { e.stopPropagation(); onRetry(paper.id) }}
                  >
                    <RefreshCw className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => { e.stopPropagation(); onDelete(paper.id) }}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              )}
              {paper.status === 'failed' && paper.error && (
                <p className="text-xs text-destructive max-w-48 truncate">{paper.error}</p>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
