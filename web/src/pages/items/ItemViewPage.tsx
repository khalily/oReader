import { useParams, useNavigate } from 'react-router-dom'
import { useItems } from '@/hooks/useItems'
import { Star, StarOff, Eye, EyeOff, ArrowLeft, ExternalLink, Loader2, Calendar, User, Rss } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'

export default function ItemViewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { useGetItem, useToggleStar, useToggleRead } = useItems()

  const { data, isLoading, error } = useGetItem(id ?? null)
  const toggleStar = useToggleStar()
  const toggleRead = useToggleRead()

  // Local optimistic state
  const [optimisticState, setOptimisticState] = useState<{
    is_starred: boolean
    is_read: boolean
  } | null>(null)

  const item = data?.item
    ? optimisticState
      ? { ...data.item, user_state: { ...data.item.user_state, ...optimisticState } }
      : data.item
    : null

  // Auto-mark as read when viewing (only once)
  useEffect(() => {
    if (data?.item && data.item.user_state && !data.item.user_state.is_read && !optimisticState?.is_read) {
      toggleRead.mutate(
        { itemId: data.item.id, read: true },
        {
          onSuccess: (response) => {
            setOptimisticState({ is_starred: response.item.user_state?.is_starred ?? false, is_read: true })
          },
        }
      )
    }
  }, [data?.item, optimisticState, toggleRead])

  const handleToggleStar = () => {
    if (data?.item) {
      const currentStarred = optimisticState?.is_starred ?? data.item.user_state?.is_starred ?? false
      // Optimistic update
      setOptimisticState({
        is_starred: !currentStarred,
        is_read: optimisticState?.is_read ?? data.item.user_state?.is_read ?? false,
      })

      toggleStar.mutate(
        { itemId: data.item.id, starred: !currentStarred },
        {
          onSuccess: (response) => {
            setOptimisticState({
              is_starred: response.item.user_state?.is_starred ?? false,
              is_read: response.item.user_state?.is_read ?? false,
            })
            // Invalidate queries to refresh other components
            queryClient.invalidateQueries({ queryKey: ['items'] })
          },
          onError: () => {
            // Revert optimistic update on error
            setOptimisticState(null)
          },
        }
      )
    }
  }

  const handleToggleRead = () => {
    if (data?.item) {
      const currentRead = optimisticState?.is_read ?? data.item.user_state?.is_read ?? false
      // Optimistic update
      setOptimisticState({
        is_starred: optimisticState?.is_starred ?? data.item.user_state?.is_starred ?? false,
        is_read: !currentRead,
      })

      toggleRead.mutate(
        { itemId: data.item.id, read: !currentRead },
        {
          onSuccess: (response) => {
            setOptimisticState({
              is_starred: response.item.user_state?.is_starred ?? false,
              is_read: response.item.user_state?.is_read ?? false,
            })
            // Invalidate queries to refresh other components
            queryClient.invalidateQueries({ queryKey: ['items'] })
          },
          onError: () => {
            // Revert optimistic update on error
            setOptimisticState(null)
          },
        }
      )
    }
  }

  const formatDate = (dateString: string | null) => {
    if (!dateString) return ''
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !item) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <h1 className="text-2xl font-semibold mb-2">Article Not Found</h1>
        <p className="text-muted-foreground mb-4">
          The article you're looking for doesn't exist or you don't have access to it.
        </p>
        <Button onClick={() => navigate('/items')}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Articles
        </Button>
      </div>
    )
  }

  const isRead = item.user_state?.is_read ?? false
  const isStarred = item.user_state?.is_starred ?? false

  return (
    <div className="max-w-4xl mx-auto py-6 px-4">
      {/* Header with actions */}
      <div className="flex items-center justify-between mb-6">
        <Button variant="ghost" onClick={() => navigate('/items')}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back
        </Button>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleToggleRead}
            disabled={toggleRead.isPending}
          >
            {isRead ? <Eye className="h-4 w-4 mr-2" /> : <EyeOff className="h-4 w-4 mr-2" />}
            {isRead ? 'Read' : 'Unread'}
          </Button>

          <Button
            variant="outline"
            size="sm"
            onClick={handleToggleStar}
            disabled={toggleStar.isPending}
            className={cn(isStarred && 'text-yellow-600 border-yellow-600 hover:bg-yellow-50')}
          >
            {isStarred ? <Star className="h-4 w-4 mr-2 fill-current" /> : <StarOff className="h-4 w-4 mr-2" />}
            {isStarred ? 'Starred' : 'Star'}
          </Button>

          {item.link && (
            <Button variant="outline" size="sm" asChild>
              <a
                href={item.link}
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Open article in new tab"
              >
                <ExternalLink className="h-4 w-4 mr-2" />
                Open Original
              </a>
            </Button>
          )}
        </div>
      </div>

      {/* Article header */}
      <Card className="mb-6">
        <CardContent className="p-6">
          <div className="flex items-center gap-2 mb-3">
            <Badge variant="secondary" className="flex items-center gap-1">
              <Rss className="h-3 w-3" />
              {item.feed.title}
            </Badge>
            {isRead && (
              <Badge variant="outline" className="flex items-center gap-1">
                <Eye className="h-3 w-3" />
                Read
              </Badge>
            )}
          </div>

          <h1 className="text-3xl font-bold mb-4">{item.title}</h1>

          <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            {item.creator && (
              <div className="flex items-center gap-1">
                <User className="h-4 w-4" />
                <span>{item.creator}</span>
              </div>
            )}
            {item.pub_date && (
              <div className="flex items-center gap-1">
                <Calendar className="h-4 w-4" />
                <span>{formatDate(item.pub_date)}</span>
              </div>
            )}
          </div>

          {item.description && item.description !== item.content && (
            <p className="text-muted-foreground mt-4 border-l-4 border-muted pl-4">
              {item.description}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Article content */}
      {item.content ? (
        <Card>
          <CardContent className="p-6">
            <MarkdownRenderer content={item.content} />
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-6">
            <p className="text-muted-foreground italic">
              No content available. Click "Open Original" to read the full article.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
