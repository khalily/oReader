import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { NewSidebar } from '../Sidebar'
import type { UserFeed } from '@/types/feed'
import type { Category } from '@/types/category'
import type { Paper } from '@/types/paper'

// --- Mock stores ---

const mockSidebarState = {
  selectedFeedId: null as string | null,
  selectedPaperId: null as string | null,
  selectionType: null as 'feed' | 'paper' | null,
  expandedCategoryIds: new Set<string>(),
  selectFeed: vi.fn(),
  selectPaper: vi.fn(),
  clearSelection: vi.fn(),
  toggleCategory: vi.fn(),
  expandCategory: vi.fn(),
}

const mockAuthState = {
  isAuthenticated: true,
  user: {
    id: 'user-1',
    email: 'test@example.com',
    nickname: 'Test',
    avatar_url: null,
    auth_provider: 'email' as const,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  csrfToken: null,
  setUser: vi.fn(),
  clearUser: vi.fn(),
  initializeFromStorage: vi.fn(),
  getCsrfToken: vi.fn(),
}

vi.mock('@/stores/sidebarStore', () => ({
  useSidebarStore: (selector?: (state: typeof mockSidebarState) => unknown) =>
    selector ? selector(mockSidebarState) : mockSidebarState,
}))

vi.mock('@/stores/authStore', () => ({
  useAuthStore: (selector?: (state: typeof mockAuthState) => unknown) =>
    selector ? selector(mockAuthState) : mockAuthState,
}))

// --- Mock lucide-react icons ---
// Must include icons used by both the new sidebar and legacy exports in the same file.
vi.mock('lucide-react', () => ({
  Plus: () => <div data-testid="icon-plus" />,
  Rss: () => <div data-testid="icon-rss" />,
  FileText: () => <div data-testid="icon-filetext" />,
  ChevronRight: () => <div data-testid="icon-chevron-right" />,
  ChevronDown: () => <div data-testid="icon-chevron-down" />,
  MoreHorizontal: () => <div data-testid="icon-more" />,
  Pencil: () => <div data-testid="icon-pencil" />,
  Trash2: () => <div data-testid="icon-trash" />,
  FolderInput: () => <div data-testid="icon-folder-input" />,
  Home: () => <div data-testid="icon-home" />,
  Star: () => <div data-testid="icon-star" />,
  Calendar: () => <div data-testid="icon-calendar" />,
  BookOpen: () => <div data-testid="icon-bookopen" />,
  Upload: () => <div data-testid="icon-upload" />,
  Download: () => <div data-testid="icon-download" />,
  Menu: () => <div data-testid="icon-menu" />,
  FolderPlus: () => <div data-testid="icon-folder-plus" />,
}))

// --- Mock data ---

const mockFeedCategories: Category[] = [
  {
    id: 'cat-feed-1',
    user_id: 'user-1',
    name: 'Tech',
    type: 'feed',
    position: 1,
    created_at: '2024-01-01T00:00:00Z',
  },
]

const mockPaperCategories: Category[] = [
  {
    id: 'cat-paper-1',
    user_id: 'user-1',
    name: 'ML Research',
    type: 'paper',
    position: 1,
    created_at: '2024-01-01T00:00:00Z',
  },
]

const mockFeeds: UserFeed[] = [
  {
    id: 'feed-1',
    title: 'Hacker News',
    feed_url: 'https://news.ycombinator.com/rss',
    description: 'Tech news',
    image_url: null,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    unread_count: 5,
    item_count: 100,
    position: 1,
    category_id: 'cat-feed-1',
  },
  {
    id: 'feed-2',
    title: 'Uncategorized Feed',
    feed_url: 'https://example.com/rss',
    description: 'No category',
    image_url: null,
    last_fetched_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    unread_count: 3,
    item_count: 50,
    position: 2,
    category_id: null,
  },
]

const mockPapers: Paper[] = [
  {
    id: 'paper-1',
    user_id: 'user-1',
    title: 'Attention Is All You Need',
    authors: '["Vaswani et al."]',
    abstract: 'A transformer paper',
    keywords: '["transformer", "attention"]',
    published_year: '2017',
    doi: null,
    pdf_size: 1234,
    markdown_content: null,
    cover_image: null,
    original_filename: 'attention.pdf',
    status: 'completed',
    error: null,
    category_id: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
]

// --- Helpers ---

const defaultCallbacks = {
  onAddClick: vi.fn(),
  onFeedClick: vi.fn(),
  onPaperClick: vi.fn(),
  onRenameCategory: vi.fn(),
  onDeleteCategory: vi.fn(),
  onMoveFeedToCategory: vi.fn(),
  onMoveFeedToNewCategory: vi.fn(),
}

function renderNewSidebar(props?: Partial<Parameters<typeof NewSidebar>[0]>) {
  return render(
    <MemoryRouter>
      <NewSidebar
        feedCategories={mockFeedCategories}
        paperCategories={mockPaperCategories}
        feeds={mockFeeds}
        papers={mockPapers}
        {...defaultCallbacks}
        {...props}
      />
    </MemoryRouter>
  )
}

// --- Tests ---

describe('NewSidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSidebarState.selectedFeedId = null
    mockSidebarState.selectedPaperId = null
    mockSidebarState.expandedCategoryIds = new Set()
  })

  it('should render the oReader branding', () => {
    renderNewSidebar()
    expect(screen.getByText('oReader')).toBeInTheDocument()
  })

  it('should render the add button', () => {
    renderNewSidebar()
    expect(screen.getByText('新增')).toBeInTheDocument()
  })

  it('should call onAddClick when the add button is clicked', () => {
    renderNewSidebar()
    fireEvent.click(screen.getByText('新增'))
    expect(defaultCallbacks.onAddClick).toHaveBeenCalledOnce()
  })

  it('should render Feeds section header with count', () => {
    renderNewSidebar()
    expect(screen.getByText('Feeds')).toBeInTheDocument()
    expect(screen.getByText('(2)')).toBeInTheDocument()
  })

  it('should render Papers section header with count', () => {
    renderNewSidebar()
    expect(screen.getByText('Papers')).toBeInTheDocument()
    expect(screen.getByText('(1)')).toBeInTheDocument()
  })

  it('should render feed categories', () => {
    renderNewSidebar()
    expect(screen.getByText('Tech')).toBeInTheDocument()
  })

  it('should render feed titles in uncategorized section', () => {
    renderNewSidebar()
    expect(screen.getByText('Uncategorized Feed')).toBeInTheDocument()
  })

  it('should render paper titles', () => {
    renderNewSidebar()
    expect(screen.getByText('Attention Is All You Need')).toBeInTheDocument()
  })

  it('should render feeds inside expanded category', () => {
    mockSidebarState.expandedCategoryIds = new Set(['cat-feed-1'])
    renderNewSidebar()
    expect(screen.getByText('Hacker News')).toBeInTheDocument()
  })

  it('should render empty state when no feeds or papers', () => {
    renderNewSidebar({ feeds: [], papers: [] })
    expect(screen.getByText('Feeds')).toBeInTheDocument()
    expect(screen.getByText('Papers')).toBeInTheDocument()
    expect(screen.getAllByText('(0)').length).toBe(2)
  })

  describe('user avatar', () => {
    it('should show user initial from nickname', () => {
      renderNewSidebar()
      expect(screen.getByText('T')).toBeInTheDocument()
    })

    it('should show fallback initial when no user', () => {
      mockAuthState.user = null
      renderNewSidebar()
      expect(screen.getByText('?')).toBeInTheDocument()
    })
  })
})
