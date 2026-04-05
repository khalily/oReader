import { useMemo } from 'react'
import { Rss } from 'lucide-react'
import { CategoryRow } from './CategoryRow'
import { FeedRow } from './FeedRow'
import type { Category } from '@/types/category'
import type { UserFeed } from '@/types/feed'

interface FeedSectionProps {
  categories: Category[]
  feeds: UserFeed[]
  selectedFeedId: string | null
  expandedCategoryIds: Set<string>
  allFeedCategories: Category[]
  onToggleCategory: (categoryId: string) => void
  onFeedClick: (feedId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
  onMoveFeedToCategory: (feedId: string, categoryId: string) => void
  onMoveFeedToNewCategory: (feedId: string, categoryName: string) => void
}

export function FeedSection({
  categories,
  feeds,
  selectedFeedId,
  expandedCategoryIds,
  allFeedCategories,
  onToggleCategory,
  onFeedClick,
  onRenameCategory,
  onDeleteCategory,
  onMoveFeedToCategory,
  onMoveFeedToNewCategory,
}: FeedSectionProps) {
  const { categorizedFeeds, uncategorizedFeeds } = useMemo(() => {
    const catIds = new Set(categories.map((c) => c.id))
    const grouped = new Map<string, UserFeed[]>()
    const uncategorized: UserFeed[] = []

    for (const feed of feeds) {
      if (feed.category_id && catIds.has(feed.category_id)) {
        const list = grouped.get(feed.category_id) || []
        list.push(feed)
        grouped.set(feed.category_id, list)
      } else {
        uncategorized.push(feed)
      }
    }

    return { categorizedFeeds: grouped, uncategorizedFeeds: uncategorized }
  }, [categories, feeds])

  return (
    <div>
      <div className="flex items-center gap-2 px-2 py-2">
        <Rss className="h-3.5 w-3.5 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Feeds
        </span>
        <span className="text-xs text-muted-foreground">({feeds.length})</span>
      </div>

      {categories.map((category) => {
        const catFeeds = categorizedFeeds.get(category.id) || []
        const isExpanded = expandedCategoryIds.has(category.id)

        return (
          <div key={category.id}>
            <CategoryRow
              category={category}
              isExpanded={isExpanded}
              itemCount={catFeeds.length}
              onToggle={() => onToggleCategory(category.id)}
              onRename={onRenameCategory}
              onDelete={onDeleteCategory}
            />
            {isExpanded &&
              catFeeds.map((feed) => (
                <FeedRow
                  key={feed.id}
                  feed={feed}
                  isSelected={feed.id === selectedFeedId}
                  onClick={onFeedClick}
                  allFeedCategories={allFeedCategories}
                  onMoveToCategory={onMoveFeedToCategory}
                  onMoveToNewCategory={onMoveFeedToNewCategory}
                />
              ))}
          </div>
        )
      })}

      {uncategorizedFeeds.length > 0 && (
        <>
          {categories.length > 0 && (
            <div className="px-2 py-1">
              <span className="text-xs text-muted-foreground">未分类</span>
            </div>
          )}
          {uncategorizedFeeds.map((feed) => (
            <FeedRow
              key={feed.id}
              feed={feed}
              isSelected={feed.id === selectedFeedId}
              onClick={onFeedClick}
              allFeedCategories={allFeedCategories}
              onMoveToCategory={onMoveFeedToCategory}
              onMoveToNewCategory={onMoveFeedToNewCategory}
            />
          ))}
        </>
      )}
    </div>
  )
}
