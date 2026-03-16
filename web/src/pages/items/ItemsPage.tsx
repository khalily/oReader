import { useState, useCallback, useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useItems } from '@/hooks/useItems'
import { useFeeds } from '@/hooks/useFeeds'
import { ItemList } from '@/components/items/ItemList'
import { Sidebar, type FilterType } from '@/components/feed/Sidebar'
import { AddFeedDialog } from '@/components/feed/AddFeedDialog'
import { useItemsStore } from '@/stores/itemsStore'
import type { Article, ListItemsOptions } from '@/types/feed'

interface ItemsPageProps {
  filterType?: FilterType
  feedId?: string
}

export function ItemsPage({ filterType = 'all', feedId }: ItemsPageProps) {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [isAddFeedOpen, setIsAddFeedOpen] = useState(false)
  const [refreshingFeedIds, setRefreshingFeedIds] = useState<Set<string>>(new Set())
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const observerTarget = useRef<HTMLDivElement>(null)

  const { useListFeeds, useRefreshFeed, useDeleteFeed, useMarkAllRead } = useFeeds()
  const { useListItems, useToggleStar, useToggleRead } = useItems()

  // Query items based on filter and feed
  const listOptions: ListItemsOptions = {}
  if (feedId) listOptions.feed_id = feedId
  if (filterType === 'unread') listOptions.read = false
  if (filterType === 'starred') listOptions.starred = true
  listOptions.limit = 20

  const { data: itemsData, isLoading: itemsLoading, refetch } = useListItems(listOptions)
  const { data: feedsData, refetch: refetchFeeds } = useListFeeds()

  const toggleStar = useToggleStar()
  const toggleRead = useToggleRead()
  const refreshFeed = useRefreshFeed()
  const deleteFeed = useDeleteFeed()
  const markAllRead = useMarkAllRead()

  const items = itemsData?.items ?? []
  const hasMore = itemsData?.has_more ?? false
  const nextCursor = itemsData?.next_cursor
  const feeds = feedsData?.feeds ?? []

  // Calculate total unread count
  const totalUnread = feeds.reduce((sum, feed) => sum + feed.unread_count, 0)

  // Store items in global store for optimistic updates
  const setItems = useItemsStore((state) => state.setItems)
  const updateItemState = useItemsStore((state) => state.updateItemState)

  useEffect(() => {
    if (items.length > 0) {
      setItems(items)
    }
  }, [items, setItems])

  // Handle item click
  const handleItemClick = useCallback(
    (itemId: string) => {
      navigate(`/items/${itemId}`)
    },
    [navigate]
  )

  // Handle star toggle
  const handleToggleStar = useCallback(
    (itemId: string, starred: boolean) => {
      toggleStar.mutate(
        { itemId, starred },
        {
          onSuccess: () => {
            updateItemState(itemId, { is_starred: starred })
            refetch()
          },
        }
      )
    },
    [toggleStar, updateItemState, refetch]
  )

  // Handle read toggle
  const handleToggleRead = useCallback(
    (itemId: string, read: boolean) => {
      toggleRead.mutate(
        { itemId, read },
        {
          onSuccess: () => {
            updateItemState(itemId, { is_read: read })
            refetch()
            refetchFeeds()
          },
        }
      )
    },
    [toggleRead, updateItemState, refetch, refetchFeeds]
  )

  // Handle feed refresh
  const handleRefreshFeed = useCallback(
    (feedId: string) => {
      setRefreshingFeedIds((prev) => new Set(prev).add(feedId))
      refreshFeed.mutate(feedId, {
        onSettled: () => {
          setRefreshingFeedIds((prev) => {
            const next = new Set(prev)
            next.delete(feedId)
            return next
          })
          refetch()
          refetchFeeds()
        },
      })
    },
    [refreshFeed, refetch, refetchFeeds]
  )

  // Handle feed delete
  const handleDeleteFeed = useCallback(
    (feedId: string) => {
      deleteFeed.mutate(feedId, {
        onSuccess: () => {
          refetch()
          refetchFeeds()
        },
      })
    },
    [deleteFeed, refetch, refetchFeeds]
  )

  // Handle mark all read
  const handleMarkAllRead = useCallback(
    (feedId: string) => {
      markAllRead.mutate(feedId, {
        onSuccess: () => {
          refetch()
          refetchFeeds()
        },
      })
    },
    [markAllRead, refetch, refetchFeeds]
  )

  // Handle feed click
  const handleFeedClick = useCallback(
    (id: string) => {
      setSearchParams({ feed: id })
    },
    [setSearchParams]
  )

  // Handle filter change
  const handleFilterChange = useCallback(
    (newFilter: FilterType) => {
      if (newFilter === 'all') {
        setSearchParams({})
      } else {
        setSearchParams({ filter: newFilter })
      }
    },
    [setSearchParams]
  )

  // Infinite scroll using Intersection Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoadingMore) {
          setIsLoadingMore(true)
          // In a real implementation, we would fetch the next page using nextCursor
          // For now, we'll just refetch with the updated cursor
          if (nextCursor) {
            // This would be handled by a more sophisticated pagination hook
            setIsLoadingMore(false)
          }
        }
      },
      { threshold: 0.1 }
    )

    const currentTarget = observerTarget.current
    if (currentTarget) {
      observer.observe(currentTarget)
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget)
      }
    }
  }, [hasMore, isLoadingMore, nextCursor])

  // Get selected feed ID from URL or state
  const selectedFeedId = feedId ?? null

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        feeds={feeds}
        selectedFeedId={selectedFeedId}
        onFeedClick={handleFeedClick}
        onAddFeed={() => setIsAddFeedOpen(true)}
        onDeleteFeed={handleDeleteFeed}
        onRefreshFeed={handleRefreshFeed}
        refreshingFeedIds={refreshingFeedIds}
        filterType={filterType}
        onFilterChange={handleFilterChange}
        totalUnread={totalUnread}
      />

      <main className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto py-6 px-4">
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-2xl font-bold">
              {filterType === 'starred' && 'Starred Articles'}
              {filterType === 'unread' && 'Unread Articles'}
              {filterType === 'all' && !feedId && 'All Articles'}
              {feedId && feeds.find((f) => f.id === feedId)?.title}
            </h1>

            {feedId && (
              <button
                onClick={() => handleMarkAllRead(feedId)}
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                Mark all as read
              </button>
            )}
          </div>

          <ItemList
            articles={items}
            onItemClick={handleItemClick}
            onToggleStar={handleToggleStar}
            onToggleRead={handleToggleRead}
            isLoading={itemsLoading}
          />

          {hasMore && (
            <div ref={observerTarget} className="py-8 text-center">
              {isLoadingMore && (
                <div className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              )}
            </div>
          )}
        </div>
      </main>

      <AddFeedDialog
        open={isAddFeedOpen}
        onOpenChange={setIsAddFeedOpen}
        onSuccess={() => {
          setIsAddFeedOpen(false)
          refetchFeeds()
        }}
      />
    </div>
  )
}

export default ItemsPage
