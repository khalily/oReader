import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { FeedList } from '../FeedList'
import type { UserFeed } from '@/types/feed'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Plus: () => <div data-testid="plus-icon" />,
  Trash2: () => <div data-testid="trash-icon" />,
  RefreshCw: () => <div data-testid="refresh-icon" />,
  Rss: () => <div data-testid="rss-icon" />,
}))

const API_BASE_URL = '/api/v1'

const mockFeeds: UserFeed[] = [
  {
    id: 'feed-1',
    title: 'Example Feed',
    feed_url: 'https://example.com/feed.xml',
    description: 'An example RSS feed',
    image_url: 'https://example.com/image.png',
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    unread_count: 5,
    position: 1,
  },
  {
    id: 'feed-2',
    title: 'Another Feed',
    feed_url: 'https://another.com/feed.xml',
    description: 'Another RSS feed',
    image_url: null,
    last_fetched_at: '2024-01-02T00:00:00Z',
    created_at: '2024-01-02T00:00:00Z',
    unread_count: 0,
    position: 2,
  },
]

const handlers = [
  http.delete(`${API_BASE_URL}/feeds/:id`, ({ params }) => {
    if (params.id === 'feed-1') {
      return new HttpResponse(null, { status: 204 })
    }
    return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Feed not found' } }, { status: 404 })
  }),
  http.post(`${API_BASE_URL}/feeds/:id/refresh`, ({ params }) => {
    return HttpResponse.json({ updated: true, new_item_count: 3 })
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

describe('FeedList component', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should render list of feeds', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={mockFeeds}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId={null}
        refreshingFeedIds={new Set()}
      />,
      { wrapper }
    )

    expect(screen.getByText('Example Feed')).toBeInTheDocument()
    expect(screen.getByText('Another Feed')).toBeInTheDocument()
    expect(screen.getByText('An example RSS feed')).toBeInTheDocument()
  })

  it('should show empty state when no feeds', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={[]}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId={null}
        refreshingFeedIds={new Set()}
      />,
      { wrapper }
    )

    expect(screen.getByText(/no feeds yet/i)).toBeInTheDocument()
  })

  it('should call onClick when feed is clicked', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={mockFeeds}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId={null}
        refreshingFeedIds={new Set()}
      />,
      { wrapper }
    )

    fireEvent.click(screen.getByText('Example Feed'))
    expect(handleClick).toHaveBeenCalledWith('feed-1')
  })

  it('should highlight selected feed', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={mockFeeds}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId="feed-1"
        refreshingFeedIds={new Set()}
      />,
      { wrapper }
    )

    // The selected feed should have a different background
    const feedCard = screen.getByText('Example Feed').closest('.cursor-pointer')
    expect(feedCard).toHaveClass('bg-accent')
  })

  it('should disable refresh button for refreshing feeds', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={mockFeeds}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId={null}
        refreshingFeedIds={new Set(['feed-1'])}
      />,
      { wrapper }
    )

    const refreshBtns = screen.getAllByTestId('refresh-icon')
    const firstRefreshBtn = refreshBtns[0].closest('button')
    expect(firstRefreshBtn).toBeDisabled()
  })

  it('should call onDelete when delete button is clicked', async () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleClick = vi.fn()

    render(
      <FeedList
        feeds={mockFeeds}
        onDelete={handleDelete}
        onRefresh={handleRefresh}
        onClick={handleClick}
        selectedFeedId={null}
        refreshingFeedIds={new Set()}
      />,
      { wrapper }
    )

    const deleteBtns = screen.getAllByTestId('trash-icon')
    const firstDeleteBtn = deleteBtns[0].closest('button')
    fireEvent.click(firstDeleteBtn!)

    await waitFor(() => {
      expect(handleDelete).toHaveBeenCalledWith('feed-1')
    })
  })
})
