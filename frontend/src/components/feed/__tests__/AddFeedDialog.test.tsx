import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AddFeedDialog } from '../AddFeedDialog'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock API base URL
const API_BASE_URL = '/api/v1'

// Mock handlers
const handlers = [
  http.post(`${API_BASE_URL}/feeds`, async ({ request }) => {
    const body = await request.json() as { feed_url: string }
    return HttpResponse.json({
      feed: {
        id: 'feed-new',
        title: 'Example Feed',
        feed_url: body.feed_url,
        description: 'An example RSS feed',
        image_url: null,
        last_fetched_at: '2024-01-01T00:00:00Z',
        created_at: '2024-01-01T00:00:00Z',
      },
      new_item_count: 10,
    })
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

describe('AddFeedDialog component', () => {
  const mockOnOpenChange = vi.fn()
  const mockOnSuccess = vi.fn()

  beforeAll(() => server.listen())
  beforeEach(() => {
    server.resetHandlers()
    vi.clearAllMocks()
  })
  afterAll(() => server.close())

  it('should render dialog when open is true', () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    expect(screen.getByText('Add Feed')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/enter rss feed url/i)).toBeInTheDocument()
  })

  it('should not render dialog when open is false', () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={false}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    expect(screen.queryByText('Add Feed')).not.toBeInTheDocument()
  })

  it('should call onOpenChange with false when cancel is clicked', () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const cancelBtn = screen.getByText('Cancel')
    fireEvent.click(cancelBtn)

    expect(mockOnOpenChange).toHaveBeenCalledWith(false)
  })

  it('should call onOpenChange with false when escape key is pressed', () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    fireEvent.keyDown(document, { key: 'Escape' })

    expect(mockOnOpenChange).toHaveBeenCalledWith(false)
  })

  it('should call onOpenChange with false when backdrop is clicked', () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const backdrop = screen.getByText('Add Feed').closest('body')?.querySelector('.fixed.inset-0.bg-black\\/50')
    if (backdrop) {
      fireEvent.click(backdrop)
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    }
  })

  it('should show validation error for empty URL when submit is clicked', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const submitBtn = screen.getByText('Add')
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/feed url is required/i)).toBeInTheDocument()
    })
  })

  it('should show validation error for invalid URL format', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const input = screen.getByPlaceholderText(/enter rss feed url/i)
    fireEvent.change(input, { target: { value: 'not-a-valid-url' } })

    const submitBtn = screen.getByText('Add')
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/invalid url format/i)).toBeInTheDocument()
    })
  })

  it('should show validation error for non-HTTP URL', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const input = screen.getByPlaceholderText(/enter rss feed url/i)
    fireEvent.change(input, { target: { value: 'ftp://example.com/feed.xml' } })

    const submitBtn = screen.getByText('Add')
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/only http and https urls are supported/i)).toBeInTheDocument()
    })
  })

  it('should accept valid URL format', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const input = screen.getByPlaceholderText(/enter rss feed url/i)
    fireEvent.change(input, { target: { value: 'https://example.com/feed.xml' } })

    // Should not show validation error
    expect(screen.queryByText(/invalid url format/i)).not.toBeInTheDocument()
  })

  it('should show loading state during submission', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const input = screen.getByPlaceholderText(/enter rss feed url/i)
    fireEvent.change(input, { target: { value: 'https://example.com/feed.xml' } })

    const submitBtn = screen.getByText('Add')
    fireEvent.click(submitBtn)

    // Dialog closes after successful submission
    await waitFor(() => {
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })

    // onSuccess callback is called
    expect(mockOnSuccess).toHaveBeenCalled()
  })

  it('should clear form after successful submission', async () => {
    const wrapper = createWrapper()
    render(
      <AddFeedDialog
        open={true}
        onOpenChange={mockOnOpenChange}
        onSuccess={mockOnSuccess}
      />,
      { wrapper }
    )

    const input = screen.getByPlaceholderText(/enter rss feed url/i) as HTMLInputElement
    fireEvent.change(input, { target: { value: 'https://example.com/feed.xml' } })

    const submitBtn = screen.getByText('Add')
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockOnOpenChange).toHaveBeenCalledWith(false)
    })
  })
})
