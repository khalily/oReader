import { useState, useCallback, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useItems } from '@/hooks/useItems'
import { useFeeds } from '@/hooks/useFeeds'
import { usePapers } from '@/hooks/usePapers'
import {
  useListCategories,
  useCreateCategory,
  useRenameCategory,
  useDeleteCategory,
  useMoveFeedToCategory,
  useMovePaperToCategory,
} from '@/hooks/useCategories'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useToast } from '@/components/ui/toast'
import { MobileDrawer } from '@/components/ui/mobile-drawer'
import { KeyboardShortcutsModal } from '@/components/ui/keyboard-shortcuts-modal'
import { ThemeToggle } from '@/components/ui/theme-toggle'
import { ItemList } from '@/components/items/ItemList'
import { ArticlePanel } from '@/components/items/ArticlePanel'
import { NewSidebar, NewSidebarContent } from '@/components/feed/Sidebar'
import { AddDialog } from '@/components/sidebar/AddDialog'
import { PaperMeta } from '@/components/papers/PaperMeta'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { MobileMenuButton } from '@/components/feed/Sidebar'
import { useSidebarStore } from '@/stores/sidebarStore'
import { useItemsStore } from '@/stores/itemsStore'
import { Loader2, AlertCircle, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ListItemsOptions } from '@/types/feed'

