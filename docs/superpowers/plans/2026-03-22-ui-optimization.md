# UI Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 oReader 三栏布局、计数显示和 Today 过滤器，提升阅读效率和信息可见性

**Architecture:** 后端新增 Stats API 端点 (Repository → Service → Handler)，前端重构 ItemsPage 为三栏布局，使用 URL 同步状态

**Tech Stack:** Go + Gin + GORM | React + TypeScript + Tailwind CSS + shadcn/ui | React Query + Zustand

---

## File Structure

### Backend (New Files)

| File | Purpose |
|------|---------|
| `internal/repository/stats_repository.go` | Stats 数据库查询 |
| `internal/repository/stats_repository_test.go` | Stats 单元测试 |
| `internal/service/stats_service.go` | Stats 业务逻辑 |
| `internal/service/stats_service_test.go` | Stats 单元测试 |
| `internal/handler/stats_handler.go` | GET /api/v1/stats HTTP handler |
| `internal/handler/stats_handler_test.go` | Handler HTTP 测试 |

### Backend (Modified Files)

| File | Change |
|------|--------|
| `internal/service/interfaces.go` | Add `StatsService` interface + `ListItemOptions.PublishedToday` |
| `internal/repository/item_repository.go` | Add published_today filter |
| `internal/handler/item_handler.go` | Parse `published_today` param |
| `cmd/server/main.go` | Register `/api/v1/stats` route |

### Frontend (New Files)

| File | Purpose |
|------|---------|
| `web/src/hooks/useStats.ts` | Stats API React Query hook |
| `web/src/hooks/__tests__/useStats.test.tsx` | Hook unit test |
| `web/src/components/items/ArticlePanel.tsx` | Article content panel component |
| `web/src/components/items/__tests__/ArticlePanel.test.tsx` | Component unit test |

### Frontend (Modified Files)

| File | Change |
|------|--------|
| `web/src/types/feed.ts` | Add `StatsResponse` interface + `published_today` |
| `web/src/components/feed/Sidebar.tsx` | Add counts + Today filter |
| `web/src/components/items/ItemList.tsx` | Add `selectedItemId` prop |
| `web/src/pages/items/ItemsPage.tsx` | Refactor to three-column layout |

---

## Task 1: Backend Stats Repository

**Files:**
- Create: `internal/repository/stats_repository.go`
- Create: `internal/repository/stats_repository_test.go`

- [ ] **Step 1.1: Write the failing test**

```go
// internal/repository/stats_repository_test.go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oreader/internal/model"
)

func TestStatsRepository_GetUserStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	repo := NewStatsRepository(db)
	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com"}
	require.NoError(t, db.Create(user).Error)

	t.Run("returns zeros for user with no feeds", func(t *testing.T) {
		stats, err := repo.GetUserStats(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Total)
		assert.Equal(t, int64(0), stats.Unread)
		assert.Equal(t, int64(0), stats.Starred)
		assert.Equal(t, int64(0), stats.Today)
	})

	t.Run("counts items correctly", func(t *testing.T) {
		// Create feed and items
		feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test"}
		require.NoError(t, db.Create(feed).Error)

		userFeed := &model.UserFeed{UserID: user.ID, FeedID: feed.ID}
		require.NoError(t, db.Create(userFeed).Error)

		// Today's item
		todayItem := &model.Item{
			FeedID:  feed.ID,
			GUID:    "today",
			Title:   "Today",
			PubDate: ptrTime(time.Now().UTC()),
		}
		require.NoError(t, db.Create(todayItem).Error)

		// Yesterday's item
		yesterdayItem := &model.Item{
			FeedID:  feed.ID,
			GUID:    "yesterday",
			Title:   "Yesterday",
			PubDate: ptrTime(time.Now().UTC().Add(-24 * time.Hour)),
		}
		require.NoError(t, db.Create(yesterdayItem).Error)

		stats, err := repo.GetUserStats(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.Total)
		assert.Equal(t, int64(2), stats.Unread)  // No state = unread
		assert.Equal(t, int64(0), stats.Starred)
		assert.Equal(t, int64(1), stats.Today)
	})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
```

