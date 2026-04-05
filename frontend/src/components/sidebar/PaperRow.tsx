import { FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Paper } from '@/types/paper'

interface PaperRowProps {
  paper: Paper
  isSelected: boolean
  onClick: (paperId: string) => void
}

export function PaperRow({ paper, isSelected, onClick }: PaperRowProps) {
  return (
    <div
      className={cn(
        'flex items-center pl-7 pr-2 py-1 rounded-sm cursor-pointer hover:bg-accent/50',
        isSelected && 'bg-accent',
      )}
      onClick={() => onClick(paper.id)}
    >
      <FileText className="h-3.5 w-3.5 shrink-0 mr-2 text-muted-foreground" />
      <span className="text-sm truncate flex-1">{paper.title}</span>
      {paper.status === 'processing' && (
        <span className="inline-block h-2 w-2 rounded-full bg-yellow-500 animate-pulse shrink-0 ml-1" />
      )}
      {paper.status === 'failed' && (
        <span className="inline-block h-2 w-2 rounded-full bg-red-500 shrink-0 ml-1" />
      )}
    </div>
  )
}