export function ItemsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedItemId = searchParams.get('id')
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false)
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isShortcutsModalOpen, setIsShortcutsModalOpen] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const observerTarget = useRef<HTMLDivElement>(null)
  const toast = useToast()

  const { useListFeeds } = useFeeds()
  const { useListItemsInfinite, useToggleStar, useToggleRead, useMarkAllRead } = useItems()
  const { useListPapers, useGetPaper, useGetPaperStatus, useDownloadPaper } = usePapers()
  const {
    data: feedCategoriesData,
  } = useListCategories('feed')
  const {
    data: paperCategoriesData,
  } = useListCategories('paper')

  // Category data
  const feedCategories = feedCategoriesData?.categories ?? []
  const paperCategories = paperCategoriesData?.categories ?? []

  // Sidebar store
  const { selectedFeedId, selectedPaperId, selectionType, selectFeed, selectPaper } = useSidebarStore()

  // Category mutations
  const createCategory = useCreateCategory()
  const renameCategory = useRenameCategory()
  const deleteCategory = useDeleteCategory()
  const moveFeedToCategory = useMoveFeedToCategory()
  const movePaperToCategory = useMovePaperToCategory()

  // Query items based on sidebar selection
  const listOptions: ListItemsOptions = {}
  if (selectionType === 'feed' && selectedFeedId) listOptions.feed_id = selectedFeedId
  listOptions.limit = 20

  const {
    data: infiniteData,
    isLoading: itemsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useListItemsInfinite(listOptions)
  const { data: feedsData, refetch: refetchFeeds } = useListFeeds()

  // Papers
  const { data: papersData } = useListPapers()
  const { data: selectedPaperData } = useGetPaper(
    selectionType === 'paper' ? selectedPaperId : null
  )
  const { data: paperStatusData } = useGetPaperStatus(
    selectionType === 'paper' ? selectedPaperId : null
  )
  const downloadPaper = useDownloadPaper()

  const toggleStar = useToggleStar()
  const toggleRead = useToggleRead()
  const markAllRead = useMarkAllRead()

  const items = infiniteData?.pages.flatMap(page => page.items) ?? []
  const hasMore = hasNextPage ?? false
  const feeds = feedsData?.feeds ?? []
  const papers = papersData?.papers ?? []

  // Store items in global store for optimistic updates
  const setItems = useItemsStore((state) => state.setItems)
  const updateItemState = useItemsStore((state) => state.updateItemState)

  useEffect(() => {
    if (items.length > 0) {
      setItems(items)
    }
  }, [items, setItems])

  // Handle item click - update URL to show article in third column
  const handleItemClick = useCallback(
    (itemId: string) => {
      setSearchParams((prev) => {
        prev.set('id', itemId)
        return prev
      })
    },
    [setSearchParams]
  )

  // Handle closing article panel on mobile
  const handleCloseArticle = useCallback(() => {
    setSearchParams((prev) => {
      prev.delete('id')
      return prev
    })
  }, [setSearchParams])

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

  // Handle mark all read
  const handleMarkAllRead = useCallback(
    (feedId: string) => {
      markAllRead.mutate(feedId, {
        onSuccess: () => {
          refetch()
          refetchFeeds()
          toast.showSuccess('All articles marked as read')
        },
      })
    },
    [markAllRead, refetch, refetchFeeds, toast]
  )

  // Category handlers
  const handleRenameCategory = useCallback(
    (categoryId: string, name: string) => {
      renameCategory.mutate(
        { categoryId, data: { name } },
        {
          onSuccess: () => toast.showSuccess('Category renamed'),
          onError: () => toast.showError('Failed to rename category'),
        }
      )
    },
    [renameCategory, toast]
  )

  const handleDeleteCategory = useCallback(
    (categoryId: string) => {
      deleteCategory.mutate(categoryId, {
        onSuccess: () => toast.showSuccess('Category deleted'),
        onError: () => toast.showError('Failed to delete category'),
      })
    },
    [deleteCategory, toast]
  )

  const handleMoveFeedToCategory = useCallback(
    (feedId: string, categoryId: string) => {
      moveFeedToCategory.mutate(
        { feedId, data: { category_id: categoryId } },
        {
          onSuccess: () => {
            refetchFeeds()
            toast.showSuccess('Feed moved to category')
          },
          onError: () => toast.showError('Failed to move feed'),
        }
      )
    },
    [moveFeedToCategory, refetchFeeds, toast]
  )

  const handleRemovePaperFromCategory = useCallback(
    (paperId: string) => {
      movePaperToCategory.mutate(
        { paperId, data: { category_id: '' } },
        {
          onSuccess: () => {
            toast.showSuccess('Paper removed from category')
          },
          onError: () => toast.showError('Failed to remove paper from category'),
        }
      )
    },
    [movePaperToCategory, toast]
  )

  const handleMoveFeedToNewCategory = useCallback(
    (feedId: string, categoryName: string) => {
      createCategory.mutate(
        { name: categoryName, type: 'feed' },
        {
          onSuccess: (newCategory) => {
            moveFeedToCategory.mutate(
              { feedId, data: { category_id: newCategory.category.id } },
              {
                onSuccess: () => {
                  refetchFeeds()
                  toast.showSuccess('Feed moved to new category')
                },
              }
            )
          },
          onError: () => toast.showError('Failed to create category'),
        }
      )
    },
    [createCategory, moveFeedToCategory, refetchFeeds, toast]
  )

  // Paper download handler
  const handleDownloadPaper = useCallback(() => {
    if (!selectedPaperId) return
    downloadPaper.mutate(selectedPaperId, {
      onSuccess: (blob) => {
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = selectedPaperData?.original_filename || 'paper.pdf'
        document.body.appendChild(a)
        a.click()
        a.remove()
        window.URL.revokeObjectURL(url)
      },
      onError: () => {
        toast.showError('Failed to download paper')
      },
    })
  }, [selectedPaperId, selectedPaperData, downloadPaper, toast])

  // Infinite scroll using Intersection Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
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
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  // Keyboard shortcuts
  useKeyboardShortcuts({
    enabled: true,
    shortcuts: [
      {
        key: 'j',
        description: 'Next article',
        action: () => {
          if (items.length > 0 && selectedIndex < items.length - 1) {
            const nextIndex = selectedIndex + 1
            setSelectedIndex(nextIndex)
            const itemElement = document.querySelector(`[data-item-id="${items[nextIndex].id}"]`)
            itemElement?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
          }
        },
      },
      {
        key: 'k',
        description: 'Previous article',
        action: () => {
          if (selectedIndex > 0) {
            const prevIndex = selectedIndex - 1
            setSelectedIndex(prevIndex)
            const itemElement = document.querySelector(`[data-item-id="${items[prevIndex].id}"]`)
            itemElement?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
          }
        },
      },
      {
        key: 'Enter',
        description: 'Open article',
        action: () => {
          if (items[selectedIndex]) {
            handleItemClick(items[selectedIndex].id)
          }
        },
      },
      {
        key: 's',
        description: 'Star/unstar article',
        action: () => {
          if (items[selectedIndex]) {
            const item = items[selectedIndex]
            const isStarred = item.user_state?.is_starred ?? false
            handleToggleStar(item.id, !isStarred)
          }
        },
      },
      {
        key: 'r',
        description: 'Mark as read/unread',
        action: () => {
          if (items[selectedIndex]) {
            const item = items[selectedIndex]
            const isRead = item.user_state?.is_read ?? false
            handleToggleRead(item.id, !isRead)
          }
        },
      },
      {
        key: 'n',
        description: 'Mark all as read',
        action: () => {
          if (selectedFeedId) {
            handleMarkAllRead(selectedFeedId)
          }
        },
      },
      {
        key: '?',
        description: 'Show keyboard shortcuts',
        action: () => setIsShortcutsModalOpen(true),
      },
      {
        key: 'Escape',
        description: 'Close modals and article',
        action: () => {
          if (isShortcutsModalOpen) setIsShortcutsModalOpen(false)
          if (isMobileMenuOpen) setIsMobileMenuOpen(false)
          if (isAddDialogOpen) setIsAddDialogOpen(false)
          if (selectedItemId) handleCloseArticle()
        },
      },
    ],
  })

  // Determine middle column title
  const middleColumnTitle = selectionType === 'feed' && selectedFeedId
    ? feeds.find((f) => f.id === selectedFeedId)?.title ?? 'Feed'
    : 'All Articles'

  // Paper content for right column
  const paper = selectedPaperData ?? null
  const paperStatus = paperStatusData ?? null

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Desktop Sidebar */}
      <NewSidebar
        feedCategories={feedCategories}
        paperCategories={paperCategories}
        feeds={feeds}
        papers={papers}
        onAddClick={() => setIsAddDialogOpen(true)}
        onFeedClick={(id) => { selectFeed(id); setIsMobileMenuOpen(false) }}
        onPaperClick={(id) => { selectPaper(id); setSearchParams(prev => { prev.delete('id'); prev.set('paper', id); return prev }); setIsMobileMenuOpen(false) }}
        onRenameCategory={handleRenameCategory}
        onDeleteCategory={handleDeleteCategory}
        onMoveFeedToCategory={handleMoveFeedToCategory}
        onMoveFeedToNewCategory={handleMoveFeedToNewCategory}
        onRemovePaperFromCategory={handleRemovePaperFromCategory}
      />

      {/* Mobile Drawer */}
      <MobileDrawer isOpen={isMobileMenuOpen} onClose={() => setIsMobileMenuOpen(false)}>
        <NewSidebarContent
          feedCategories={feedCategories}
          paperCategories={paperCategories}
          feeds={feeds}
          papers={papers}
          onAddClick={() => {
            setIsAddDialogOpen(true)
            setIsMobileMenuOpen(false)
          }}
          onFeedClick={(id) => { selectFeed(id); setIsMobileMenuOpen(false) }}
          onPaperClick={(id) => { selectPaper(id); setSearchParams(prev => { prev.delete('id'); prev.set('paper', id); return prev }); setIsMobileMenuOpen(false) }}
          onRenameCategory={handleRenameCategory}
          onDeleteCategory={handleDeleteCategory}
          onMoveFeedToCategory={handleMoveFeedToCategory}
          onMoveFeedToNewCategory={handleMoveFeedToNewCategory}
          onRemovePaperFromCategory={handleRemovePaperFromCategory}
        />
      </MobileDrawer>

      {/* Middle column: ItemList (feed items) or Paper list */}
      <div className="hidden md:flex w-80 lg:w-96 border-r flex-shrink-0 flex-col bg-background">
        <div className="p-4 border-b flex items-center justify-between">
          <div className="flex items-center gap-3">
            <MobileMenuButton
              onClick={() => setIsMobileMenuOpen(true)}
              unreadCount={0}
            />
            <h1 className="text-lg font-bold truncate">
              {selectionType === 'paper' ? 'Papers' : middleColumnTitle}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            {selectedFeedId && (
              <button
                onClick={() => handleMarkAllRead(selectedFeedId)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Mark all read
              </button>
            )}
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {selectionType === 'paper' ? (
            papers.map((p) => (
              <button
                key={p.id}
                onClick={() => {
                  selectPaper(p.id)
                  setSearchParams(prev => { prev.delete('id'); prev.set('paper', p.id); return prev })
                }}
                className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors hover:bg-accent ${
                  selectedPaperId === p.id ? 'bg-accent' : ''
                }`}
              >
                <div className="font-medium truncate">{p.title || p.original_filename}</div>
                <div className="text-xs text-muted-foreground mt-0.5">
                  {(() => { try { const a = JSON.parse(p.authors); return Array.isArray(a) ? a.slice(0, 2).join(', ') : ''; } catch { return ''; } })()}
                  {p.published_year ? ` · ${p.published_year}` : ''}
                </div>
                {p.status !== 'completed' && (
                  <span className={`inline-block text-xs mt-1 px-1.5 py-0.5 rounded ${
                    p.status === 'pending' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
                    : p.status === 'processing' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300'
                    : 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                  }`}>
                    {p.status}
                  </span>
                )}
              </button>
            ))
          ) : (
            <>
              <ItemList
                articles={items}
                selectedItemId={selectedItemId}
                onItemClick={handleItemClick}
                onToggleStar={handleToggleStar}
                onToggleRead={handleToggleRead}
                isLoading={itemsLoading}
              />

              {hasMore && (
                <div ref={observerTarget} className="py-8 text-center">
                  {isFetchingNextPage && (
                    <div className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Right column (desktop): ArticlePanel or Paper content */}
      <div className="flex-1 overflow-hidden hidden lg:flex">
        {selectionType === 'paper' && selectedPaperId && paper ? (
          <div className="flex-1 overflow-y-auto">
            {paper.status === 'pending' || paper.status === 'processing' ? (
              <div className="flex flex-col items-center justify-center py-24 text-center px-4">
                <Loader2 className="h-12 w-12 animate-spin mx-auto mb-4 text-primary" />
                <h2 className="text-xl font-semibold mb-2">Converting Paper...</h2>
                <p className="text-muted-foreground mb-4">{paper.original_filename}</p>
                {paperStatus && (
                  <div className="w-64 bg-muted rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all"
                      style={{ width: `${paperStatus.progress}%` }}
                    />
                  </div>
                )}
              </div>
            ) : paper.status === 'failed' ? (
              <div className="flex flex-col items-center justify-center py-24 text-center px-4">
                <AlertCircle className="h-12 w-12 mx-auto mb-4 text-destructive" />
                <h2 className="text-xl font-semibold mb-2">Conversion Failed</h2>
                <p className="text-muted-foreground mb-4">{paper.error || 'Unknown error'}</p>
              </div>
            ) : (
              <div className="max-w-4xl mx-auto py-6 px-6">
                <div className="mb-6">
                  <PaperMeta paper={paper} onDownload={handleDownloadPaper} />
                </div>
                {paper.markdown_content ? (
                  <MarkdownRenderer content={paper.markdown_content} />
                ) : (
                  <p className="text-center text-muted-foreground py-8">No content available</p>
                )}
              </div>
            )}
          </div>
        ) : (
          <ArticlePanel
            itemId={selectedItemId}
            showBackButton={false}
          />
        )}
      </div>

      {/* Mobile: full-screen ArticlePanel when an item is selected */}
      {selectedItemId && (
        <div className="fixed inset-0 bg-background z-50 md:hidden">
          <ArticlePanel
            itemId={selectedItemId}
            showBackButton={true}
            onClose={handleCloseArticle}
          />
        </div>
      )}

      {/* Mobile: full-screen paper content when a paper is selected */}
      {selectionType === 'paper' && selectedPaperId && paper && !selectedItemId && (
        <div className="fixed inset-0 bg-background z-40 md:hidden">
          <Button
            variant="ghost"
            onClick={() => selectPaper(selectedPaperId)}
            className="m-2"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          {paper.status === 'completed' ? (
            <div className="px-4 pb-8">
              <div className="mb-4">
                <PaperMeta paper={paper} onDownload={handleDownloadPaper} />
              </div>
              {paper.markdown_content ? (
                <MarkdownRenderer content={paper.markdown_content} />
              ) : (
                <p className="text-center text-muted-foreground py-8">No content available</p>
              )}
            </div>
          ) : paper.status === 'failed' ? (
            <div className="flex flex-col items-center justify-center py-24 text-center px-4">
              <AlertCircle className="h-12 w-12 mx-auto mb-4 text-destructive" />
              <h2 className="text-xl font-semibold mb-2">Conversion Failed</h2>
              <p className="text-muted-foreground">{paper.error || 'Unknown error'}</p>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-24 text-center px-4">
              <Loader2 className="h-12 w-12 animate-spin mx-auto mb-4 text-primary" />
              <h2 className="text-xl font-semibold mb-2">Converting Paper...</h2>
              <p className="text-muted-foreground">{paper.original_filename}</p>
              {paperStatus && (
                <div className="w-64 bg-muted rounded-full h-2 mt-4">
                  <div
                    className="bg-primary h-2 rounded-full transition-all"
                    style={{ width: `${paperStatus.progress}%` }}
                  />
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Mobile: full-screen ItemList (no selection) */}
      <main className="flex-1 overflow-y-auto md:hidden">
        <div className="py-4 px-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <MobileMenuButton
                onClick={() => setIsMobileMenuOpen(true)}
                unreadCount={0}
              />
              <h1 className="text-lg font-bold">
                {selectionType === 'paper' ? 'Papers' : middleColumnTitle}
              </h1>
            </div>
            <div className="flex items-center gap-2">
              <ThemeToggle />
            </div>
          </div>

          {selectionType === 'paper' ? (
            papers.map((p) => (
              <button
                key={p.id}
                onClick={() => {
                  selectPaper(p.id)
                  setSearchParams(prev => { prev.delete('id'); prev.set('paper', p.id); return prev })
                }}
                className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors hover:bg-accent ${
                  selectedPaperId === p.id ? 'bg-accent' : ''
                }`}
              >
                <div className="font-medium truncate">{p.title || p.original_filename}</div>
                <div className="text-xs text-muted-foreground mt-0.5">
                  {(() => { try { const a = JSON.parse(p.authors); return Array.isArray(a) ? a.slice(0, 2).join(', ') : ''; } catch { return ''; } })()}
                  {p.published_year ? ` · ${p.published_year}` : ''}
                </div>
              </button>
            ))
          ) : (
            <ItemList
              articles={items}
              selectedItemId={selectedItemId}
              onItemClick={handleItemClick}
              onToggleStar={handleToggleStar}
              onToggleRead={handleToggleRead}
              isLoading={itemsLoading}
            />
          )}
        </div>
      </main>

      <AddDialog
        open={isAddDialogOpen}
        onOpenChange={setIsAddDialogOpen}
        onSuccess={() => {
          refetchFeeds()
          refetch()
        }}
      />

      <KeyboardShortcutsModal
        isOpen={isShortcutsModalOpen}
        onClose={() => setIsShortcutsModalOpen(false)}
      />
    </div>
  )
}

export default ItemsPage
