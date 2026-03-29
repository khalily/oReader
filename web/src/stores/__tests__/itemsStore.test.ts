import { describe, it, expect, beforeEach } from 'vitest'
import { act } from '@testing-library/react'
import { useItemsStore } from '../itemsStore'
import type { Article } from '@/types/feed'

// Mock Article data
const mockArticle: Article = {
  id: '1',
  title: 'Test Article',
  link: 'https://example.com/test',
  description: 'Test Description',
  content: 'Test Content',
  pub_date: '2024-01-01T00:00:00Z',
  feed_id: 'feed-1',
  feed_title: 'Test Feed',
  user_state: {
    item_id: '1',
    is_read: false,
    is_starred: false,
    read_at: null,
  },
}

const mockArticle2: Article = {
  id: '2',
  title: 'Test Article 2',
  link: 'https://example.com/test2',
  description: 'Test Description 2',
  content: 'Test Content 2',
  pub_date: '2024-01-02T00:00:00Z',
  feed_id: 'feed-1',
  feed_title: 'Test Feed',
  user_state: {
    item_id: '2',
    is_read: true,
    is_starred: true,
    read_at: '2024-01-02T12:00:00Z',
  },
}

describe('itemsStore', () => {
  beforeEach(() => {
    // Reset store before each test
    act(() => {
      useItemsStore.setState({ items: [] })
    })
  })

  describe('setItems', () => {
    it('should set items array', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle, mockArticle2])
      })

      const state = useItemsStore.getState()
      expect(state.items).toHaveLength(2)
      expect(state.items[0].id).toBe('1')
      expect(state.items[1].id).toBe('2')
    })

    it('should replace existing items', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle])
      })
      act(() => {
        useItemsStore.getState().setItems([mockArticle2])
      })

      const state = useItemsStore.getState()
      expect(state.items).toHaveLength(1)
      expect(state.items[0].id).toBe('2')
    })

    it('should handle empty array', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle])
      })
      act(() => {
        useItemsStore.getState().setItems([])
      })

      const state = useItemsStore.getState()
      expect(state.items).toHaveLength(0)
    })
  })

  describe('updateItemState', () => {
    it('should update is_starred state', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle])
      })
      act(() => {
        useItemsStore.getState().updateItemState('1', { is_starred: true })
      })

      const state = useItemsStore.getState()
      expect(state.items[0].user_state?.is_starred).toBe(true)
    })

    it('should update is_read state', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle])
      })
      act(() => {
        useItemsStore.getState().updateItemState('1', { is_read: true })
      })

      const state = useItemsStore.getState()
      expect(state.items[0].user_state?.is_read).toBe(true)
    })

    it('should create user_state if not exists', () => {
      const articleWithoutState: Article = {
        ...mockArticle,
        id: '3',
        user_state: undefined,
      }

      act(() => {
        useItemsStore.getState().setItems([articleWithoutState])
      })
      act(() => {
        useItemsStore.getState().updateItemState('3', { is_starred: true })
      })

      const state = useItemsStore.getState()
      expect(state.items[0].user_state?.is_starred).toBe(true)
      expect(state.items[0].user_state?.item_id).toBe('3')
    })

    it('should not modify other items', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle, mockArticle2])
      })
      act(() => {
        useItemsStore.getState().updateItemState('1', { is_starred: true })
      })

      const state = useItemsStore.getState()
      expect(state.items[0].user_state?.is_starred).toBe(true)
      // Article 2 should be unchanged
      expect(state.items[1].user_state?.is_starred).toBe(true)
      expect(state.items[1].user_state?.is_read).toBe(true)
    })

    it('should preserve existing state when updating', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle2])
      })
      act(() => {
        useItemsStore.getState().updateItemState('2', { is_read: false })
      })

      const state = useItemsStore.getState()
      // is_starred should be preserved
      expect(state.items[0].user_state?.is_starred).toBe(true)
      // is_read should be updated
      expect(state.items[0].user_state?.is_read).toBe(false)
    })
  })

  describe('getItem', () => {
    it('should return item by id', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle, mockArticle2])
      })

      const item = useItemsStore.getState().getItem('1')
      expect(item).toBeDefined()
      expect(item?.id).toBe('1')
    })

    it('should return undefined for non-existent id', () => {
      act(() => {
        useItemsStore.getState().setItems([mockArticle])
      })

      const item = useItemsStore.getState().getItem('999')
      expect(item).toBeUndefined()
    })

    it('should return undefined when items array is empty', () => {
      const item = useItemsStore.getState().getItem('1')
      expect(item).toBeUndefined()
    })
  })
})
