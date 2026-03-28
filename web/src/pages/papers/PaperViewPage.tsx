import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { PaperMeta } from '@/components/papers/PaperMeta'
import { usePapers } from '@/hooks/usePapers'

export default function PaperViewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { useGetPaper, useGetPaperStatus } = usePapers()

  const { data, isLoading, error } = useGetPaper(id ?? null)
  const { data: statusData } = useGetPaperStatus(id ?? null)

  const paper = data ?? null
  const status = statusData ?? null

  const handleDownload = () => {
    if (!id) return
    window.open(`/api/v1/papers/${id}/download`, '_blank')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !paper) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <AlertCircle className="h-12 w-12 text-muted-foreground mb-4" />
        <h1 className="text-2xl font-semibold mb-2">Paper Not Found</h1>
        <Button onClick={() => navigate('/papers')}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
      </div>
    )
  }

  // Show processing/pending state
  if (paper.status === 'pending' || paper.status === 'processing') {
    return (
      <div className="max-w-4xl mx-auto py-6 px-4">
        <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
        <Card>
          <CardContent className="p-8 text-center">
            <Loader2 className="h-12 w-12 animate-spin mx-auto mb-4 text-primary" />
            <h2 className="text-xl font-semibold mb-2">Converting Paper...</h2>
            <p className="text-muted-foreground mb-4">{paper.original_filename}</p>
            {status && (
              <div className="w-full bg-muted rounded-full h-2">
                <div
                  className="bg-primary h-2 rounded-full transition-all"
                  style={{ width: `${status.progress}%` }}
                />
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    )
  }

  // Show failed state
  if (paper.status === 'failed') {
    return (
      <div className="max-w-4xl mx-auto py-6 px-4">
        <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Papers
        </Button>
        <Card>
          <CardContent className="p-8 text-center">
            <AlertCircle className="h-12 w-12 mx-auto mb-4 text-destructive" />
            <h2 className="text-xl font-semibold mb-2">Conversion Failed</h2>
            <p className="text-muted-foreground mb-4">{paper.error || 'Unknown error'}</p>
            <Button onClick={() => navigate('/papers')}>
              Back to Papers
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Show completed paper
  return (
    <div className="max-w-4xl mx-auto py-6 px-4">
      <Button variant="ghost" onClick={() => navigate('/papers')} className="mb-6">
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to Papers
      </Button>

      {/* Metadata */}
      <Card className="mb-6">
        <CardContent className="p-6">
          <PaperMeta paper={paper} onDownload={handleDownload} />
        </CardContent>
      </Card>

      {/* Markdown Content */}
      {paper.markdown_content ? (
        <Card>
          <CardContent className="p-6">
            <MarkdownRenderer content={paper.markdown_content} />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-6 text-center text-muted-foreground">
            No content available
          </CardContent>
        </Card>
      )}
    </div>
  )
}
