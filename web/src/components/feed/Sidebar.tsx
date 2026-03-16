import { Home, Star, Rss } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { FeedList } from './FeedList'
import { cn } from '@/lib/utils'
import type { UserFeed } from '@/types/feed'

export type FilterType = 'all' | 'unread' | 'starred'

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
  isOpen?: boolean
  onClose?: () => void
}

const filters = [
  { type: 'all' as FilterType, label: 'All Items', icon: Home },
  { type: 'unread' as FilterType, label: 'Unread', icon: Rss },
  { type: 'starred' as FilterType, label: 'Starred', icon: Star },
]

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
  isOpen = true,
  onClose,
}: SidebarProps) {
  const handleDelete = (feedId: string) => {
    if (confirm('Are you sure you want to delete this feed?')) {
      onDeleteFeed(feedId)
    }
  }

  return (
    <aside
      className={cn(
        "w-64 border-r bg-background flex-shrink-0 flex flex-col h-full",
        !isOpen && "hidden"
      )}
    >
      <div className="p-4 border-b">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">oReader</h2>
          {onClose && (
            <Button variant="ghost" size="icon" onClick={onClose}>
              ✕
            </Button>
          )}
        </div>

        <nav className="space-y-1">
          {filters.map((filter) => {
            const Icon = filter.icon
            const isActive = filterType === filter.type
            const unreadCount =
              filter.type === 'unread'
                ? totalUnread
                : feeds.reduce((sum, f) => sum + f.unread_count, 0)

            return (
              <Button
                key={filter.type}
                variant={isActive ? 'secondary' : 'ghost'}
                className={cn('w-full justify-start', isActive && 'bg-accent')}
                onClick={() => onFilterChange(filter.type)}
              >
                <Icon className="w-4 h-4 mr-2" />
                {filter.label}
                {filter.type === 'unread' && unreadCount > 0 && (
                  <Badge variant="default" className="ml-auto text-xs">
                    {unreadCount}
                  </Badge>
                )}
              </Button>
            )
          })}
        </nav>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        <h3 className="text-sm font-medium text-muted-foreground mb-2">Feeds</h3>
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
    </aside>
  )
}
