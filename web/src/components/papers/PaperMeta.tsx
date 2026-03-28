import { Calendar, Users, ExternalLink, Download, Tag } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Paper } from '@/types/paper'

function parseJsonArray(str: string | null): string[] {
  if (!str) return []
  try { return JSON.parse(str) } catch { return [] }
}

interface PaperMetaProps {
  paper: Paper
  onDownload: () => void
}

export function PaperMeta({ paper, onDownload }: PaperMetaProps) {
  const authors = parseJsonArray(paper.authors)
  const keywords = parseJsonArray(paper.keywords)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl md:text-3xl font-bold">{paper.title || 'Untitled'}</h1>

      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        {authors.length > 0 && (
          <div className="flex items-center gap-1">
            <Users className="h-4 w-4" />
            <span>{authors.join(', ')}</span>
          </div>
        )}
        {paper.published_year && (
          <div className="flex items-center gap-1">
            <Calendar className="h-4 w-4" />
            <span>{paper.published_year}</span>
          </div>
        )}
        {paper.doi && (
          <div className="flex items-center gap-1">
            <ExternalLink className="h-4 w-4" />
            <a
              href={`https://doi.org/${paper.doi}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              DOI: {paper.doi}
            </a>
          </div>
        )}
      </div>

      {paper.abstract && (
        <div className="border-l-4 border-muted pl-4">
          <h3 className="font-medium mb-1">Abstract</h3>
          <p className="text-sm text-muted-foreground">{paper.abstract}</p>
        </div>
      )}

      {keywords.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {keywords.map((kw) => (
            <Badge key={kw} variant="secondary" className="flex items-center gap-1">
              <Tag className="h-3 w-3" />
              {kw}
            </Badge>
          ))}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onDownload}>
          <Download className="h-4 w-4 mr-2" />
          Download PDF
        </Button>
        <span className="text-xs text-muted-foreground">
          {(paper.pdf_size / 1024 / 1024).toFixed(2)} MB
        </span>
      </div>
    </div>
  )
}
