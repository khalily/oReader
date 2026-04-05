import { useState, useRef, useEffect } from 'react'
import { FileText, MoreHorizontal, FolderMinus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { Paper } from '@/types/paper'

interface PaperRowProps {
  paper: Paper
  isSelected: boolean
  onClick: (paperId: string) => void
  onRemoveFromCategory?: (paperId: string) => void
}

export function PaperRow({ paper, isSelected, onClick, onRemoveFromCategory }: PaperRowProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close menu on click outside
  useEffect(() => {
    if (!menuOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  return (
    <div
      className={cn(
        'group flex items-center pl-7 pr-2 py-1 rounded-sm cursor-pointer hover:bg-accent/50',
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
      <div ref={menuRef} className="relative">
        <Button
          variant="ghost"
          size="icon"
          className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={(e) => {
            e.stopPropagation()
            setMenuOpen(!menuOpen)
          }}
        >
          <MoreHorizontal className="h-3.5 w-3.5" />
        </Button>
        {menuOpen && paper.category_id && onRemoveFromCategory && (
          <div className="absolute right-0 top-full mt-1 bg-popover border rounded-md shadow-md z-50 min-w-[120px]">
            <button
              className="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-accent text-left"
              onClick={() => {
                setMenuOpen(false)
                onRemoveFromCategory(paper.id)
              }}
            >
              <FolderMinus className="h-3 w-3" />
              移除分类
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
