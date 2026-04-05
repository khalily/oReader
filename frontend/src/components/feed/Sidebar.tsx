import { Home, Star, Rss, Menu, Calendar, BookOpen, Upload, Download } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { FeedList } from './FeedList'
import { cn } from '@/lib/utils'
import { SidebarHeader } from '@/components/sidebar/SidebarHeader'
import { FeedSection } from '@/components/sidebar/FeedSection'
import { PaperSection } from '@/components/sidebar/PaperSection'
import { useSidebarStore } from '@/stores/sidebarStore'
import type { UserFeed, StatsResponse } from '@/types/feed'
import type { Category } from '@/types/category'
import type { Paper } from '@/types/paper'

// ============================================================
// Legacy exports (kept for backward compatibility with ItemsPage/App)
// TODO: Remove in Task 9 (Three-Column Layout + Route Simplification)
// ============================================================

export type FilterType = 'all' | 'unread' | 'starred' | 'today'

const filterConfig = [
  { type: 'all' as FilterType, label: 'All', icon: Home },
  { type: 'unread' as FilterType, label: 'Unread', icon: Rss },
  { type: 'starred' as FilterType, label: 'Starred', icon: Star },
  { type: 'today' as FilterType, label: 'Today', icon: Calendar },
]

interface LegacySidebarProps {
  feeds: UserFeed[]
  selectedFeedId: string | null
  onFeedClick: (feedId: string) => void
  onAddFeed: () => void
  onDeleteFeed: (feedId: string) => void
  onRefreshFeed: (feedId: string) => void
  refreshingFeedIds: Set<string>
  filterType: FilterType
  onFilterChange: (filter: FilterType) => void
  totalUnread?: number
  stats?: StatsResponse
  isMobileOpen?: boolean
  onMobileClose?: () => void
  onImportOpml?: () => void
  onExportOpml?: () => void
  isExporting?: boolean
}

/**
 * @deprecated Use the new SidebarContent from this file instead.
 * Legacy sidebar content - kept for backward compatibility with ItemsPage.
 */
export function SidebarContent({
  feeds,
  selectedFeedId,
  onFeedClick,
  onAddFeed,
  onDeleteFeed,
  onRefreshFeed,
  refreshingFeedIds,
  filterType,
  onFilterChange,
  totalUnread = 0,
  stats,
  onImportOpml,
  onExportOpml,
  isExporting,
}: Omit<LegacySidebarProps, 'isMobileOpen' | 'onMobileClose'>) {
  const navigate = useNavigate()

  const handleDelete = (feedId: string) => {
    if (confirm('Are you sure you want to delete this feed?')) {
      onDeleteFeed(feedId)
    }
  }

  return (
    <>
      <div className="p-4 border-b">
        <h2 className="text-lg font-semibold mb-4">oReader</h2>

        <nav className="space-y-1">
          {filterConfig.map((filter) => {
            const Icon = filter.icon
            const isActive = filterType === filter.type
            const count = stats
              ? filter.type === 'all'
                ? stats.total
                : filter.type === 'unread'
                ? stats.unread
                : filter.type === 'starred'
                ? stats.starred
                : stats.today
              : filter.type === 'unread'
                ? totalUnread
                : feeds.reduce((sum, f) => sum + f.unread_count, 0)

            return (
              <Button
                key={filter.type}
                variant={isActive ? 'secondary' : 'ghost'}
                className={cn('w-full justify-start', isActive && 'bg-accent')}
                onClick={() => {
                  onFilterChange(filter.type)
                }}
              >
                <Icon className="w-4 h-4 mr-2" />
                {filter.label}
                {count > 0 && (
                  <Badge variant="default" className="ml-auto text-xs">
                    {count}
                  </Badge>
                )}
              </Button>
            )
          })}
        </nav>

        <div className="border-t my-2" />

        <Button
          variant="ghost"
          className="w-full justify-start"
          onClick={() => navigate('/papers')}
        >
          <BookOpen className="w-4 h-4 mr-2" />
          Papers
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-medium text-muted-foreground">Feeds</h3>
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={onImportOpml}
              title="Import OPML"
            >
              <Upload className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={onExportOpml}
              disabled={isExporting}
              title="Export OPML"
            >
              <Download className="h-3 w-3" />
            </Button>
          </div>
        </div>
        <FeedList
          feeds={feeds}
          onDelete={handleDelete}
          onRefresh={onRefreshFeed}
          onClick={onFeedClick}
          selectedFeedId={selectedFeedId}
          refreshingFeedIds={refreshingFeedIds}
          onAddFeed={onAddFeed}
        />
      </div>
    </>
  )
}

/**
 * @deprecated Use the new Sidebar from this file instead.
 * Legacy Sidebar component - kept for backward compatibility with ItemsPage.
 */
export function LegacySidebar({
  feeds,
  selectedFeedId,
  onFeedClick,
  onAddFeed,
  onDeleteFeed,
  onRefreshFeed,
  refreshingFeedIds,
  filterType,
  onFilterChange,
  totalUnread = 0,
  stats,
  isMobileOpen: _isMobileOpen = false, // eslint-disable-line @typescript-eslint/no-unused-vars
  onMobileClose: _onMobileClose, // eslint-disable-line @typescript-eslint/no-unused-vars
  onImportOpml,
  onExportOpml,
  isExporting,
}: LegacySidebarProps) {
  return (
    <>
      {/* Desktop Sidebar */}
      <aside className="hidden md:flex w-64 border-r bg-background flex-shrink-0 flex-col h-full">
        <SidebarContent
          feeds={feeds}
          selectedFeedId={selectedFeedId}
          onFeedClick={onFeedClick}
          onAddFeed={onAddFeed}
          onDeleteFeed={onDeleteFeed}
          onRefreshFeed={onRefreshFeed}
          refreshingFeedIds={refreshingFeedIds}
          filterType={filterType}
          onFilterChange={onFilterChange}
          totalUnread={totalUnread}
          stats={stats}
          onImportOpml={onImportOpml}
          onExportOpml={onExportOpml}
          isExporting={isExporting}
        />
      </aside>

      {/* Mobile Sidebar - rendered in drawer by parent */}
    </>
  )
}

// Backward-compatible alias
export const Sidebar = LegacySidebar

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
