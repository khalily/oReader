import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ArticlePanel } from '../ArticlePanel'
import type { Article } from '@/types/feed'
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
  Calendar: () => <div data-testid="calendar-icon" />,
  User: () => <div data-testid="user-icon" />,
  Rss: () => <div data-testid="rss-icon" />,
  ArrowLeft: () => <div data-testid="arrow-left-icon" />,
  ChevronDown: () => <div data-testid="chevron-down-icon" />,
  ChevronUp: () => <div data-testid="chevron-up-icon" />,
  Check: () => <div data-testid="check-icon" />,
  Copy: () => <div data-testid="copy-icon" />,
}))

const API_BASE_URL = '/api/v1'

const mockArticle: Article = {
  id: 'item-1',
  feed_id: 'feed-1',
  guid: 'guid-1',
  title: 'Test Article Title',
  link: 'https://example.com/article1',
  description: 'This is a brief description of the article',
  content: 'This is the full article content.\n\nIt contains multiple paragraphs.',
  pub_date: '2024-01-15T10:30:00Z',
  creator: 'John Doe',
  created_at: '2024-01-15T10:30:00Z',
  feed: {
    id: 'feed-1',
    title: 'Test Feed',
    feed_url: 'https://example.com/feed.xml',
    description: 'A test RSS feed',
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
}

const mockReadArticle: Article = {
  ...mockArticle,
  user_state: {
    item_id: 'item-1',
    is_read: true,
    is_starred: false,
    read_at: '2024-01-15T10:30:00Z',
  },
}

const mockStarredArticle: Article = {
  ...mockArticle,
  user_state: {
    item_id: 'item-1',
    is_read: false,
    is_starred: true,
    read_at: null,
  },
}

let starRequestBody: { starred: boolean } | null = null
let readRequestBody: { read: boolean } | null = null

const handlers = [
  http.get(`${API_BASE_URL}/items/:id`, ({ params }) => {
    if (params.id === 'item-1') {
      return HttpResponse.json({ item: mockArticle })
    }
    if (params.id === 'item-read') {
      return HttpResponse.json({ item: mockReadArticle })
    }
    if (params.id === 'item-starred') {
      return HttpResponse.json({ item: mockStarredArticle })
    }
    return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
  }),
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ request }) => {
    const body = await request.json() as { starred: boolean }
    starRequestBody = body
    return HttpResponse.json({
      item: {
        ...mockArticle,
        user_state: {
          ...mockArticle.user_state!,
          is_starred: body.starred,
        },
      },
    })
  }),
  http.put(`${API_BASE_URL}/items/:id/read`, async ({ request }) => {
    const body = await request.json() as { read: boolean }
    readRequestBody = body
    return HttpResponse.json({
      item: {
        ...mockArticle,
        user_state: {
          ...mockArticle.user_state!,
          is_read: body.read,
          read_at: body.read ? new Date().toISOString() : null,
        },
      },
    })
  }),
  http.get(`${API_BASE_URL}/items/error-id`, () => {
    return HttpResponse.json({ error: { code: 'INTERNAL_ERROR', message: 'Internal server error' } }, { status: 500 })
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

describe('ArticlePanel component', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  describe('Empty state', () => {
    it('should show empty state when itemId is null', () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId={null} />, { wrapper })

      expect(screen.getByText('Select an article to read')).toBeInTheDocument()
      expect(screen.getByTestId('rss-icon')).toBeInTheDocument()
      expect(screen.getByText(/Choose an article from the list/)).toBeInTheDocument()
    })
  })

  describe('Loading state', () => {
    it('should show loading spinner while fetching article', () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      expect(screen.getByTestId('loader-icon')).toBeInTheDocument()
    })
  })

  describe('Error state', () => {
    it('should show error message when fetch fails', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-error" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText(/Unable to load article/)).toBeInTheDocument()
      })
    })

    it('should show retry button in error state', async () => {
      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-error" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Try Again/i })).toBeInTheDocument()
      })
    })

    it('should refetch when retry button is clicked', async () => {
      let fetchCount = 0
      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          fetchCount++
          if (fetchCount === 1) {
            return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'Item not found' } }, { status: 404 })
          }
          return HttpResponse.json({ item: mockArticle })
        })
      )

      const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false } },
      })

      const queryWrapper = ({ children }: { children: React.ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      )

      render(<ArticlePanel itemId="item-retry" />, { wrapper: queryWrapper })

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Try Again/i })).toBeInTheDocument()
      })

      fireEvent.click(screen.getByRole('button', { name: /Try Again/i }))

      await waitFor(() => {
        expect(fetchCount).toBe(2)
      })
    })
  })

  describe('Content display', () => {
    it('should display article title', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })
    })

    it('should display feed name', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Test Feed')).toBeInTheDocument()
      })
    })

    it('should display creator when available', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument()
        // Find the creator section which contains the user icon
        const creatorText = screen.getByText('John Doe').parentElement
        expect(creatorText?.querySelector('[data-testid="user-icon"]')).toBeInTheDocument()
      })
    })

    it('should display publication date', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        // Date format should include month name, day, year
        const dateText = screen.getByText(/January.*15.*2024/i)
        expect(dateText).toBeInTheDocument()
      })
    })

    it('should display article content', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('This is the full article content.')).toBeInTheDocument()
      })
    })

    it('should show "No content available" message when content is null', async () => {
      const articleWithoutContent: Article = {
        ...mockArticle,
        content: null,
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithoutContent })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-no-content" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText(/No content available/i)).toBeInTheDocument()
        expect(screen.getByText(/Click "Open" to read the full article/i)).toBeInTheDocument()
      })
    })
  })

  describe('Summary section', () => {
    it('should show summary section when description differs from content', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Summary')).toBeInTheDocument()
        expect(screen.getByText('This is a brief description of the article')).toBeInTheDocument()
      })
    })

    it('should collapse/expand summary on click', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        const summaryButton = screen.getByText('Summary').closest('button')
        expect(summaryButton).toBeInTheDocument()
        expect(screen.getByTestId('chevron-up-icon')).toBeInTheDocument()
      })

      // Click to collapse
      const summaryButton = screen.getByText('Summary').closest('button')
      fireEvent.click(summaryButton!)

      await waitFor(() => {
        expect(screen.getByTestId('chevron-down-icon')).toBeInTheDocument()
        // Description should no longer be visible
        expect(screen.queryByText('This is a brief description of the article')).not.toBeInTheDocument()
      })

      // Click to expand
      fireEvent.click(summaryButton!)

      await waitFor(() => {
        expect(screen.getByTestId('chevron-up-icon')).toBeInTheDocument()
        expect(screen.getByText('This is a brief description of the article')).toBeInTheDocument()
      })
    })

    it('should not show summary when description equals content', async () => {
      const articleWithSameContent: Article = {
        ...mockArticle,
        description: 'This is the full article content.\n\nIt contains multiple paragraphs.',
        content: 'This is the full article content.\n\nIt contains multiple paragraphs.',
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithSameContent })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-same-content" />, { wrapper })

      await waitFor(() => {
        expect(screen.queryByText('Summary')).not.toBeInTheDocument()
      })
    })
  })

  describe('Action buttons', () => {
    it('should show read/unread button', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-read" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Read/i })).toBeInTheDocument()
        // Check that the button contains the eye icon
        const readButton = screen.getByRole('button', { name: /Read/i })
        expect(readButton.querySelector('[data-testid="eye-icon"]')).toBeInTheDocument()
      })
    })

    it('should show unread button for unread article', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Unread/i })).toBeInTheDocument()
        // Check that the button contains the eye-off icon
        const unreadButton = screen.getByRole('button', { name: /Unread/i })
        expect(unreadButton.querySelector('[data-testid="eye-off-icon"]')).toBeInTheDocument()
      })
    })

    it('should show star button and call API on click', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Star')).toBeInTheDocument()
        // Check that the button contains the star-off icon
        const starButton = screen.getByText('Star').closest('button')
        expect(starButton?.querySelector('[data-testid="star-off-icon"]')).toBeInTheDocument()
      })

      const starButton = screen.getByText('Star').closest('button')
      fireEvent.click(starButton!)

      await waitFor(() => {
        expect(starRequestBody).toEqual({ starred: true })
      })
    })

    it('should show unstar button for starred article', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-starred" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Starred')).toBeInTheDocument()
        // Check that the button contains the star icon
        const starButton = screen.getByText('Starred').closest('button')
        expect(starButton?.querySelector('[data-testid="star-icon"]')).toBeInTheDocument()
      })
    })

    it('should show external link button when link exists', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Open')).toBeInTheDocument()
        // Check that the button contains the external link icon
        const openButton = screen.getByText('Open').closest('button')
        expect(openButton?.querySelector('[data-testid="external-link-icon"]')).toBeInTheDocument()

        const link = screen.getByText('Open').closest('a')
        expect(link).toHaveAttribute('href', 'https://example.com/article1')
        expect(link).toHaveAttribute('target', '_blank')
        expect(link).toHaveAttribute('rel', 'noopener noreferrer')
      })
    })

    it('should not show external link button when link is null', async () => {
      const articleWithoutLink: Article = {
        ...mockArticle,
        link: null,
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithoutLink })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-no-link" />, { wrapper })

      await waitFor(() => {
        expect(screen.queryByText('Open')).not.toBeInTheDocument()
      })
    })

    it('should toggle read status when read button is clicked', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-read" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Read/i })).toBeInTheDocument()
      })

      const readButton = screen.getByRole('button', { name: /Read/i })
      fireEvent.click(readButton)

      await waitFor(() => {
        expect(readRequestBody).toEqual({ read: false })
      })
    })
  })

  describe('Mobile back button', () => {
    it('should not show back button when showBackButton is false', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" showBackButton={false} />, { wrapper })

      await waitFor(() => {
        expect(screen.queryByText('Back')).not.toBeInTheDocument()
        const arrowIcons = screen.queryAllByTestId('arrow-left-icon')
        expect(arrowIcons.length).toBe(0)
      })
    })

    it('should show back button when showBackButton is true', async () => {
      const wrapper = createWrapper()
      const onClose = vi.fn()
      render(<ArticlePanel itemId="item-1" showBackButton={true} onClose={onClose} />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Back')).toBeInTheDocument()
        const backButton = screen.getByText('Back').closest('button')
        expect(backButton?.querySelector('[data-testid="arrow-left-icon"]')).toBeInTheDocument()
      })
    })

    it('should call onClose when back button is clicked', async () => {
      const wrapper = createWrapper()
      const onClose = vi.fn()
      render(<ArticlePanel itemId="item-1" showBackButton={true} onClose={onClose} />, { wrapper })

      await waitFor(() => {
        const backButton = screen.getByText('Back')
        expect(backButton).toBeInTheDocument()
      })

      const backButton = screen.getByText('Back')
      fireEvent.click(backButton)

      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('Auto-mark as read', () => {
    it('should auto-mark unread article as read when loaded', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        // Wait for article to load
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })

      // Wait for auto-mark read to complete
      await waitFor(
        () => {
          expect(readRequestBody).toEqual({ read: true })
        },
        { timeout: 3000 }
      )
    })

    it('should not auto-mark already read article', async () => {
      readRequestBody = null
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-read" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })

      // Give it some time but expect no API call
      await waitFor(
        () => {
          expect(readRequestBody).toBeNull()
        },
        { timeout: 1000 }
      )
    })

    it('should only trigger auto-mark read API once (no duplicate requests)', async () => {
      // Track the number of times the read API is called
      let readApiCallCount = 0

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: mockArticle })
        }),
        http.put(`${API_BASE_URL}/items/:id/read`, async ({ request }) => {
          readApiCallCount++
          const body = await request.json() as { read: boolean }
          return HttpResponse.json({
            item: {
              ...mockArticle,
              user_state: {
                ...mockArticle.user_state!,
                is_read: body.read,
                read_at: body.read ? new Date().toISOString() : null,
              },
            },
          })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      // Wait for article to load and auto-mark to complete
      await waitFor(() => {
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })

      // Wait for the API call to be made
      await waitFor(
        () => {
          expect(readApiCallCount).toBeGreaterThanOrEqual(1)
        },
        { timeout: 3000 }
      )

      // Wait additional time to ensure no duplicate calls
      await new Promise((resolve) => setTimeout(resolve, 500))

      // Should have exactly 1 call, not multiple
      expect(readApiCallCount).toBe(1)
    })
  })

  describe('Badges', () => {
    it('should show Read badge for read articles', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-read" />, { wrapper })

      await waitFor(() => {
        // Badge appears in the article header with text "Read"
        const badges = screen.getAllByText('Read')
        expect(badges.length).toBeGreaterThan(0)
        // Check that there are at least 2 eye icons (one in button, one in badge)
        const eyeIcons = screen.getAllByTestId('eye-icon')
        expect(eyeIcons.length).toBe(2)
      })
    })

    it('should show feed badge', async () => {
      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-1" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Test Feed')).toBeInTheDocument()
        // The feed badge contains the rss icon
        const feedBadge = screen.getByText('Test Feed').closest('.rounded-full')
        expect(feedBadge?.querySelector('[data-testid="rss-icon"]')).toBeInTheDocument()
      })
    })
  })

  describe('Edge cases', () => {
    it('should handle missing creator', async () => {
      const articleWithoutCreator: Article = {
        ...mockArticle,
        creator: null,
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithoutCreator })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-no-creator" />, { wrapper })

      await waitFor(() => {
        // No creator means no user icon in the metadata section
        const userIcons = screen.queryAllByTestId('user-icon')
        expect(userIcons.length).toBe(0)
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })
    })

    it('should handle missing pub_date', async () => {
      const articleWithoutDate: Article = {
        ...mockArticle,
        pub_date: null,
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithoutDate })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-no-date" />, { wrapper })

      await waitFor(() => {
        // No pub_date means no calendar icon in the metadata section
        const calendarIcons = screen.queryAllByTestId('calendar-icon')
        expect(calendarIcons.length).toBe(0)
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
      })
    })

    it('should handle missing user_state', async () => {
      const articleWithoutUserState: Article = {
        ...mockArticle,
        user_state: null,
      }

      server.use(
        http.get(`${API_BASE_URL}/items/:id`, () => {
          return HttpResponse.json({ item: articleWithoutUserState })
        })
      )

      const wrapper = createWrapper()
      render(<ArticlePanel itemId="item-no-state" />, { wrapper })

      await waitFor(() => {
        expect(screen.getByText('Test Article Title')).toBeInTheDocument()
        // Should treat as unread
        expect(screen.getByText('Unread')).toBeInTheDocument()
        expect(screen.getByText('Star')).toBeInTheDocument()
      })
    })
  })
})
