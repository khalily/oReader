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
            user_state: item.user_state
              ? {
                  ...item.user_state,
                  item_id: item.user_state.item_id ?? item.id,
                  ...state,
                }
              : {
                  item_id: item.id,
                  is_read: state.is_read ?? false,
                  is_starred: state.is_starred ?? false,
                  read_at: null,
                },
          } as Article
        }
        return item
      }),
    }))
  },

  getItem: (itemId) => {
    return get().items.find((item) => item.id === itemId)
  },
}))
