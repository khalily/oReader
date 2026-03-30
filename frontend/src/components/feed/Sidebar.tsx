import { Home, Star, Rss, Menu, Calendar, BookOpen } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { FeedList } from './FeedList'
import { cn } from '@/lib/utils'
import type { UserFeed, StatsResponse } from '@/types/feed'

export type FilterType = 'all' | 'unread' | 'starred' | 'today'

const filterConfig = [
  { type: 'all' as FilterType, label: 'All', icon: Home },
  { type: 'unread' as FilterType, label: 'Unread', icon: Rss },
  { type: 'starred' as FilterType, label: 'Starred', icon: Star },
  { type: 'today' as FilterType, label: 'Today', icon: Calendar },
]

interface SidebarProps {
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
}

/**
 * Sidebar content - can be used in desktop sidebar or mobile drawer
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
}: Omit<SidebarProps, 'isMobileOpen' | 'onMobileClose'>) {
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
                  // Close mobile drawer after selection
                  if (window.innerWidth < 768) {
                    // Triggered by parent
                  }
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
        <h3 className="text-sm font-medium text-muted-foreground mb-2">Feeds</h3>
        <FeedList
          feeds={feeds}
          onDelete={handleDelete}
          onRefresh={onRefreshFeed}
          onClick={(feedId) => {
            onFeedClick(feedId)
            // Close mobile drawer after selection
            if (window.innerWidth < 768) {
              // Triggered by parent
            }
          }}
          selectedFeedId={selectedFeedId}
          refreshingFeedIds={refreshingFeedIds}
          onAddFeed={onAddFeed}
        />
      </div>
    </>
  )
}

/**
 * Main Sidebar component - responsive with desktop/mobile views
 */
export function Sidebar({
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
}: SidebarProps) {
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
        />
      </aside>

      {/* Mobile Sidebar - rendered in drawer by parent */}
    </>
  )
}

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