- [ ] **Step 1.2: Run test to verify it fails**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/repository/... -run TestStatsRepository -v`
Expected: FAIL - "stats_repository.go: no such file"

- [ ] **Step 1.3: Write the implementation**

```go
// internal/repository/stats_repository.go
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserStats represents aggregated statistics for a user
type UserStats struct {
	Total   int64 `json:"total"`
	Unread  int64 `json:"unread"`
	Starred int64 `json:"starred"`
	Today   int64 `json:"today"`
}

// StatsRepository defines the interface for stats data access
type StatsRepository interface {
	GetUserStats(ctx context.Context, userID string) (*UserStats, error)
}

type statsRepository struct {
	db *gorm.DB
}

// NewStatsRepository creates a new stats repository
func NewStatsRepository(db *gorm.DB) StatsRepository {
	return &statsRepository{db: db}
}

// GetUserStats retrieves aggregated statistics for a user
func (r *statsRepository) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	var stats UserStats

	todayStart := time.Now().UTC().Truncate(24 * time.Hour)

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN uis.is_read = false OR uis.is_read IS NULL THEN 1 END) as unread,
			COUNT(CASE WHEN uis.is_starred = true THEN 1 END) as starred,
			COUNT(CASE WHEN i.pub_date >= ? THEN 1 END) as today
		FROM items i
		INNER JOIN user_feeds uf ON i.feed_id = uf.feed_id
		LEFT JOIN user_item_states uis ON i.id = uis.item_id AND uis.user_id = ?
		WHERE uf.user_id = ? AND uf.deleted_at IS NULL
	`

	err := r.db.WithContext(ctx).Raw(query, todayStart, userID, userID).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
```

- [ ] **Step 1.4: Run test to verify it passes**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/repository/... -run TestStatsRepository -v`
Expected: PASS

- [ ] **Step 1.5: Commit**

```bash
git add internal/repository/stats_repository.go internal/repository/stats_repository_test.go
git commit -m "feat(stats): add stats repository with GetUserStats method"
```

---

## Task 2: Backend Stats Service

**Files:**
- Create: `internal/service/stats_service.go`
- Create: `internal/service/stats_service_test.go`

- [ ] **Step 2.1: Add StatsService interface to interfaces.go**

```go
// Add to internal/service/interfaces.go after ItemService interface

// StatsService defines the interface for stats business logic
type StatsService interface {
	// GetUserStats retrieves aggregated statistics for a user
	GetUserStats(ctx context.Context, userID string) (*UserStatsResponse, error)
}

// UserStatsResponse contains the stats API response
type UserStatsResponse struct {
	Total   int64 `json:"total"`
	Unread  int64 `json:"unread"`
	Starred int64 `json:"starred"`
	Today   int64 `json:"today"`
}
```

- [ ] **Step 2.2: Write the failing test**

```go
// internal/service/stats_service_test.go
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oreader/internal/repository"
)

func TestStatsService_GetUserStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockStatsRepository(ctrl)
	svc := NewStatsService(mockRepo)
	ctx := context.Background()

	t.Run("returns stats from repository", func(t *testing.T) {
		expected := &repository.UserStats{
			Total:   100,
			Unread:  25,
			Starred: 10,
			Today:   5,
		}
		mockRepo.EXPECT().GetUserStats(ctx, "user-1").Return(expected, nil)

		result, err := svc.GetUserStats(ctx, "user-1")
		require.NoError(t, err)
		assert.Equal(t, int64(100), result.Total)
		assert.Equal(t, int64(25), result.Unread)
		assert.Equal(t, int64(10), result.Starred)
		assert.Equal(t, int64(5), result.Today)
	})
}
```

- [ ] **Step 2.3: Run test to verify it fails**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/service/... -run TestStatsService -v`
Expected: FAIL

- [ ] **Step 2.4: Write the implementation**

