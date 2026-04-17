import { useMemo } from 'react'
import { FileText } from 'lucide-react'
import { CategoryRow } from './CategoryRow'
import { PaperRow } from './PaperRow'
import type { Category } from '@/types/category'
import type { Paper } from '@/types/paper'

interface PaperSectionProps {
  categories: Category[]
  papers: Paper[]
  selectedPaperId: string | null
  expandedCategoryIds: Set<string>
  onToggleCategory: (categoryId: string) => void
  onPaperClick: (paperId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
  onRemovePaperFromCategory?: (paperId: string) => void
}

export function PaperSection({
  categories,
  papers,
  selectedPaperId,
  expandedCategoryIds,
  onToggleCategory,
  onPaperClick,
  onRenameCategory,
  onDeleteCategory,
  onRemovePaperFromCategory,
}: PaperSectionProps) {
  const { categorizedPapers, uncategorizedPapers } = useMemo(() => {
    const catIds = new Set(categories.map((c) => c.id))
    const grouped = new Map<string, Paper[]>()
    const uncategorized: Paper[] = []

    for (const paper of papers) {
      if (paper.category_id && catIds.has(paper.category_id)) {
        const list = grouped.get(paper.category_id) || []
        list.push(paper)
        grouped.set(paper.category_id, list)
      } else {
        uncategorized.push(paper)
      }
    }

    return { categorizedPapers: grouped, uncategorizedPapers: uncategorized }
  }, [categories, papers])

  return (
    <div>
      <div className="flex items-center gap-2 px-2 py-2">
        <FileText className="h-3.5 w-3.5 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Papers
        </span>
        <span className="text-xs text-muted-foreground">({papers.length})</span>
      </div>

      {categories.map((category) => {
        const catPapers = categorizedPapers.get(category.id) || []
        const isExpanded = expandedCategoryIds.has(category.id)

        return (
          <div key={category.id}>
            <CategoryRow
              category={category}
              isExpanded={isExpanded}
              itemCount={catPapers.length}
              onToggle={() => onToggleCategory(category.id)}
              onRename={onRenameCategory}
              onDelete={onDeleteCategory}
            />
            {isExpanded &&
              catPapers.map((paper) => (
                <PaperRow
                  key={paper.id}
                  paper={paper}
                  isSelected={paper.id === selectedPaperId}
                  onClick={onPaperClick}
                  onRemoveFromCategory={onRemovePaperFromCategory}
                />
              ))}
          </div>
        )
      })}

      {uncategorizedPapers.length > 0 && (
        <>
          {categories.length > 0 && (
            <div className="px-2 py-1">
              <span className="text-xs text-muted-foreground">未分类</span>
            </div>
          )}
          {uncategorizedPapers.map((paper) => (
            <PaperRow
              key={paper.id}
              paper={paper}
              isSelected={paper.id === selectedPaperId}
              onClick={onPaperClick}
            />
          ))}
        </>
      )}
    </div>
  )
}
