import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Sidebar } from '../Sidebar'
import type { UserFeed } from '@/types/feed'

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Plus: () => <div data-testid="plus-icon" />,
  Home: () => <div data-testid="home-icon" />,
  Star: () => <div data-testid="star-icon" />,
  Rss: () => <div data-testid="rss-icon" />,
  Menu: () => <div data-testid="menu-icon" />,
  X: () => <div data-testid="x-icon" />,
  RefreshCw: () => <div data-testid="refresh-icon" />,
  Trash2: () => <div data-testid="trash-icon" />,
}))

const mockFeeds: UserFeed[] = [
  {
    id: 'feed-1',
    title: 'Example Feed',
    feed_url: 'https://example.com/feed.xml',
    description: 'An example RSS feed',
    image_url: null,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    unread_count: 5,
    position: 1,
  },
]

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

describe('Sidebar component', () => {
  it('should render navigation items', () => {
    const wrapper = createWrapper()
    const handleFeedClick = vi.fn()
    const handleAddFeed = vi.fn()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()

    render(
      <Sidebar
        feeds={mockFeeds}
        selectedFeedId={null}
        onFeedClick={handleFeedClick}
        onAddFeed={handleAddFeed}
        onDeleteFeed={handleDelete}
        onRefreshFeed={handleRefresh}
        refreshingFeedIds={new Set()}
        filterType="all"
        onFilterChange={vi.fn()}
      />,
      { wrapper }
    )

    expect(screen.getByText('All Items')).toBeInTheDocument()
    expect(screen.getByText('Unread')).toBeInTheDocument()
    expect(screen.getByText('Starred')).toBeInTheDocument()
  })

  it('should render feeds list', () => {
    const wrapper = createWrapper()
    const handleFeedClick = vi.fn()
    const handleAddFeed = vi.fn()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()

    render(
      <Sidebar
        feeds={mockFeeds}
        selectedFeedId={null}
        onFeedClick={handleFeedClick}
        onAddFeed={handleAddFeed}
        onDeleteFeed={handleDelete}
        onRefreshFeed={handleRefresh}
        refreshingFeedIds={new Set()}
        filterType="all"
        onFilterChange={vi.fn()}
      />,
      { wrapper }
    )

    expect(screen.getByText('Example Feed')).toBeInTheDocument()
  })

  it('should call onFilterChange when filter is clicked', () => {
    const wrapper = createWrapper()
    const handleFeedClick = vi.fn()
    const handleAddFeed = vi.fn()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()
    const handleFilterChange = vi.fn()

    render(
      <Sidebar
        feeds={mockFeeds}
        selectedFeedId={null}
        onFeedClick={handleFeedClick}
        onAddFeed={handleAddFeed}
        onDeleteFeed={handleDelete}
        onRefreshFeed={handleRefresh}
        refreshingFeedIds={new Set()}
        filterType="all"
        onFilterChange={handleFilterChange}
      />,
      { wrapper }
    )

    fireEvent.click(screen.getByText('Unread'))
    expect(handleFilterChange).toHaveBeenCalledWith('unread')
  })

  it('should highlight active filter', () => {
    const wrapper = createWrapper()
    const handleFeedClick = vi.fn()
    const handleAddFeed = vi.fn()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()

    render(
      <Sidebar
        feeds={mockFeeds}
        selectedFeedId={null}
        onFeedClick={handleFeedClick}
        onAddFeed={handleAddFeed}
        onDeleteFeed={handleDelete}
        onRefreshFeed={handleRefresh}
        refreshingFeedIds={new Set()}
        filterType="unread"
        onFilterChange={vi.fn()}
      />,
      { wrapper }
    )

    const unreadButton = screen.getByText('Unread').closest('button')
    expect(unreadButton).toHaveClass('bg-accent')
  })

  it('should show unread count badge', () => {
    const wrapper = createWrapper()
    const handleFeedClick = vi.fn()
    const handleAddFeed = vi.fn()
    const handleDelete = vi.fn()
    const handleRefresh = vi.fn()

    render(
      <Sidebar
        feeds={mockFeeds}
        selectedFeedId={null}
        onFeedClick={handleFeedClick}
        onAddFeed={handleAddFeed}
        onDeleteFeed={handleDelete}
        onRefreshFeed={handleRefresh}
        refreshingFeedIds={new Set()}
        filterType="all"
        onFilterChange={vi.fn()}
        totalUnread={5}
      />,
      { wrapper }
    )

    // Should show unread count in the Unread filter button
    const unreadButton = screen.getByText('Unread').closest('button')
    expect(unreadButton?.textContent).toContain('5')

    // Should also show unread count in feed card
    expect(screen.getByText('Example Feed')).toBeInTheDocument()
  })
})
