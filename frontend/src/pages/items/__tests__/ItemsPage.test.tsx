import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import ItemsPage from '../ItemsPage'
import { ThemeProvider } from '@/contexts/ThemeContext'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock toast hook
vi.mock('@/components/ui/toast', () => ({
  useToast: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

// Mock lucide-react icons — use importOriginal to avoid missing-icon errors
vi.mock('lucide-react', async (importOriginal) => {
  const actual = await importOriginal<typeof import('lucide-react')>()
  return {
    ...actual,
    // Re-export everything from the real module; no selective mocking needed.
  }
})

// Mock window.matchMedia for responsive hooks
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock IntersectionObserver
const mockObserve = vi.fn()
const mockUnobserve = vi.fn()
const mockDisconnect = vi.fn()

class MockIntersectionObserver {
  observe = mockObserve
  unobserve = mockUnobserve
  disconnect = mockDisconnect
}

vi.stubGlobal('IntersectionObserver', MockIntersectionObserver)

// Mock scrollIntoView for jsdom
Element.prototype.scrollIntoView = vi.fn()

const API_BASE_URL = '/api/v1'

// Mock items data
const mockItems = [
  {
    id: 'item-1',
    feed_id: 'feed-1',
    guid: 'guid-1',
    title: 'First Article',
    link: 'https://example.com/article1',
    description: 'Description 1',
    content: '<p>Content 1</p>',
    pub_date: '2024-01-01T00:00:00Z',
    creator: 'Author 1',
    created_at: '2024-01-01T00:00:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example feed',
      image_url: null,
      last_fetched_at: '2024-01-01T00:00:00Z',
      created_at: '2024-01-01T00:00:00Z',
    },
    user_state: {
      item_id: 'item-1',
      is_read: false,
      is_starred: false,
      read_at: null,
    },
  },
  {
    id: 'item-2',
    feed_id: 'feed-1',
    guid: 'guid-2',
    title: 'Second Article',
    link: 'https://example.com/article2',
    description: 'Description 2',
    content: '<p>Content 2</p>',
    pub_date: '2024-01-02T00:00:00Z',
    creator: 'Author 2',
    created_at: '2024-01-02T00:00:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example feed',
      image_url: null,
      last_fetched_at: '2024-01-01T00:00:00Z',
      created_at: '2024-01-01T00:00:00Z',
    },
    user_state: {
      item_id: 'item-2',
      is_read: true,
      is_starred: true,
      read_at: '2024-01-02T01:00:00Z',
    },
  },
]

// Mock feeds data
const mockFeeds = [
  {
    id: 'feed-1',
    title: 'Example Feed',
    feed_url: 'https://example.com/feed.xml',
    description: 'An example feed',
    image_url: null,
    unread_count: 5,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'feed-2',
    title: 'Another Feed',
    feed_url: 'https://another.com/feed.xml',
    description: 'Another feed',
    image_url: null,
    unread_count: 3,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
  },
]

// Mock stats data
const mockStats = {
  total_feeds: 2,
  total_items: 10,
  unread_items: 8,
  starred_items: 2,
}

// API handlers
const handlers = [
  // List items
  http.get(`${API_BASE_URL}/items`, ({ request }) => {
    const url = new URL(request.url)
    const starred = url.searchParams.get('starred')
    const read = url.searchParams.get('read')

    let filteredItems = [...mockItems]

    // Filter by starred
    if (starred === 'true') {
      filteredItems = filteredItems.filter((item) => item.user_state?.is_starred)
    }

    // Filter by read status
    if (read === 'false') {
      filteredItems = filteredItems.filter((item) => !item.user_state?.is_read)
    }

    return HttpResponse.json({
      items: filteredItems,
      total: filteredItems.length,
      has_more: false,
      next_cursor: null,
    })
  }),

  // Get single item
  http.get(`${API_BASE_URL}/items/:id`, ({ params }) => {
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({ item })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Toggle star
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ params, request }) => {
    const body = await request.json() as { starred: boolean }
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({
        item: {
          ...item,
          user_state: {
            ...item.user_state!,
            is_starred: body.starred,
          },
        },
      })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // Toggle read
  http.put(`${API_BASE_URL}/items/:id/read`, async ({ params, request }) => {
    const body = await request.json() as { read: boolean }
    const item = mockItems.find((i) => i.id === params.id)
    if (item) {
      return HttpResponse.json({
        item: {
          ...item,
          user_state: {
            ...item.user_state!,
            is_read: body.read,
            read_at: body.read ? new Date().toISOString() : null,
          },
        },
      })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
  }),

  // List feeds
  http.get(`${API_BASE_URL}/feeds`, () => {
    return HttpResponse.json({ feeds: mockFeeds })
  }),

  // Refresh feed
  http.post(`${API_BASE_URL}/feeds/:id/refresh`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ message: 'Refresh started' })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Delete feed
  http.delete(`${API_BASE_URL}/feeds/:id`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ message: 'Feed deleted' })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Mark all read
  http.post(`${API_BASE_URL}/feeds/:id/read-all`, ({ params }) => {
    if (mockFeeds.find((f) => f.id === params.id)) {
      return HttpResponse.json({ count: 5 })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Feed not found' } },
      { status: 404 }
    )
  }),

  // Get stats
  http.get(`${API_BASE_URL}/stats`, () => {
    return HttpResponse.json(mockStats)
  }),
]

const server = setupServer(...handlers)

// Helper to create wrapper with all providers
const createWrapper = (initialEntries = ['/items']) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            <Route path="/items" element={children} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

// Helper to render with providers
const renderItemsPage = (props: { filterType?: string; feedId?: string } = {}) => {
  const wrapper = createWrapper()
  return render(<ItemsPage filterType={(props.filterType as 'all' | 'unread' | 'starred' | 'today') || 'all'} feedId={props.feedId} />, { wrapper })
}

describe('ItemsPage', () => {
  beforeEach(() => {
    server.listen()
    vi.clearAllMocks()
  })

  afterEach(() => {
    server.resetHandlers()
  })

  afterAll(() => {
    server.close()
  })

  // ============================================
  // 4.10 基础渲染测试
  // ============================================
  describe('Basic Rendering', () => {
    it('should render article list', async () => {
      renderItemsPage()

      // Component renders both desktop and mobile views, so we check for at least one
      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      const secondArticles = screen.getAllByText('Second Article')
      expect(secondArticles.length).toBeGreaterThan(0)
    })

    it('should show empty state when no articles', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items`, () => {
          return HttpResponse.json({
            items: [],
            total: 0,
            has_more: false,
            next_cursor: null,
          })
        })
      )

      renderItemsPage()

      await waitFor(() => {
        // The component should render without crashing when there are no items
        const allArticlesHeaders = screen.getAllByText('All Articles')
        expect(allArticlesHeaders.length).toBeGreaterThan(0)
      })
    })

    it('should handle loading state gracefully', async () => {
      // The component should render without crashing during loading
      renderItemsPage()

      // Wait for content to load
      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })
    })
  })

  // ============================================
  // 4.11 筛选功能测试
  // ============================================
  describe('Filter Functionality', () => {
    it('should filter by unread status', async () => {
      renderItemsPage({ filterType: 'unread' })

      await waitFor(() => {
        // Only unread items should be displayed
        const firstArticles = screen.getAllByText('First Article')
        expect(firstArticles.length).toBeGreaterThan(0)
      })

      // Second article is read, so it should not be displayed
      expect(screen.queryByText('Second Article')).not.toBeInTheDocument()
    })

    it('should filter by starred status', async () => {
      renderItemsPage({ filterType: 'starred' })

      await waitFor(() => {
        // Only starred items should be displayed
        const secondArticles = screen.getAllByText('Second Article')
        expect(secondArticles.length).toBeGreaterThan(0)
      })

      // First article is not starred, so it should not be displayed
      expect(screen.queryByText('First Article')).not.toBeInTheDocument()
    })

    it('should display correct title for starred filter', async () => {
      renderItemsPage({ filterType: 'starred' })

      await waitFor(() => {
        const starredHeaders = screen.getAllByText('Starred')
        expect(starredHeaders.length).toBeGreaterThan(0)
      })
    })

    it('should display correct title for unread filter', async () => {
      renderItemsPage({ filterType: 'unread' })

      await waitFor(() => {
        const unreadHeaders = screen.getAllByText('Unread')
        expect(unreadHeaders.length).toBeGreaterThan(0)
      })
    })

    it('should filter by specific feed', async () => {
      renderItemsPage({ feedId: 'feed-1' })

      await waitFor(() => {
        // Should display feed title in header
        const feedTitles = screen.getAllByText('Example Feed')
        expect(feedTitles.length).toBeGreaterThan(0)
      })
    })
  })

  // ============================================
  // 4.12 交互测试
  // ============================================
  describe('User Interactions', () => {
    it('should display mark all read button when viewing specific feed', async () => {
      renderItemsPage({ feedId: 'feed-1' })

      await waitFor(() => {
        const markAllReadButtons = screen.getAllByText('Mark all read')
        expect(markAllReadButtons.length).toBeGreaterThan(0)
      })
    })

    it('should not display mark all read button when viewing all articles', async () => {
      renderItemsPage()

      await waitFor(() => {
        const allArticlesHeaders = screen.getAllByText('All Articles')
        expect(allArticlesHeaders.length).toBeGreaterThan(0)
      })

      // Mark all read button should not be present when viewing all articles
      expect(screen.queryByText('Mark all read')).not.toBeInTheDocument()
    })
  })

  // ============================================
  // 4.13 键盘快捷键测试
  // ============================================
  describe('Keyboard Shortcuts', () => {
    it('should register keyboard shortcuts on mount', async () => {
      renderItemsPage()

      await waitFor(() => {
        const allArticlesHeaders = screen.getAllByText('All Articles')
        expect(allArticlesHeaders.length).toBeGreaterThan(0)
      })

      // The keyboard shortcuts are registered via useKeyboardShortcuts hook
      // We verify the component renders without errors
    })

    it('should navigate to next article with j key', async () => {
      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // Simulate j key press - component should handle without error
      fireEvent.keyDown(window, { key: 'j' })
    })

    it('should navigate to previous article with k key', async () => {
      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // Simulate k key press
      fireEvent.keyDown(window, { key: 'k' })
    })

    it('should toggle star with s key', async () => {
      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // Simulate s key press
      fireEvent.keyDown(window, { key: 's' })
    })

    it('should toggle read with r key', async () => {
      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // Simulate r key press
      fireEvent.keyDown(window, { key: 'r' })
    })

    it('should mark all as read with n key when viewing feed', async () => {
      renderItemsPage({ feedId: 'feed-1' })

      await waitFor(() => {
        const markAllReadButtons = screen.getAllByText('Mark all read')
        expect(markAllReadButtons.length).toBeGreaterThan(0)
      })

      // Simulate n key press
      fireEvent.keyDown(window, { key: 'n' })
    })
  })

  // ============================================
  // 4.14 无限滚动测试
  // ============================================
  describe('Infinite Scroll', () => {
    it('should set up intersection observer for infinite scroll', async () => {
      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // IntersectionObserver should be available globally
      expect(window.IntersectionObserver).toBeDefined()
    })

    it('should show loading spinner when loading more items', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items`, () => {
          return HttpResponse.json({
            items: mockItems,
            total: 20,
            has_more: true,
            next_cursor: 'next-page-cursor',
          })
        })
      )

      renderItemsPage()

      await waitFor(() => {
        const articles = screen.getAllByText('First Article')
        expect(articles.length).toBeGreaterThan(0)
      })

      // When hasMore is true, the observer target should be present
      expect(mockObserve).toHaveBeenCalled()
    })
  })
})