```go
// internal/service/stats_service.go
package service

import (
	"context"

	"oreader/internal/repository"
)

type statsService struct {
	statsRepo repository.StatsRepository
}

// NewStatsService creates a new stats service
func NewStatsService(statsRepo repository.StatsRepository) StatsService {
	return &statsService{statsRepo: statsRepo}
}

// GetUserStats retrieves aggregated statistics for a user
func (s *statsService) GetUserStats(ctx context.Context, userID string) (*UserStatsResponse, error) {
	stats, err := s.statsRepo.GetUserStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserStatsResponse{
		Total:   stats.Total,
		Unread:  stats.Unread,
		Starred: stats.Starred,
		Today:   stats.Today,
	}, nil
}
```

- [ ] **Step 2.5: Add mock to repository interface**

```go
// Add to internal/repository/stats_repository.go if needed for mock generation
// Or use moq to generate: moq -out stats_repository_mock.go . StatsRepository
```

- [ ] **Step 2.6: Run test to verify it passes**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/service/... -run TestStatsService -v`
Expected: PASS

- [ ] **Step 2.7: Commit**

```bash
git add internal/service/stats_service.go internal/service/stats_service_test.go internal/service/interfaces.go
git commit -m "feat(stats): add stats service with GetUserStats method"
```

---

## Task 3: Backend Stats Handler

**Files:**
- Create: `internal/handler/stats_handler.go`
- Create: `internal/handler/stats_handler_test.go`

- [ ] **Step 3.1: Write the failing test**

```go
// internal/handler/stats_handler_test.go
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"oreader/internal/service"
)

func TestStatsHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := service.NewMockStatsService(ctrl)
	handler := NewStatsHandler(mockService)

	t.Run("returns 401 for unauthenticated user", func(t *testing.T) {
		router := gin.New()
		router.GET("/stats", handler.GetStats)

		req := httptest.NewRequest("GET", "/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns stats for authenticated user", func(t *testing.T) {
		expected := &service.UserStatsResponse{
			Total:   100,
			Unread:  25,
			Starred: 10,
			Today:   5,
		}
		mockService.EXPECT().
			GetUserStats(gomock.Any(), "user-1").
			Return(expected, nil)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", "user-1")
			c.Next()
		})
		router.GET("/stats", handler.GetStats)

		req := httptest.NewRequest("GET", "/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"total":100`)
		assert.Contains(t, w.Body.String(), `"today":5`)
	})
}
```

- [ ] **Step 3.2: Run test to verify it fails**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/handler/... -run TestStatsHandler -v`
Expected: FAIL

- [ ] **Step 3.3: Write the implementation**

```go
// internal/handler/stats_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "oreader/internal/infra/errors"
	"oreader/internal/service"
)

// StatsHandler handles stats-related HTTP requests
type StatsHandler struct {
	statsService service.StatsService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(statsService service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetStats handles GET /api/v1/stats
func (h *StatsHandler) GetStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized,
			"User not authenticated", nil)
		return
	}

	stats, err := h.statsService.GetUserStats(c.Request.Context(), userID.(string))
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal,
			"Failed to retrieve stats", nil)
		return
	}

	c.JSON(http.StatusOK, stats)
}
```

- [ ] **Step 3.4: Run test to verify it passes**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/handler/... -run TestStatsHandler -v`
Expected: PASS

- [ ] **Step 3.5: Commit**

```bash
git add internal/handler/stats_handler.go internal/handler/stats_handler_test.go
git commit -m "feat(stats): add stats handler with GET /api/v1/stats endpoint"
```

---

## Task 4: Register Stats Route

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 4.1: Add stats service and handler initialization**

Find the section with other service/handler initializations in `cmd/server/main.go` and add:

```go
// After other services (around line 104)
statsService := service.NewStatsService(repository.NewStatsRepository(db))

