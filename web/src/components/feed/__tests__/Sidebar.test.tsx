import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar } from '../Sidebar'
import type { UserFeed, StatsResponse } from '@/types/feed'

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
  Calendar: () => <div data-testid="calendar-icon" />,
  BookOpen: () => <div data-testid="bookopen-icon" />,
}))

// Mock stats data
const mockStats: StatsResponse = {
  total: 100,
  unread: 25,
  starred: 10,
  today: 5,
}

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
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </MemoryRouter>
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

    expect(screen.getByText('All')).toBeInTheDocument()
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

  describe('Today filter', () => {
    it('should render Today filter button', () => {
      const wrapper = createWrapper()
      const handleFilterChange = vi.fn()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={handleFilterChange}
        />,
        { wrapper }
      )

      expect(screen.getByText('Today')).toBeInTheDocument()
      expect(screen.getByTestId('calendar-icon')).toBeInTheDocument()
    })

    it('should highlight Today filter when active', () => {
      const wrapper = createWrapper()
      const handleFilterChange = vi.fn()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="today"
          onFilterChange={handleFilterChange}
        />,
        { wrapper }
      )

      const todayButton = screen.getByText('Today').closest('button')
      expect(todayButton).toHaveClass('bg-accent')
    })

    it('should call onFilterChange with "today" when Today is clicked', () => {
      const wrapper = createWrapper()
      const handleFilterChange = vi.fn()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={handleFilterChange}
        />,
        { wrapper }
      )

      fireEvent.click(screen.getByText('Today'))
      expect(handleFilterChange).toHaveBeenCalledWith('today')
    })
  })

  describe('Stats display', () => {
    it('should display stats counts when stats prop is provided', () => {
      const wrapper = createWrapper()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={vi.fn()}
          stats={mockStats}
        />,
        { wrapper }
      )

      // Check that all filter buttons are rendered with stats
      expect(screen.getByText('All')).toBeInTheDocument()
      expect(screen.getByText('Unread')).toBeInTheDocument()
      expect(screen.getByText('Starred')).toBeInTheDocument()
      expect(screen.getByText('Today')).toBeInTheDocument()
    })

    it('should display total count for All filter', () => {
      const wrapper = createWrapper()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={vi.fn()}
          stats={mockStats}
        />,
        { wrapper }
      )

      const allButton = screen.getByText('All').closest('button')
      expect(allButton?.textContent).toContain('100')
    })

    it('should display unread count for Unread filter', () => {
      const wrapper = createWrapper()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={vi.fn()}
          stats={mockStats}
        />,
        { wrapper }
      )

      const unreadButton = screen.getByText('Unread').closest('button')
      expect(unreadButton?.textContent).toContain('25')
    })

    it('should display starred count for Starred filter', () => {
      const wrapper = createWrapper()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={vi.fn()}
          stats={mockStats}
        />,
        { wrapper }
      )

      const starredButton = screen.getByText('Starred').closest('button')
      expect(starredButton?.textContent).toContain('10')
    })

    it('should display today count for Today filter', () => {
      const wrapper = createWrapper()

      render(
        <Sidebar
          feeds={mockFeeds}
          selectedFeedId={null}
          onFeedClick={vi.fn()}
          onAddFeed={vi.fn()}
          onDeleteFeed={vi.fn()}
          onRefreshFeed={vi.fn()}
          refreshingFeedIds={new Set()}
          filterType="all"
          onFilterChange={vi.fn()}
          stats={mockStats}
        />,
        { wrapper }
      )

      const todayButton = screen.getByText('Today').closest('button')
      expect(todayButton?.textContent).toContain('5')
    })
  })
})
