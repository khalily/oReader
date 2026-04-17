import { create } from 'zustand'

export type SidebarSelectionType = 'feed' | 'paper' | null

interface SidebarState {
  selectedFeedId: string | null
  selectedPaperId: string | null
  selectionType: SidebarSelectionType
  expandedCategoryIds: Set<string>

  selectFeed: (feedId: string) => void
  selectPaper: (paperId: string) => void
  clearSelection: () => void
  toggleCategory: (categoryId: string) => void
  expandCategory: (categoryId: string) => void
}

export const useSidebarStore = create<SidebarState>((set) => ({
  selectedFeedId: null,
  selectedPaperId: null,
  selectionType: null,
  expandedCategoryIds: new Set(),

  selectFeed: (feedId) =>
    set({
      selectedFeedId: feedId,
      selectedPaperId: null,
      selectionType: 'feed',
    }),

  selectPaper: (paperId) =>
    set({
      selectedFeedId: null,
      selectedPaperId: paperId,
      selectionType: 'paper',
    }),

  clearSelection: () =>
    set({
      selectedFeedId: null,
      selectedPaperId: null,
      selectionType: null,
    }),

  toggleCategory: (categoryId) =>
    set((state) => {
      const next = new Set(state.expandedCategoryIds)
      if (next.has(categoryId)) {
        next.delete(categoryId)
      } else {
        next.add(categoryId)
      }
      return { expandedCategoryIds: next }
    }),

  expandCategory: (categoryId) =>
    set((state) => {
      const next = new Set(state.expandedCategoryIds)
      next.add(categoryId)
      return { expandedCategoryIds: next }
    }),
}))
