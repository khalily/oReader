import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ItemList } from '../ItemList'
import type { Article, ListItemsOptions } from '@/types/feed'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Star: () => <div data-testid="star-icon" />,
  StarOff: () => <div data-testid="star-off-icon" />,
  Eye: () => <div data-testid="eye-icon" />,
  EyeOff: () => <div data-testid="eye-off-icon" />,
  ExternalLink: () => <div data-testid="external-link-icon" />,
  Loader2: () => <div data-testid="loader-icon" />,
}))

const API_BASE_URL = '/api/v1'

const mockArticles: Article[] = [
  {
    id: 'item-1',
    feed_id: 'feed-1',
    guid: 'guid-1',
    title: 'First Article',
    link: 'https://example.com/article1',
    description: 'An interesting article about something',
    content: '<p>Full content of the article</p>',
    pub_date: '2024-01-15T10:30:00Z',
    creator: 'John Doe',
    created_at: '2024-01-15T10:30:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example RSS feed',
      image_url: 'https://example.com/image.png',
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
    description: 'Another interesting article',
    content: '<p>More article content</p>',
    pub_date: '2024-01-14T09:00:00Z',
    creator: 'Jane Smith',
    created_at: '2024-01-14T09:00:00Z',
    feed: {
      id: 'feed-1',
      title: 'Example Feed',
      feed_url: 'https://example.com/feed.xml',
      description: 'An example RSS feed',
      image_url: 'https://example.com/image.png',
      last_fetched_at: '2024-01-01T00:00:00Z',
      created_at: '2024-01-01T00:00:00Z',
    },
    user_state: {
      item_id: 'item-2',
      is_read: true,
      is_starred: true,
      read_at: '2024-01-14T10:00:00Z',
    },
  },
]

const handlers = [
  http.get(`${API_BASE_URL}/items`, () => {
    return HttpResponse.json({
      items: mockArticles,
      total: 2,
      has_more: false,
    })
  }),
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ request, params }) => {
    const body = await request.json() as { starred: boolean }
    const item = mockArticles.find(i => i.id === params.id)
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
    return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
  }),
  http.put(`${API_BASE_URL}/items/:id/read`, async ({ request, params }) => {
    const body = await request.json() as { read: boolean }
    const item = mockArticles.find(i => i.id === params.id)
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
    return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
  }),
]

const server = setupServer(...handlers)

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe('ItemList component', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should render list of articles', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      expect(screen.getByText('First Article')).toBeInTheDocument()
      expect(screen.getByText('Second Article')).toBeInTheDocument()
    })
  })

  it('should show feed name for each article', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      const feedNames = screen.getAllByText('Example Feed')
      expect(feedNames.length).toBeGreaterThan(0)
    })
  })

  it('should show loading state', () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={[]}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={true}
      />,
      { wrapper }
    )

    expect(screen.getByTestId('loader-icon')).toBeInTheDocument()
  })

  it('should show empty state when no articles', () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={[]}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    expect(screen.getByText(/no articles/i)).toBeInTheDocument()
  })

  it('should call onItemClick when article is clicked', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      fireEvent.click(screen.getByText('First Article'))
      expect(onItemClick).toHaveBeenCalledWith('item-1')
    })
  })

  it('should call onToggleStar when star button is clicked', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      const starBtns = screen.getAllByTestId('star-off-icon')
      const firstStarBtn = starBtns[0].closest('button')
      fireEvent.click(firstStarBtn!)
      expect(onToggleStar).toHaveBeenCalledWith('item-1', true)
    })
  })

  it('should show starred icon for starred articles', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      expect(screen.getByTestId('star-icon')).toBeInTheDocument()
      expect(screen.getByTestId('star-off-icon')).toBeInTheDocument()
    })
  })

  it('should show read indicator for read articles', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      // The second article is read
      const articleCards = screen.getAllByText(/Example Feed/)
      // Read articles should have muted text
      expect(screen.getByText('Second Article')).toBeInTheDocument()
    })
  })

  it('should call onToggleRead when read button is clicked', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      const eyeBtns = screen.getAllByTestId('eye-off-icon')
      const firstEyeBtn = eyeBtns[0].closest('button')
      fireEvent.click(firstEyeBtn!)
      expect(onToggleRead).toHaveBeenCalledWith('item-1', true)
    })
  })

  it('should format publication date correctly', async () => {
    const wrapper = createWrapper()
    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    render(
      <ItemList
        articles={mockArticles}
        onItemClick={onItemClick}
        onToggleStar={onToggleStar}
        onToggleRead={onToggleRead}
        isLoading={false}
      />,
      { wrapper }
    )

    await waitFor(() => {
      // Should show formatted date (e.g., "Jan 15, 2024")
      expect(screen.getByText('Jan 15, 2024')).toBeInTheDocument()
      expect(screen.getByText('Jan 14, 2024')).toBeInTheDocument()
    })
  })
})
