import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { FeedCard } from './FeedCard'
import type { UserFeed } from '@/types/feed'

interface FeedListProps {
  feeds: UserFeed[]
  onDelete: (feedId: string) => void
  onRefresh: (feedId: string) => void
  onClick: (feedId: string) => void
  selectedFeedId: string | null
  refreshingFeedIds: Set<string>
  onAddFeed?: () => void
}

export function FeedList({
  feeds,
  onDelete,
  onRefresh,
  onClick,
  selectedFeedId,
  refreshingFeedIds,
  onAddFeed,
}: FeedListProps) {
  if (feeds.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
        <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
          <Plus className="w-8 h-8 text-muted-foreground" />
        </div>
        <h3 className="text-lg font-medium mb-2">No feeds yet</h3>
        <p className="text-sm text-muted-foreground mb-6 max-w-sm">
          Subscribe to RSS feeds to get started. Click the button below to add your first feed.
        </p>
        {onAddFeed && (
          <Button onClick={onAddFeed}>
            <Plus className="w-4 h-4 mr-2" />
            Add Feed
          </Button>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {onAddFeed && (
        <div className="mb-4">
          <Button onClick={onAddFeed} className="w-full">
            <Plus className="w-4 h-4 mr-2" />
            Add Feed
          </Button>
        </div>
      )}
      {feeds.map((feed) => (
        <FeedCard
          key={feed.id}
          feed={feed}
          onDelete={onDelete}
          onRefresh={onRefresh}
          onClick={onClick}
          isRefreshing={refreshingFeedIds.has(feed.id)}
          isSelected={selectedFeedId === feed.id}
        />
      ))}
    </div>
  )
}
