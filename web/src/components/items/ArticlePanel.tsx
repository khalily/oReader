import { Star, StarOff, Eye, EyeOff, ExternalLink, Loader2, Calendar, User, Rss, ArrowLeft, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useState, useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useItems } from '@/hooks/useItems'
import type { Article } from '@/types/feed'

interface ArticlePanelProps {
  itemId: string | null
  onClose?: () => void
  showBackButton?: boolean
}

export function ArticlePanel({ itemId, onClose, showBackButton = false }: ArticlePanelProps) {
  const queryClient = useQueryClient()
  const { useGetItem, useToggleStar, useToggleRead } = useItems()

  const { data, isLoading, error } = useGetItem(itemId)
  const toggleStar = useToggleStar()
  const toggleRead = useToggleRead()

  // Local optimistic state
  const [optimisticState, setOptimisticState] = useState<{
    is_starred: boolean
    is_read: boolean
  } | null>(null)

  // Summary collapse state
  const [summaryExpanded, setSummaryExpanded] = useState(true)

  // Track if we've already triggered auto-mark-as-read for this item
  // Using ref to avoid triggering re-renders and useEffect loops
  const hasAutoMarkedRead = useRef(false)

  const item = data?.item
    ? optimisticState
      ? { ...data.item, user_state: { ...data.item.user_state, ...optimisticState } }
      : data.item
    : null

  // Reset optimistic state when item changes
  useEffect(() => {
    setOptimisticState(null)
  }, [itemId])

  // Reset auto-mark flag when item changes
  useEffect(() => {
    hasAutoMarkedRead.current = false
  }, [itemId])

  // Auto-mark as read when viewing (only once per item)
  useEffect(() => {
    if (
      data?.item &&
      data.item.user_state &&
      !data.item.user_state.is_read &&
      !hasAutoMarkedRead.current
    ) {
      // Set flag immediately to prevent duplicate requests
      hasAutoMarkedRead.current = true

      toggleRead.mutate(
        { itemId: data.item.id, read: true },
        {
          onSuccess: (response) => {
            setOptimisticState({ is_starred: response.item.user_state?.is_starred ?? false, is_read: true })
          },
          onError: () => {
            // Reset flag on error to allow retry
            hasAutoMarkedRead.current = false
          },
        }
      )
    }
  }, [data?.item])

  const handleToggleStar = () => {
    if (data?.item) {
      const currentStarred = optimisticState?.is_starred ?? data.item.user_state?.is_starred ?? false
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
            queryClient.invalidateQueries({ queryKey: ['items'] })
            queryClient.invalidateQueries({ queryKey: ['stats'] })
          },
          onError: () => {
            setOptimisticState(null)
          },
        }
      )
    }
  }

  const handleToggleRead = () => {
    if (data?.item) {
      const currentRead = optimisticState?.is_read ?? data.item.user_state?.is_read ?? false
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
            queryClient.invalidateQueries({ queryKey: ['items'] })
            queryClient.invalidateQueries({ queryKey: ['stats'] })
          },
          onError: () => {
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

  const createSafeHTML = (html: string | null) => {
    if (!html) return { __html: '' }
    const sanitized = html
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '')
      .replace(/on\w+="[^"]*"/gi, '')
      .replace(/javascript:/gi, '')
    return { __html: sanitized }
  }

  // Empty state - no item selected
  if (!itemId) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-8">
        <Rss className="h-16 w-16 text-muted-foreground/50 mb-4" />
        <h2 className="text-xl font-semibold text-muted-foreground mb-2">
          Select an article to read
        </h2>
        <p className="text-sm text-muted-foreground/70">
          Choose an article from the list to view its content
        </p>
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Error state
  if (error || !item) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-8">
        <h2 className="text-xl font-semibold mb-2">Unable to load article</h2>
        <p className="text-sm text-muted-foreground mb-4">
          The article could not be loaded. It may have been deleted or you don't have access.
        </p>
        <Button variant="outline" onClick={() => queryClient.invalidateQueries({ queryKey: ['items', itemId] })}>
          Try Again
        </Button>
      </div>
    )
  }

  const isRead = item.user_state?.is_read ?? false
  const isStarred = item.user_state?.is_starred ?? false
  const hasSummary = item.description && item.description !== item.content

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header with actions */}
      <div className="flex items-center justify-between p-4 border-b bg-background sticky top-0 z-10">
        <div className="flex items-center gap-2">
          {showBackButton && (
            <Button variant="ghost" size="sm" onClick={onClose} className="lg:hidden">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleToggleRead}
            disabled={toggleRead.isPending}
          >
            {isRead ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
            <span className="ml-2 hidden sm:inline">{isRead ? 'Read' : 'Unread'}</span>
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={handleToggleStar}
            disabled={toggleStar.isPending}
            className={cn(isStarred && 'text-yellow-600')}
          >
            {isStarred ? <Star className="h-4 w-4 fill-current" /> : <StarOff className="h-4 w-4" />}
            <span className="ml-2 hidden sm:inline">{isStarred ? 'Starred' : 'Star'}</span>
          </Button>

          {item.link && (
            <Button variant="ghost" size="sm" asChild>
              <a
                href={item.link}
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Open article in new tab"
              >
                <ExternalLink className="h-4 w-4" />
                <span className="ml-2 hidden sm:inline">Open</span>
              </a>
            </Button>
          )}
        </div>
      </div>

      {/* Article content - scrollable */}
      <div className="flex-1 overflow-y-auto p-4">
        {/* Article header */}
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <Badge variant="secondary" className="flex items-center gap-1">
              <Rss className="h-3 w-3" />
              {item.feed?.title || 'Unknown Feed'}
            </Badge>
            {isRead && (
              <Badge variant="outline" className="flex items-center gap-1">
                <Eye className="h-3 w-3" />
                Read
              </Badge>
            )}
          </div>

          <h1 className="text-2xl font-bold mb-3">{item.title}</h1>

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
        </div>

        {/* Summary section - collapsible */}
        {hasSummary && (
          <Card className="mb-6">
            <CardContent className="p-4">
              <button
                onClick={() => setSummaryExpanded(!summaryExpanded)}
                className="flex items-center justify-between w-full text-left"
              >
                <span className="text-sm font-medium text-muted-foreground">Summary</span>
                {summaryExpanded ? (
                  <ChevronUp className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                )}
              </button>
              {summaryExpanded && (
                <p className="text-muted-foreground mt-2 text-sm border-l-2 border-muted pl-3">
                  {item.description}
                </p>
              )}
            </CardContent>
          </Card>
        )}

        {/* Article content */}
        {item.content ? (
          <Card>
            <CardContent className="p-6">
              <article
                className="prose prose-slate max-w-none dark:prose-invert prose-sm"
                dangerouslySetInnerHTML={createSafeHTML(item.content)}
              />
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="p-6">
              <p className="text-muted-foreground italic text-center">
                No content available. Click "Open" to read the full article.
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