// After other handlers (around line 127)
statsHandler := handler.NewStatsHandler(statsService)
```

- [ ] **Step 4.2: Register the route**

Find the API routes section and add:

```go
// After api.GET("/feeds", ...) routes (around line 160)
api.GET("/stats", statsHandler.GetStats)
```

- [ ] **Step 4.3: Verify with manual test**

Run: `./dev.sh`
Then: `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats`
Expected: `{"total":X,"unread":X,"starred":X,"today":X}`

- [ ] **Step 4.4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(stats): register GET /api/v1/stats route"
```

---

## Task 5: Backend Today Filter

**Files:**
- Modify: `internal/service/interfaces.go`
- Modify: `internal/repository/item_repository.go`
- Modify: `internal/handler/item_handler.go`

- [ ] **Step 5.1: Add PublishedToday to ListItemOptions**

```go
// In internal/service/interfaces.go, update ListItemOptions struct
type ListItemOptions struct {
	Limit           int    `json:"limit"`
	Cursor          string `json:"cursor,omitempty"`
	FeedID          string `json:"feed_id,omitempty"`
	Starred         *bool  `json:"starred,omitempty"`
	Read            *bool  `json:"read,omitempty"`
	PublishedToday  *bool  `json:"published_today,omitempty"` // NEW
}
```

- [ ] **Step 5.2: Add today filter to item repository**

Find the `ListItems` method in `internal/repository/item_repository.go` and add:

```go
// After other filter conditions in the WHERE clause
if opts.PublishedToday != nil && *opts.PublishedToday {
	todayStart := time.Now().UTC().Truncate(24 * time.Hour)
	query = query.Where("i.pub_date >= ?", todayStart)
}
```

- [ ] **Step 5.3: Parse published_today param in handler**

Find `ListItems` function in `internal/handler/item_handler.go` and add after the `read` param parsing:

```go
var publishedToday *bool
if todayStr := c.Query("published_today"); todayStr != "" {
	if todayStr == "true" {
		val := true
		publishedToday = &val
	}
}
```

Then add to opts:
```go
opts := service.ListItemOptions{
	// ... existing fields ...
	PublishedToday: publishedToday,
}
```

- [ ] **Step 5.4: Write test for today filter**

```go
// Add to internal/handler/item_handler_test.go
t.Run("filters by published_today", func(t *testing.T) {
	// Test that published_today=true is passed to service
})
```

- [ ] **Step 5.5: Run all backend tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/... -v`
Expected: All PASS

- [ ] **Step 5.6: Commit**

```bash
git add internal/service/interfaces.go internal/repository/item_repository.go internal/handler/item_handler.go
git commit -m "feat(items): add published_today filter parameter"
```

---

## Task 6: Frontend Types

**Files:**
- Modify: `web/src/types/feed.ts`

- [ ] **Step 6.1: Add StatsResponse and update ListItemsOptions**

```typescript
// Add to web/src/types/feed.ts

// Stats response
export interface StatsResponse {
  total: number
  unread: number
  starred: number
  today: number
}

// Update ListItemsOptions interface
export interface ListItemsOptions {
  limit?: number
  cursor?: string
  feed_id?: string
  starred?: boolean
  read?: boolean
  published_today?: boolean  // NEW
}
```

- [ ] **Step 6.2: Commit**

```bash
git add web/src/types/feed.ts
git commit -m "feat(types): add StatsResponse and published_today to ListItemsOptions"
```

---

## Task 7: useStats Hook

**Files:**
- Create: `web/src/hooks/useStats.ts`
- Create: `web/src/hooks/__tests__/useStats.test.tsx`

- [ ] **Step 7.1: Write the failing test**

```typescript
// web/src/hooks/__tests__/useStats.test.tsx
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { rest } from 'msw'
import { setupServer } from 'msw/node'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useStats } from '../useStats'

