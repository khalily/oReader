import { Trash2, RefreshCw, Rss } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { UserFeed } from '@/types/feed'

interface FeedCardProps {
  feed: UserFeed
  onDelete: (feedId: string) => void
  onRefresh: (feedId: string) => void
  onClick: (feedId: string) => void
  isRefreshing?: boolean
  isSelected?: boolean
}

export function FeedCard({
  feed,
  onDelete,
  onRefresh,
  onClick,
  isRefreshing = false,
  isSelected = false,
}: FeedCardProps) {
  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    onDelete(feed.id)
  }

  const handleRefresh = (e: React.MouseEvent) => {
    e.stopPropagation()
    onRefresh(feed.id)
  }

  const handleClick = () => {
    onClick(feed.id)
  }

  return (
    <Card
      className={cn(
        "cursor-pointer transition-colors hover:bg-accent/50",
        isSelected && "bg-accent"
      )}
      onClick={handleClick}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3 min-w-0 flex-1">
            {feed.image_url ? (
              <img
                src={feed.image_url}
                alt={feed.title}
                className="w-10 h-10 rounded object-cover flex-shrink-0"
              />
            ) : (
              <div className="w-10 h-10 rounded bg-muted flex items-center justify-center flex-shrink-0">
                <Rss className="w-5 h-5 text-muted-foreground" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-medium truncate">{feed.title}</h3>
                {feed.unread_count > 0 && (
                  <Badge variant="default" className="text-xs">
                    {feed.unread_count}
                  </Badge>
                )}
              </div>
              {feed.description && (
                <p className="text-sm text-muted-foreground truncate">
                  {feed.description}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={handleRefresh}
              disabled={isRefreshing}
            >
              <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive"
              onClick={handleDelete}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
