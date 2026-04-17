import { useState, useRef, useEffect } from 'react'
import { ChevronRight, ChevronDown, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { Category } from '@/types/category'

interface CategoryRowProps {
  category: Category
  isExpanded: boolean
  itemCount: number
  onToggle: () => void
  onRename: (categoryId: string, name: string) => void
  onDelete: (categoryId: string) => void
}

export function CategoryRow({
  category,
  isExpanded,
  itemCount,
  onToggle,
  onRename,
  onDelete,
}: CategoryRowProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [isRenaming, setIsRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState(category.name)
  const menuRef = useRef<HTMLDivElement>(null)
  const renameInputRef = useRef<HTMLInputElement>(null)

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

  // Focus rename input when entering rename mode
  useEffect(() => {
    if (isRenaming && renameInputRef.current) {
      renameInputRef.current.focus()
      renameInputRef.current.select()
    }
  }, [isRenaming])

  const handleRenameSubmit = () => {
    const trimmed = renameValue.trim()
    if (trimmed && trimmed !== category.name) {
      onRename(category.id, trimmed)
    }
    setIsRenaming(false)
    setRenameValue(category.name)
  }

  const handleRenameKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleRenameSubmit()
    } else if (e.key === 'Escape') {
      setIsRenaming(false)
      setRenameValue(category.name)
    }
  }

  const handleDelete = () => {
    setMenuOpen(false)
    if (confirm(`确定删除分类「${category.name}」？`)) {
      onDelete(category.id)
    }
  }

  const handleStartRename = () => {
    setMenuOpen(false)
    setRenameValue(category.name)
    setIsRenaming(true)
  }

  if (isRenaming) {
    return (
      <div className="flex items-center px-2 py-1">
        <input
          ref={renameInputRef}
          className="flex-1 text-sm bg-background border rounded px-1.5 py-0.5 outline-none focus:ring-1 focus:ring-primary"
          value={renameValue}
          onChange={(e) => setRenameValue(e.target.value)}
          onBlur={handleRenameSubmit}
          onKeyDown={handleRenameKeyDown}
        />
      </div>
    )
  }

  return (
    <div
      className={cn(
        'group flex items-center px-2 py-1 rounded-sm cursor-pointer hover:bg-accent/50',
      )}
      onClick={onToggle}
    >
      {isExpanded ? (
        <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      ) : (
        <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      )}
      <span className="ml-1 text-sm font-medium truncate flex-1">{category.name}</span>
      <span className="text-xs text-muted-foreground mr-1">{itemCount}</span>
      <div ref={menuRef} className="relative">
        <Button
          variant="ghost"
          size="icon"
          className={cn(
            'h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity',
          )}
          onClick={(e) => {
            e.stopPropagation()
            setMenuOpen(!menuOpen)
          }}
        >
          <MoreHorizontal className="h-3.5 w-3.5" />
        </Button>
        {menuOpen && (
          <div className="absolute right-0 top-full mt-1 bg-popover border rounded-md shadow-md z-50 min-w-[120px]">
            <button
              className="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-accent text-left"
              onClick={handleStartRename}
            >
              <Pencil className="h-3 w-3" />
              重命名
            </button>
            <button
              className="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-accent text-left text-destructive"
              onClick={handleDelete}
            >
              <Trash2 className="h-3 w-3" />
              删除
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