const server = setupServer()

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } }
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useStats', () => {
  it('should fetch stats from API', async () => {
    server.use(
      rest.get('http://localhost:8080/api/v1/stats', (req, res, ctx) => {
        return res(ctx.json({
          total: 100,
          unread: 25,
          starred: 10,
          today: 5,
        }))
      })
    )

    const { result } = renderHook(() => useStats(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    })
  })

  it('should handle API error', async () => {
    server.use(
      rest.get('http://localhost:8080/api/v1/stats', (req, res, ctx) => {
        return res(ctx.status(500))
      })
    )

    const { result } = renderHook(() => useStats(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
```

- [ ] **Step 7.2: Run test to verify it fails**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run useStats`
Expected: FAIL - module not found

- [ ] **Step 7.3: Write the implementation**

```typescript
// web/src/hooks/useStats.ts
import { useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type { StatsResponse } from '@/types/feed'

export function useStats() {
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['stats'],
    queryFn: async () => {
      const { data } = await apiClient.get<StatsResponse>('/api/v1/stats')
      return data
    },
    staleTime: 30 * 1000, // 30 seconds
    refetchOnWindowFocus: true,
  })

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['stats'] })
  }

  return {
    ...query,
    invalidate,
  }
}
```

- [ ] **Step 7.4: Run test to verify it passes**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run useStats`
Expected: PASS

- [ ] **Step 7.5: Commit**

```bash
git add web/src/hooks/useStats.ts web/src/hooks/__tests__/useStats.test.tsx
git commit -m "feat(hooks): add useStats hook with React Query"
```

---

## Task 8: ArticlePanel Component

**Files:**
- Create: `web/src/components/items/ArticlePanel.tsx`
- Create: `web/src/components/items/__tests__/ArticlePanel.test.tsx`

- [ ] **Step 8.1: Write the failing test**

```typescript
// web/src/components/items/__tests__/ArticlePanel.test.tsx
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ArticlePanel } from '../ArticlePanel'
import type { Article } from '@/types/feed'

const mockArticle: Article = {
  id: '1',
  feed_id: 'feed-1',
  guid: 'guid-1',
  title: 'Test Article',
  link: 'https://example.com/article',
  description: 'Test description',
  content: '<p>Test content</p>',
  pub_date: '2026-03-22T10:00:00Z',
  creator: 'Test Author',
  created_at: '2026-03-22T10:00:00Z',
  feed: {
    id: 'feed-1',
    title: 'Test Feed',
    feed_url: 'https://example.com/feed.xml',
    description: null,
    image_url: null,
    last_fetched_at: null,
    created_at: '2026-03-22T00:00:00Z',
  },
  user_state: {
    item_id: '1',
    is_read: false,
    is_starred: false,
    read_at: null,
  },
}

describe('ArticlePanel', () => {
  it('should show empty state when itemId is null', () => {
    render(
      <ArticlePanel
        itemId={null}
        onToggleStar={vi.fn()}
        onToggleRead={vi.fn()}
      />
    )
    expect(screen.getByText(/select an article/i)).toBeInTheDocument()
  })

  it('should show article content when loaded', () => {
    render(
      <ArticlePanel
        itemId="1"
        article={mockArticle}
        isLoading={false}
        onToggleStar={vi.fn()}
        onToggleRead={vi.fn()}
      />
    )
    expect(screen.getByText('Test Article')).toBeInTheDocument()
  })
})
```

- [ ] **Step 8.2: Run test to verify it fails**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run ArticlePanel`
Expected: FAIL

- [ ] **Step 8.3: Write the implementation**

```typescript
// web/src/components/items/ArticlePanel.tsx
import { ArrowLeft, Star, StarOff, Eye, EyeOff, ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { Article } from '@/types/feed'

interface ArticlePanelProps {
  itemId: string | null
  article?: Article | null
  isLoading?: boolean
  error?: Error | null
  onClose?: () => void
  onToggleStar: (id: string, starred: boolean) => void
  onToggleRead: (id: string, read: boolean) => void
  onRetry?: () => void
}

export function ArticlePanel({
  itemId,
  article,
  isLoading = false,
  error,
  onClose,
  onToggleStar,
  onToggleRead,
  onRetry,
}: ArticlePanelProps) {
  // Empty state
  if (!itemId) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        <p>Select an article to read</p>
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-4">
        <p className="text-destructive mb-4">Failed to load article</p>
        {onRetry && (
          <Button onClick={onRetry} variant="outline">Retry</Button>
        )}
      </div>
    )
  }

  // No article data
  if (!article) {
    return null
  }

  const isStarred = article.user_state?.is_starred ?? false
  const isRead = article.user_state?.is_read ?? false

  return (
    <div className="flex flex-col h-full">
      {/* Header with actions */}
      <div className="flex items-center justify-between p-4 border-b">
        {/* Mobile back button */}
        {onClose && (
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={onClose}
            aria-label="Back"
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
        )}

        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onToggleRead(article.id, !isRead)}
            aria-label={isRead ? 'Mark as unread' : 'Mark as read'}
          >
            {isRead ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onToggleStar(article.id, !isStarred)}
            className={cn(isStarred && 'text-yellow-500')}
            aria-label={isStarred ? 'Unstar' : 'Star'}
          >
            {isStarred ? <Star className="h-4 w-4 fill-current" /> : <StarOff className="h-4 w-4" />}
          </Button>
          {article.link && (
            <Button
              variant="ghost"
              size="icon"
              asChild
            >
              <a
                href={article.link}
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Open original"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          )}
        </div>
      </div>

      {/* Article content */}
      <div className="flex-1 overflow-y-auto p-6">
        {/* Meta */}
        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
          <span>{article.feed.title}</span>
          {article.pub_date && (
            <>
              <span>•</span>
              <span>{new Date(article.pub_date).toLocaleDateString()}</span>
            </>
          )}
          {article.creator && (
            <>
              <span>•</span>
              <span>{article.creator}</span>
            </>
          )}
        </div>

        {/* Title */}
        <h1 className="text-2xl font-bold mb-4">{article.title}</h1>

        {/* Summary (description) */}
        {article.description && (
          <div className="bg-muted/50 rounded-lg p-4 mb-6 text-sm text-muted-foreground">
            {article.description.replace(/<[^>]*>/g, '')}
          </div>
        )}

        {/* Content */}
        {article.content && (
          <div
            className="prose prose-sm dark:prose-invert max-w-none"
            dangerouslySetInnerHTML={{ __html: article.content }}
          />
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 8.4: Run test to verify it passes**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run ArticlePanel`
Expected: PASS

- [ ] **Step 8.5: Commit**

```bash
git add web/src/components/items/ArticlePanel.tsx web/src/components/items/__tests__/ArticlePanel.test.tsx
git commit -m "feat(components): add ArticlePanel component for three-column layout"
```

---

## Task 9: Sidebar Updates

**Files:**
- Modify: `web/src/components/feed/Sidebar.tsx`

- [ ] **Step 9.1: Add 'today' to FilterType and update filters**

```typescript
// In web/src/components/feed/Sidebar.tsx

// Update FilterType
export type FilterType = 'all' | 'unread' | 'starred' | 'today'

// Update filters array
import { Home, Rss, Star, Calendar } from 'lucide-react'

const filters = [
  { type: 'all' as FilterType, label: 'All Items', icon: Home },
  { type: 'unread' as FilterType, label: 'Unread', icon: Rss },
  { type: 'starred' as FilterType, label: 'Starred', icon: Star },
  { type: 'today' as FilterType, label: 'Today', icon: Calendar },  // NEW
]
```

- [ ] **Step 9.2: Add stats prop and display counts**

```typescript
// Update SidebarProps interface
interface SidebarProps {
  // ... existing props
  stats?: { total: number; unread: number; starred: number; today: number }
}

// In SidebarContent, update filter button rendering
{filters.map((filter) => {
  const Icon = filter.icon
  const isActive = filterType === filter.type

  // Get count for each filter type
  const count = filter.type === 'all' ? stats?.total ?? 0
    : filter.type === 'unread' ? stats?.unread ?? 0
    : filter.type === 'starred' ? stats?.starred ?? 0
    : filter.type === 'today' ? stats?.today ?? 0
    : 0

  return (
    <Button
      key={filter.type}
      variant={isActive ? 'secondary' : 'ghost'}
      className={cn('w-full justify-start', isActive && 'bg-accent')}
      onClick={() => onFilterChange(filter.type)}
    >
      <Icon className="w-4 h-4 mr-2" />
      {filter.label}
      {count > 0 && (
        <Badge variant="secondary" className="ml-auto">
          {count}
        </Badge>
      )}
    </Button>
  )
})}
```

- [ ] **Step 9.3: Update tests**

Add test case for Today filter and stats display.

- [ ] **Step 9.4: Run tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run Sidebar`
Expected: PASS

- [ ] **Step 9.5: Commit**

```bash
git add web/src/components/feed/Sidebar.tsx web/src/components/feed/__tests__/Sidebar.test.tsx
git commit -m "feat(sidebar): add Today filter and stats count display"
```

---

## Task 10: ItemList Selection

**Files:**
- Modify: `web/src/components/items/ItemList.tsx`

- [ ] **Step 10.1: Add selectedItemId prop**

```typescript
// In web/src/components/items/ItemList.tsx

interface ItemListProps {
  // ... existing props
  selectedItemId?: string | null  // NEW
}

// In the Card component, add selection styling
<Card
  key={article.id}
  className={cn(
    "cursor-pointer transition-colors hover:bg-accent/50",
    isRead && "bg-muted/30",
    selectedItemId === article.id && "ring-2 ring-primary bg-accent"  // NEW
  )}
  onClick={() => handleCardClick(article.id)}
>
```

- [ ] **Step 10.2: Run tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --run ItemList`
Expected: PASS

- [ ] **Step 10.3: Commit**

```bash
git add web/src/components/items/ItemList.tsx
git commit -m "feat(item-list): add selectedItemId prop for visual highlight"
```

---

## Task 11: ItemsPage Three-Column Layout

**Files:**
- Modify: `web/src/pages/items/ItemsPage.tsx`

- [ ] **Step 11.1: Add selectedItemId state and URL sync**

```typescript
// In ItemsPage.tsx

const [selectedItemId, setSelectedItemId] = useState<string | null>(() => {
  return searchParams.get('id')
})

// Sync URL when selection changes
useEffect(() => {
  if (selectedItemId) {
    setSearchParams(prev => {
      prev.set('id', selectedItemId)
      return prev
    })
  } else {
    setSearchParams(prev => {
      prev.delete('id')
      return prev
    })
  }
}, [selectedItemId, setSearchParams])

// Handle item click (update instead of navigate)
const handleItemClick = useCallback((itemId: string) => {
  setSelectedItemId(itemId)
  // Mark as read when selected
  if (!items.find(i => i.id === itemId)?.user_state?.is_read) {
    handleToggleRead(itemId, true)
  }
}, [items, handleToggleRead])
```

- [ ] **Step 11.2: Add today filter support**

```typescript
// Update listOptions
const listOptions: ListItemsOptions = {}
if (feedId) listOptions.feed_id = feedId
if (filterType === 'unread') listOptions.read = false
if (filterType === 'starred') listOptions.starred = true
if (filterType === 'today') listOptions.published_today = true  // NEW
listOptions.limit = 20
```

- [ ] **Step 11.3: Refactor to three-column layout**

```tsx
// Replace the return statement with three-column layout

const { data: stats } = useStats()
const selectedArticle = selectedItemId ? items.find(i => i.id === selectedItemId) : null

return (
  <div className="flex h-[calc(100vh-4rem)]">
    {/* Left: Sidebar - always visible on md+ */}
    <aside className="hidden md:flex w-64 border-r bg-background flex-shrink-0">
      <SidebarContent
        feeds={feeds}
        selectedFeedId={feedId}
        onFeedClick={handleFeedClick}
        onAddFeed={() => setIsAddFeedOpen(true)}
        onDeleteFeed={handleDeleteFeed}
        onRefreshFeed={handleRefreshFeed}
        refreshingFeedIds={refreshingFeedIds}
        filterType={filterType}
        onFilterChange={handleFilterChange}
        totalUnread={totalUnread}
        stats={stats}
      />
    </aside>

    {/* Mobile drawer */}
    <MobileDrawer open={isMobileMenuOpen} onClose={() => setIsMobileMenuOpen(false)}>
      <SidebarContent ... />
    </MobileDrawer>

    {/* Center: ItemList */}
    <section className={cn(
      "w-80 border-r overflow-y-auto flex-shrink-0",
      selectedItemId && "hidden lg:block"  // Hide on tablet when article selected
    )}>
      <ItemList
        articles={items}
        onItemClick={handleItemClick}
        onToggleStar={handleToggleStar}
        onToggleRead={handleToggleRead}
        isLoading={itemsLoading}
        selectedItemId={selectedItemId}
      />
    </section>

    {/* Right: ArticlePanel */}
    <section className={cn(
      "flex-1 overflow-hidden",
      !selectedItemId && "hidden lg:flex"  // Hide when no selection
    )}>
      <ArticlePanel
        itemId={selectedItemId}
        article={selectedArticle}
        isLoading={itemsLoading && selectedItemId && !selectedArticle}
        onClose={() => setSelectedItemId(null)}
        onToggleStar={handleToggleStar}
        onToggleRead={handleToggleRead}
      />
    </section>
  </div>
)
```

- [ ] **Step 11.4: Update tests**

Update ItemsPage tests to handle new layout and state.

- [ ] **Step 11.5: Run tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test`
Expected: All PASS

- [ ] **Step 11.6: Commit**

```bash
git add web/src/pages/items/ItemsPage.tsx web/src/pages/items/__tests__/ItemsPage.test.tsx
git commit -m "feat(items-page): refactor to three-column layout with URL sync"
```

---

## Task 12: Integration & Verification

- [ ] **Step 12.1: Run all backend tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui && go test ./internal/... -v -cover`
Expected: All PASS

- [ ] **Step 12.2: Run all frontend tests**

Run: `cd /data00/home/wangyang.backend/work/oReader-ui/web && npm test -- --coverage`
Expected: All PASS

- [ ] **Step 12.3: Start development server**

Run: `./dev.sh`

- [ ] **Step 12.4: Manual: Verify three-column layout on desktop (>=1024px)**
- [ ] **Step 12.5: Manual: Verify two-column layout on tablet (768-1023px)**
- [ ] **Step 12.6: Manual: Verify single-column layout on mobile (<768px)**
- [ ] **Step 12.7: Manual: Verify all filter counts display correctly**
- [ ] **Step 12.8: Manual: Verify Today filter shows today's articles only**
- [ ] **Step 12.9: Manual: Verify URL updates on article selection**
- [ ] **Step 12.10: Manual: Verify browser back/forward works**

- [ ] **Step 12.11: Final commit**

```bash
git add -A
git commit -m "feat(ui): complete three-column layout with stats and today filter

- Add GET /api/v1/stats endpoint
- Add published_today filter to ListItems API
- Add useStats hook and ArticlePanel component
- Refactor ItemsPage to three-column layout
- Add responsive breakpoints (lg:3-col, md:2-col, sm:1-col)
- Add Today filter with count display
- URL sync for article selection (?id=xxx)"
```

---

## Verification Commands

```bash
# Backend tests
go test ./internal/... -v -cover

# Frontend tests
cd web && npm test -- --coverage

# API verification
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats

# Today filter verification
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/api/v1/items?published_today=true"
```

---

## References

- PRD: `design/PRD/ui-optimization.md`
- Design: `docs/superpowers/specs/2026-03-22-ui-optimization-design.md`
- OpenSpec: `openspec/changes/ui-optimization/`
- Data Model: `openspec/specs/data-model/spec.md`
