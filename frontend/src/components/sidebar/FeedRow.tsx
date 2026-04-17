import { useState, useRef, useEffect } from 'react'
import { Rss, MoreHorizontal, FolderPlus, FolderMinus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { UserFeed } from '@/types/feed'
import type { Category } from '@/types/category'

interface FeedRowProps {
  feed: UserFeed
  isSelected: boolean
  onClick: (feedId: string) => void
  allFeedCategories: Category[]
  onMoveToCategory: (feedId: string, categoryId: string) => void
  onMoveToNewCategory: (feedId: string, categoryName: string) => void
}

export function FeedRow({
  feed,
  isSelected,
  onClick,
  allFeedCategories,
  onMoveToCategory,
  onMoveToNewCategory,
}: FeedRowProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [isNewCategoryInput, setIsNewCategoryInput] = useState(false)
  const [newCategoryName, setNewCategoryName] = useState('')
  const menuRef = useRef<HTMLDivElement>(null)
  const newCategoryInputRef = useRef<HTMLInputElement>(null)

  // Close menu on click outside
  useEffect(() => {
    if (!menuOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
        setIsNewCategoryInput(false)
        setNewCategoryName('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  // Focus new category input
  useEffect(() => {
    if (isNewCategoryInput && newCategoryInputRef.current) {
      newCategoryInputRef.current.focus()
    }
  }, [isNewCategoryInput])

  const handleAddToCategory = (categoryId: string) => {
    setMenuOpen(false)
    onMoveToCategory(feed.id, categoryId)
  }

  const handleNewCategoryKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      const trimmed = newCategoryName.trim()
      if (trimmed) {
        onMoveToNewCategory(feed.id, trimmed)
        setMenuOpen(false)
        setIsNewCategoryInput(false)
        setNewCategoryName('')
      }
    } else if (e.key === 'Escape') {
      setIsNewCategoryInput(false)
      setNewCategoryName('')
    }
  }

  const handleNewCategoryBlur = () => {
    const trimmed = newCategoryName.trim()
    if (trimmed) {
      onMoveToNewCategory(feed.id, trimmed)
    }
    setIsNewCategoryInput(false)
    setNewCategoryName('')
  }

  return (
    <div
      className={cn(
        'group flex items-center pl-7 pr-2 py-1 rounded-sm cursor-pointer hover:bg-accent/50',
        isSelected && 'bg-accent',
      )}
      onClick={() => onClick(feed.id)}
    >
      {feed.image_url ? (
        <img
          src={feed.image_url}
          alt=""
          className="h-3.5 w-3.5 rounded-sm shrink-0 mr-2"
          onError={(e) => {
            ;(e.target as HTMLImageElement).style.display = 'none'
          }}
        />
      ) : (
        <Rss className="h-3.5 w-3.5 shrink-0 mr-2 text-muted-foreground" />
      )}
      <span className="text-sm truncate flex-1">{feed.title}</span>
      {feed.unread_count > 0 && (
        <span className="text-xs text-muted-foreground bg-muted rounded-full px-1.5 min-w-[20px] text-center mr-1">
          {feed.unread_count > 99 ? '99+' : feed.unread_count}
        </span>
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
        {menuOpen && (
          <div className="absolute right-0 top-full mt-1 bg-popover border rounded-md shadow-md z-50 min-w-[140px]">
            {allFeedCategories.length > 0 && allFeedCategories.map((cat) => (
              <button
                key={cat.id}
                className="flex items-center w-full px-3 py-1.5 text-sm hover:bg-accent text-left"
                onClick={() => handleAddToCategory(cat.id)}
              >
                {cat.name}
              </button>
            ))}
            {allFeedCategories.length > 0 && <div className="border-t" />}
            {isNewCategoryInput ? (
              <input
                ref={newCategoryInputRef}
                className="w-full px-3 py-1.5 text-sm bg-background border-0 outline-none"
                placeholder="分类名称"
                value={newCategoryName}
                onChange={(e) => setNewCategoryName(e.target.value)}
                onKeyDown={handleNewCategoryKeyDown}
                onBlur={handleNewCategoryBlur}
              />
            ) : (
              <button
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-accent text-left"
                onClick={() => setIsNewCategoryInput(true)}
              >
                <FolderPlus className="h-3 w-3" />
                新建分类
              </button>
            )}
            {feed.category_id && (
              <>
                <div className="border-t" />
                <button
                  className="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-accent text-left"
                  onClick={() => {
                    setMenuOpen(false)
                    onMoveToCategory(feed.id, '')
                  }}
                >
                  <FolderMinus className="h-3 w-3" />
                  移除分类
                </button>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
