import { Menu } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SidebarHeader } from '@/components/sidebar/SidebarHeader'
import { FeedSection } from '@/components/sidebar/FeedSection'
import { PaperSection } from '@/components/sidebar/PaperSection'
import { useSidebarStore } from '@/stores/sidebarStore'
import type { UserFeed } from '@/types/feed'
import type { Category } from '@/types/category'
import type { Paper } from '@/types/paper'

/**
 * Mobile menu button component
 */
export function MobileMenuButton({ onClick, unreadCount = 0 }: { onClick: () => void; unreadCount?: number }) {
  return (
    <Button
      variant="ghost"
      size="icon"
      className="md:hidden"
      onClick={onClick}
      aria-label="Open menu"
    >
      <div className="relative">
        <Menu className="h-5 w-5" />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-red-500 text-[10px] font-medium text-white flex items-center justify-center">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </div>
    </Button>
  )
}

// ============================================================
// New category-based sidebar components
// ============================================================

export interface NewSidebarProps {
  feedCategories: Category[]
  paperCategories: Category[]
  feeds: UserFeed[]
  papers: Paper[]
  onAddClick: () => void
  onFeedClick: (feedId: string) => void
  onPaperClick: (paperId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
  onMoveFeedToCategory: (feedId: string, categoryId: string) => void
  onMoveFeedToNewCategory: (feedId: string, categoryName: string) => void
  onRemovePaperFromCategory?: (paperId: string) => void
  isMobileOpen?: boolean
  onMobileClose?: () => void
}

export function NewSidebarContent(props: Omit<NewSidebarProps, 'isMobileOpen' | 'onMobileClose'>) {
  const { selectedFeedId, selectedPaperId, expandedCategoryIds, toggleCategory } =
    useSidebarStore()

  return (
    <>
      <SidebarHeader onAddClick={props.onAddClick} />
      <div className="flex-1 overflow-y-auto px-2 py-1">
        <FeedSection
          categories={props.feedCategories}
          feeds={props.feeds}
          selectedFeedId={selectedFeedId}
          expandedCategoryIds={expandedCategoryIds}
          allFeedCategories={props.feedCategories}
          onToggleCategory={toggleCategory}
          onFeedClick={props.onFeedClick}
          onRenameCategory={props.onRenameCategory}
          onDeleteCategory={props.onDeleteCategory}
          onMoveFeedToCategory={props.onMoveFeedToCategory}
          onMoveFeedToNewCategory={props.onMoveFeedToNewCategory}
        />
        <div className="border-t my-2" />
        <PaperSection
          categories={props.paperCategories}
          papers={props.papers}
          selectedPaperId={selectedPaperId}
          expandedCategoryIds={expandedCategoryIds}
          onToggleCategory={toggleCategory}
          onPaperClick={props.onPaperClick}
          onRenameCategory={props.onRenameCategory}
          onDeleteCategory={props.onDeleteCategory}
          onRemovePaperFromCategory={props.onRemovePaperFromCategory}
        />
      </div>
    </>
  )
}

export function NewSidebar(props: NewSidebarProps) {
  return (
    <aside className="hidden md:flex w-[260px] border-r bg-background flex-shrink-0 flex-col h-full">
      <NewSidebarContent {...props} />
    </aside>
  )
}
