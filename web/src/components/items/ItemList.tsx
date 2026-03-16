import { Star, StarOff, Eye, EyeOff, ExternalLink, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { Article } from '@/types/feed'

interface ItemListProps {
  articles: Article[]
  onItemClick: (itemId: string) => void
  onToggleStar: (itemId: string, starred: boolean) => void
  onToggleRead: (itemId: string, read: boolean) => void
  isLoading?: boolean
}

function formatDate(dateString: string | null): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  const now = new Date()
  const diffInMs = now.getTime() - date.getTime()
  const diffInHours = diffInMs / (1000 * 60 * 60)
  const diffInDays = diffInMs / (1000 * 60 * 60 * 24)

  // For articles less than 24 hours old, show relative time
  if (diffInHours < 1) {
    const minutes = Math.floor(diffInMs / (1000 * 60))
    return `${minutes}m ago`
  }
  if (diffInHours < 24) {
    const hours = Math.floor(diffInHours)
    return `${hours}h ago`
  }

  // For older articles, show the date
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  const month = months[date.getMonth()]
  const day = date.getDate()
  const year = date.getFullYear()

  if (year !== now.getFullYear()) {
    return `${month} ${day}, ${year}`
  }
  return `${month} ${day}`
}

export function ItemList({
  articles,
  onItemClick,
  onToggleStar,
  onToggleRead,
  isLoading = false,
}: ItemListProps) {
  const handleStarClick = (e: React.MouseEvent, itemId: string, currentStarred: boolean) => {
    e.stopPropagation()
    onToggleStar(itemId, !currentStarred)
  }

  const handleReadClick = (e: React.MouseEvent, itemId: string, currentRead: boolean) => {
    e.stopPropagation()
    onToggleRead(itemId, !currentRead)
  }

  const handleCardClick = (itemId: string) => {
    onItemClick(itemId)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (articles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <p className="text-lg text-muted-foreground">No articles found</p>
        <p className="text-sm text-muted-foreground">
          Try adjusting your filters or subscribe to some feeds
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {articles.map((article) => {
        const isRead = article.user_state?.is_read ?? false
        const isStarred = article.user_state?.is_starred ?? false

        return (
          <Card
            key={article.id}
            className={cn(
              "cursor-pointer transition-colors hover:bg-accent/50",
              isRead && "bg-muted/30"
            )}
            onClick={() => handleCardClick(article.id)}
          >
            <CardContent className="p-4">
              <div className="flex items-start gap-3">
                {/* Action buttons */}
                <div className="flex flex-col gap-1 flex-shrink-0">
                  <Button
                    variant="ghost"
                    size="icon"
                    className={cn(
                      "h-8 w-8",
                      isStarred && "text-yellow-500 hover:text-yellow-600"
                    )}
                    onClick={(e) => handleStarClick(e, article.id, isStarred)}
                  >
                    {isStarred ? <Star className="h-4 w-4 fill-current" /> : <StarOff className="h-4 w-4" />}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={(e) => handleReadClick(e, article.id, isRead)}
                  >
                    {isRead ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
                  </Button>
                </div>

                {/* Article content */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <h3
                        className={cn(
                          "font-medium line-clamp-2",
                          isRead ? "text-muted-foreground" : "text-foreground"
                        )}
                      >
                        {article.title}
                      </h3>
                      <div className="flex items-center gap-2 mt-1 flex-wrap">
                        <span className="text-xs text-muted-foreground">
                          {article.feed.title}
                        </span>
                        <span className="text-xs text-muted-foreground">•</span>
                        <span className="text-xs text-muted-foreground">
                          {formatDate(article.pub_date)}
                        </span>
                        {article.creator && (
                          <>
                            <span className="text-xs text-muted-foreground">•</span>
                            <span className="text-xs text-muted-foreground">
                              {article.creator}
                            </span>
                          </>
                        )}
                      </div>
                      {article.description && (
                        <p className="text-sm text-muted-foreground line-clamp-2 mt-2">
                          {article.description.replace(/<[^>]*>/g, '')}
                        </p>
                      )}
                    </div>

                    {article.link && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 flex-shrink-0"
                        asChild
                        onClick={(e) => e.stopPropagation()}
                      >
                        <a
                          href={article.link}
                          target="_blank"
                          rel="noopener noreferrer"
                          aria-label="Open article in new tab"
                        >
                          <ExternalLink className="h-4 w-4" />
                        </a>
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
