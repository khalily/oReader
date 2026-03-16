import { create } from 'zustand'
import type { Article } from '@/types/feed'

interface ItemsState {
  items: Article[]
  setItems: (items: Article[]) => void
  updateItemState: (itemId: string, state: { is_starred?: boolean; is_read?: boolean }) => void
  getItem: (itemId: string) => Article | undefined
}

export const useItemsStore = create<ItemsState>((set, get) => ({
  items: [],

  setItems: (items) => set({ items }),

  updateItemState: (itemId, state) => {
    set((currentState) => ({
      items: currentState.items.map((item) => {
        if (item.id === itemId) {
          return {
            ...item,
            user_state: {
              ...item.user_state,
              ...state,
            },
          }
        }
        return item
      }),
    }))
  },

  getItem: (itemId) => {
    return get().items.find((item) => item.id === itemId)
  },
}))
