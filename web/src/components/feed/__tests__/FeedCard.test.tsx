import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { FeedCard } from '../FeedCard'
import type { UserFeed } from '@/types/feed'

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Trash2: () => <div data-testid="trash-icon" />,
  RefreshCw: () => <div data-testid="refresh-icon" />,
  Rss: () => <div data-testid="rss-icon" />,
}))

const mockFeed: UserFeed = {
  id: 'feed-1',
  title: 'Example Feed',
  feed_url: 'https://example.com/feed.xml',
  description: 'An example RSS feed',
  image_url: 'https://example.com/image.png',
  last_fetched_at: '2024-01-01T00:00:00Z',
  created_at: '2024-01-01T00:00:00Z',
  unread_count: 5,
  position: 1,
}

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

describe('FeedCard component', () => {
  it('should render feed title and description', () => {
    const wrapper = createWrapper()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        onClick={vi.fn()}
      />,
      { wrapper }
    )

    expect(screen.getByText('Example Feed')).toBeInTheDocument()
    expect(screen.getByText('An example RSS feed')).toBeInTheDocument()
  })

  it('should render unread count badge when unread items exist', () => {
    const wrapper = createWrapper()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        onClick={vi.fn()}
      />,
      { wrapper }
    )

    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('should not render unread count badge when no unread items', () => {
    const wrapper = createWrapper()
    const feedWithoutUnread = { ...mockFeed, unread_count: 0 }
    render(
      <FeedCard
        feed={feedWithoutUnread}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        onClick={vi.fn()}
      />,
      { wrapper }
    )

    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('should call onClick when card is clicked', () => {
    const wrapper = createWrapper()
    const handleClick = vi.fn()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        onClick={handleClick}
      />,
      { wrapper }
    )

    const card = screen.getByText('Example Feed').closest('div')
    fireEvent.click(card!)

    expect(handleClick).toHaveBeenCalledWith('feed-1')
  })

  it('should call onRefresh when refresh button is clicked', () => {
    const wrapper = createWrapper()
    const handleRefresh = vi.fn()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={vi.fn()}
        onRefresh={handleRefresh}
        onClick={vi.fn()}
      />,
      { wrapper }
    )

    const refreshBtn = screen.getByTestId('refresh-icon').closest('button')
    fireEvent.click(refreshBtn!)

    expect(handleRefresh).toHaveBeenCalledWith('feed-1')
  })

  it('should call onDelete when delete button is clicked', () => {
    const wrapper = createWrapper()
    const handleDelete = vi.fn()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={handleDelete}
        onRefresh={vi.fn()}
        onClick={vi.fn()}
      />,
      { wrapper }
    )

    const deleteBtn = screen.getByTestId('trash-icon').closest('button')
    fireEvent.click(deleteBtn!)

    expect(handleDelete).toHaveBeenCalledWith('feed-1')
  })

  it('should stop propagation when action buttons are clicked', () => {
    const wrapper = createWrapper()
    const handleClick = vi.fn()
    const handleDelete = vi.fn()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={handleDelete}
        onRefresh={vi.fn()}
        onClick={handleClick}
      />,
      { wrapper }
    )

    const deleteBtn = screen.getByTestId('trash-icon').closest('button')
    fireEvent.click(deleteBtn!)

    expect(handleClick).not.toHaveBeenCalled()
  })

  it('should show loading state when isRefreshing is true', () => {
    const wrapper = createWrapper()
    render(
      <FeedCard
        feed={mockFeed}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        onClick={vi.fn()}
        isRefreshing
      />,
      { wrapper }
    )

    const refreshBtn = screen.getByTestId('refresh-icon').closest('button')
    expect(refreshBtn).toBeDisabled()
  })
})
