import { describe, it, expect, vi, beforeEach, afterEach, afterAll } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import ItemViewPage from '../ItemViewPage'
import type { Article } from '@/types/feed'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock lucide-react icons
vi.mock('lucide-react', () => ({
  Star: () => <div data-testid="star-icon" />,
  StarOff: () => <div data-testid="star-off-icon" />,
  Eye: () => <div data-testid="eye-icon" />,
  EyeOff: () => <div data-testid="eye-off-icon" />,
  ArrowLeft: () => <div data-testid="arrow-left-icon" />,
  ExternalLink: () => <div data-testid="external-link-icon" />,
  Loader2: () => <div data-testid="loader-icon" />,
  Calendar: () => <div data-testid="calendar-icon" />,
  User: () => <div data-testid="user-icon" />,
  Rss: () => <div data-testid="rss-icon" />,
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
  description: 'A brief description of the article',
  content: 'This is the full article content with **Markdown formatting**.\n\nIt has multiple paragraphs.',
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
}

const handlers = [
  http.get(`${API_BASE_URL}/items/:id`, ({ params }) => {
    if (params.id === 'item-1') {
      return HttpResponse.json({ item: mockArticle })
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Article not found' } },
      { status: 404 }
    )
  }),
  http.put(`${API_BASE_URL}/items/:id/star`, async ({ request, params }) => {
    const body = await request.json() as { starred: boolean }
    if (params.id === 'item-1') {
      return HttpResponse.json({
        item: {
          ...mockArticle,
          user_state: {
            ...mockArticle.user_state!,
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
  http.put(`${API_BASE_URL}/items/:id/read`, async ({ request, params }) => {
    const body = await request.json() as { read: boolean }
    if (params.id === 'item-1') {
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
    }
    return HttpResponse.json(
      { error: { code: 'NOT_FOUND', message: 'Item not found' } },
      { status: 404 }
    )
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

const renderWithRouter = (ui: React.ReactElement, initialEntries = ['/items/item-1']) => {
  const wrapper = createWrapper()
  return {
    ...render(
      <MemoryRouter initialEntries={initialEntries}>
        <Routes>
          <Route path="/items/:id" element={ui} />
          <Route path="/items" element={<div data-testid="items-page">Items Page</div>} />
        </Routes>
      </MemoryRouter>,
      { wrapper }
    ),
  }
}

describe('ItemViewPage', () => {
  beforeEach(() => {
    server.listen()
  })

  afterEach(() => {
    server.resetHandlers()
  })

  afterAll(() => {
    server.close()
  })

  it('should render article content', async () => {
    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    expect(screen.getByText(/A brief description of the article/)).toBeInTheDocument()
    expect(screen.getByText(/This is the full article content/)).toBeInTheDocument()
  })

  it('should show loading state initially', () => {
    server.use(
      http.get(`${API_BASE_URL}/items/:id`, () => {
        return new Promise(() => {}) // Never resolve
      })
    )

    renderWithRouter(<ItemViewPage />)

    expect(screen.getByTestId('loader-icon')).toBeInTheDocument()
  })

  it('should show error state on 404', async () => {
    server.use(
      http.get(`${API_BASE_URL}/items/:id`, () => {
        return HttpResponse.json(
          { error: { code: 'NOT_FOUND', message: 'Article not found' } },
          { status: 404 }
        )
      })
    )

    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText(/article not found/i)).toBeInTheDocument()
    })
  })

  it('should display article metadata', async () => {
    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    // Check feed name is displayed
    expect(screen.getByText('Example Feed')).toBeInTheDocument()

    // Check author is displayed
    expect(screen.getByText('John Doe')).toBeInTheDocument()
  })

  it('should navigate back when back button is clicked', async () => {
    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    const backBtn = screen.getByTestId('arrow-left-icon').closest('button')
    fireEvent.click(backBtn!)

    await waitFor(() => {
      expect(screen.getByTestId('items-page')).toBeInTheDocument()
    })
  })

  it('should toggle star status', async () => {
    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    const starBtn = screen.getByTestId('star-off-icon').closest('button')
    fireEvent.click(starBtn!)

    await waitFor(() => {
      expect(screen.getByTestId('star-icon')).toBeInTheDocument()
    })
  })

  it('should toggle read status', async () => {
    // Start with an already-read article to test the toggle
    const readArticle: Article = {
      ...mockArticle,
      user_state: {
        item_id: 'item-1',
        is_read: true,
        is_starred: false,
        read_at: '2024-01-15T10:00:00Z',
      },
    }

    server.use(
      http.get(`${API_BASE_URL}/items/:id`, () => {
        return HttpResponse.json({ item: readArticle })
      })
    )

    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    // Article starts as read, so we should see the "Read" button
    const readBtn = screen.getByRole('button', { name: /read/i })
    expect(readBtn).toBeInTheDocument()

    fireEvent.click(readBtn)

    await waitFor(() => {
      // After toggling, should show eye-off-icon
      expect(screen.getByTestId('eye-off-icon')).toBeInTheDocument()
    })
  })

  it('should render Markdown content safely', async () => {
    const articleWithMarkdown: Article = {
      ...mockArticle,
      content: 'Safe **Markdown** content\n\n```javascript\nconsole.log("code")\n```',
    }

    server.use(
      http.get(`${API_BASE_URL}/items/:id`, () => {
        return HttpResponse.json({ item: articleWithMarkdown })
      })
    )

    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      // Markdown **Markdown** becomes <strong>Markdown</strong>, so we match "Safe" and "content"
      expect(screen.getByText(/Safe/)).toBeInTheDocument()
      expect(screen.getByText('Markdown')).toBeInTheDocument()
      expect(screen.getByText(/content/)).toBeInTheDocument()
    })

    // MarkdownRenderer should render the content without executing any code
    // ReactMarkdown handles sanitization automatically
    const scriptElements = document.querySelectorAll('script')
    expect(scriptElements.length).toBe(0)
  })

  it('should open article in new tab when external link is clicked', async () => {
    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    const externalLink = screen.getByRole('link', { name: /open article in new tab/i })
    expect(externalLink).toHaveAttribute('href', 'https://example.com/article1')
    expect(externalLink).toHaveAttribute('target', '_blank')
    expect(externalLink).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should handle missing user state', async () => {
    const articleWithoutState: Article = {
      ...mockArticle,
      user_state: null,
    }

    server.use(
      http.get(`${API_BASE_URL}/items/:id`, () => {
        return HttpResponse.json({ item: articleWithoutState })
      })
    )

    renderWithRouter(<ItemViewPage />)

    await waitFor(() => {
      expect(screen.getByText('Test Article Title')).toBeInTheDocument()
    })

    // Should show star-off and eye-off icons (default state)
    expect(screen.getByTestId('star-off-icon')).toBeInTheDocument()
    expect(screen.getByTestId('eye-off-icon')).toBeInTheDocument()
  })
})
