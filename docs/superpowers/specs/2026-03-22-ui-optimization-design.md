# UI 优化项目设计文档

> **Document Type**: Technical Design Document
> **Status**: Approved
> **Author**: Claude Code
> **Date**: 2026-03-22
> **PRD Reference**: `design/PRD/ui-optimization.md`

---

## 1. Overview

本文档详细描述 oReader UI 优化的技术实现设计，包括：

1. **三栏布局** - Sidebar | ItemList | ArticlePanel
2. **计数显示优化** - All/Unread/Starred/Today 计数
3. **Today 过滤器** - 新增时间维度过滤

### 1.1 设计原则

- **最小改动**: 保持现有架构，仅添加必要组件
- **URL 同步**: 保持 URL 可分享，支持浏览器历史
- **响应式**: 渐进式降级 三栏 → 两栏 → 单栏
- **TDD**: 测试先行，确保质量

---

## 2. Component Design

### 2.1 新组件: ArticlePanel

**文件**: `web/src/components/items/ArticlePanel.tsx`

```typescript
interface ArticlePanelProps {
  itemId: string | null        // 选中的文章 ID，null 时显示空状态
  onClose?: () => void         // 移动端关闭按钮回调
  onToggleStar: (id: string, starred: boolean) => void
  onToggleRead: (id: string, read: boolean) => void
}
```

**UI 结构**:

```
┌─────────────────────────────────────────────┐
│  ← Back (仅移动端)    [Read] [Star] [Link] │  ← Actions bar
├─────────────────────────────────────────────┤
│  Feed Badge  •  Date  •  Author             │  ← Meta
│  Title                                       │
│  Description (摘要，可折叠)                  │
├─────────────────────────────────────────────┤
│  Article Content (prose)                    │  ← HTML 内容
└─────────────────────────────────────────────┘
```

**状态管理**:

| State | Condition | Display |
|-------|-----------|---------|
| loading | itemId 存在但数据未加载 | Loading Spinner |
| empty | itemId 为 null | "Select an article" 占位 |
| content | 数据加载完成 | 文章内容 |
| error | 加载失败 | Error message |

### 2.2 修改组件: Sidebar

**文件**: `web/src/components/feed/Sidebar.tsx`

**变更**:

```typescript
// 新增 prop
interface SidebarProps {
  // ... 现有 props
  stats?: StatsResponse  // 🆕 统计数据
}

// 扩展类型
export type FilterType = 'all' | 'unread' | 'starred' | 'today'  // 🆕 'today'

// 新增过滤器
const filters = [
  { type: 'all', label: 'All Items', icon: Home },
  { type: 'unread', label: 'Unread', icon: Rss },
  { type: 'starred', label: 'Starred', icon: Star },
  { type: 'today', label: 'Today', icon: Calendar },  // 🆕
]
```

**计数渲染**:

```tsx
{filters.map(filter => (
  <Button key={filter.type} ...>
    <Icon className="w-4 h-4 mr-2" />
    {filter.label}
    <Badge className="ml-auto">
      {stats?.[filter.type] ?? 0}
    </Badge>
  </Button>
))}
```

### 2.3 修改组件: ItemList

**文件**: `web/src/components/items/ItemList.tsx`

**变更**:

```typescript
interface ItemListProps {
  // ... 现有 props
  selectedItemId?: string | null  // 🆕 选中状态
}

// 选中样式
<Card className={cn(
  "cursor-pointer transition-colors",
  selectedItemId === article.id && "ring-2 ring-primary bg-accent"
)}>
```

### 2.4 重构页面: ItemsPage

**文件**: `web/src/pages/items/ItemsPage.tsx`

**新增状态**:

```typescript
const [selectedItemId, setSelectedItemId] = useState<string | null>(null)
const { data: stats } = useStats()
```

**URL 同步**:

```typescript
// 读取 URL 中的选中文章
useEffect(() => {
  const id = searchParams.get('id')
  if (id) setSelectedItemId(id)
}, [searchParams])

// 更新 URL
const handleItemClick = (id: string) => {
  setSelectedItemId(id)
  setSearchParams(prev => {
    prev.set('id', id)
    return prev
  })
}
```

**三栏布局**:

```tsx
<div className="flex h-screen">
  {/* 左栏: Sidebar */}
  <aside className="hidden md:flex w-64 border-r bg-background">
    <Sidebar stats={stats} ... />
  </aside>

  {/* 中栏 + 右栏 */}
  <main className="flex flex-1 overflow-hidden">
    {/* 中栏: ItemList */}
    <section
      className={cn(
        "w-80 border-r overflow-y-auto",
        selectedItemId && "hidden lg:block"  // 移动端隐藏
      )}
    >
      <ItemList
        selectedItemId={selectedItemId}
        onItemClick={handleItemClick}
        ...
      />
    </section>

    {/* 右栏: ArticlePanel */}
    <section className={cn(
      "flex-1 overflow-y-auto",
      !selectedItemId && "hidden lg:flex"  // 无选中时隐藏
    )}>
      <ArticlePanel
        itemId={selectedItemId}
        onClose={() => setSelectedItemId(null)}
        ...
      />
    </section>
  </main>
</div>
```

---

## 3. Data Flow Design

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ItemsPage                                      │
│                                                                             │
│  ┌─────────────┐    ┌──────────────────┐    ┌────────────────────────┐     │
│  │   Sidebar   │    │    ItemList      │    │    ArticlePanel        │     │
│  │             │    │                  │    │                        │     │
│  │ useStats()  │    │ useItems()       │    │ useGetItem(itemId)     │     │
│  │     │       │    │     │            │    │        │               │     │
│  │     ▼       │    │     ▼            │    │        ▼               │     │
│  │ [stats]     │    │ [items]          │    │ [item]                 │     │
│  │             │    │                  │    │                        │     │
│  │ onFilterChg │    │ onItemClick ─────┼────┼──▶ setSelectedItemId   │     │
│  │     │       │    │                  │    │                        │     │
│  └─────┼───────┘    └──────────────────┘    └────────────────────────┘     │
│        │                                                                    │
│        ▼                                                                    │
│  setSearchParams({ filter, feed })                                          │
│        │                                                                    │
│        ▼                                                                    │
│  URL: /items?filter=unread&feed=xxx&id=yyy                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 状态层级

| Level | Source | Purpose |
|-------|--------|---------|
| 1 | URL | Single Source of Truth (filter, feed, id) |
| 2 | React Query | Server State Cache (stats, items, feeds) |
| 3 | Zustand | Optimistic UI State (star/read 即时反馈) |

### 3.3 关键交互流程

#### 切换过滤器

```
User clicks "Today"
       │
       ▼
onFilterChange('today')
       │
       ▼
setSearchParams({ filter: 'today' })
       │
       ▼
ItemsPage reads filterType = 'today'
       │
       ▼
useItems({ published_today: true })
       │
       ▼
ItemList updates
```

#### 选中文章

```
User clicks article
       │
       ▼
onItemClick(itemId)
       │
       ▼
setSelectedItemId(itemId)
       │
       ▼
setSearchParams({ id: itemId })
       │
       ▼
ArticlePanel receives new itemId
       │
       ▼
useGetItem(itemId) fetches content
```

#### Star/Read 操作后同步

```
User clicks Star
       │
       ├──────────────────────────────┐
       ▼                              ▼
itemsStore.updateItem()         toggleStar.mutate()
(乐观更新 UI)                    (API 请求)
       │                              │
       │                              ▼
       │                         onSuccess:
       │                         └── invalidateQueries(['stats'])
       │                             invalidateQueries(['items'])
       ▼
UI responds immediately
```

### 3.4 useStats Hook

**文件**: `web/src/hooks/useStats.ts`

```typescript
import { useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'

interface StatsResponse {
  total: number
  unread: number
  starred: number
  today: number
}

export function useStats() {
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['stats'],
    queryFn: async () => {
      const { data } = await apiClient.get<StatsResponse>('/api/v1/stats')
      return data
    },
    staleTime: 30 * 1000,  // 30 seconds
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

---

## 4. Backend Code Structure

### 4.1 文件结构

```
internal/
├── handler/
│   └── stats_handler.go        # 🆕 新建
├── service/
│   └── stats_service.go        # 🆕 新建
├── repository/
│   ├── stats_repository.go     # 🆕 新建
│   └── item_repository.go      # 修改: 添加 today 支持
└── model/
    └── (无需修改)

cmd/server/
└── main.go                     # 修改: 注册路由
```

### 4.2 StatsRepository

**文件**: `internal/repository/stats_repository.go`

```go
package repository

import (
    "context"
    "database/sql"
    "time"
)

type StatsRepository interface {
    GetUserStats(ctx context.Context, userID string) (*UserStats, error)
}

type UserStats struct {
    Total   int64 `json:"total"`
    Unread  int64 `json:"unread"`
    Starred int64 `json:"starred"`
    Today   int64 `json:"today"`
}

type statsRepository struct {
    db *sql.DB
}

func NewStatsRepository(db *sql.DB) StatsRepository {
    return &statsRepository{db: db}
}

func (r *statsRepository) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
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

    todayStart := time.Now().UTC().Truncate(24 * time.Hour)

    var stats UserStats
    err := r.db.QueryRowContext(ctx, query, todayStart, userID, userID).Scan(
        &stats.Total,
        &stats.Unread,
        &stats.Starred,
        &stats.Today,
    )
    if err != nil {
        return nil, err
    }

    return &stats, nil
}
```

### 4.3 StatsService

**文件**: `internal/service/stats_service.go`

```go
package service

import (
    "context"
    "oreader/internal/repository"
)

type StatsService interface {
    GetUserStats(ctx context.Context, userID string) (*UserStatsResponse, error)
}

type UserStatsResponse struct {
    Total   int64 `json:"total"`
    Unread  int64 `json:"unread"`
    Starred int64 `json:"starred"`
    Today   int64 `json:"today"`
}

type statsService struct {
    statsRepo repository.StatsRepository
}

func NewStatsService(statsRepo repository.StatsRepository) StatsService {
    return &statsService{statsRepo: statsRepo}
}

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

### 4.4 StatsHandler

**文件**: `internal/handler/stats_handler.go`

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    apperrors "oreader/internal/infra/errors"
    "oreader/internal/service"
)

type StatsHandler struct {
    statsService service.StatsService
}

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

### 4.5 路由注册

**文件**: `cmd/server/main.go` (修改)

```go
func setupRoutes(r *gin.Engine, handlers *Handlers) {
    api := r.Group("/api/v1")
    api.Use(authMiddleware)

    // ... 现有路由 ...

    // 🆕 Stats API
    api.GET("/stats", handlers.Stats.GetStats)
}
```

### 4.6 ItemRepository 扩展

**文件**: `internal/repository/item_repository.go` (修改)

```go
type ListItemOptions struct {
    Limit           int
    Cursor          string
    FeedID          string
    Starred         *bool
    Read            *bool
    PublishedToday  *bool  // 🆕 新增
}

func (r *itemRepository) ListItems(ctx context.Context, userID string, opts ListItemOptions) ([]*ItemWithState, int64, error) {
    query := `
        SELECT i.*, uis.is_starred, uis.is_read, uis.read_at, f.title as feed_title
        FROM items i
        INNER JOIN user_feeds uf ON i.feed_id = uf.feed_id
        INNER JOIN feeds f ON i.feed_id = f.id
        LEFT JOIN user_item_states uis ON i.id = uis.item_id AND uis.user_id = ?
        WHERE uf.user_id = ? AND uf.deleted_at IS NULL
    `
    args := []interface{}{userID, userID}

    // 🆕 Today 过滤
    if opts.PublishedToday != nil && *opts.PublishedToday {
        query += " AND i.pub_date >= ?"
        todayStart := time.Now().UTC().Truncate(24 * time.Hour)
        args = append(args, todayStart)
    }

    // ... 其他过滤条件 ...
}
```

---

## 5. Testing Strategy

### 5.1 测试金字塔

```
               ┌───────────┐
               │   API     │  ← HTTP Handler 测试
               │Integration│
              ╱└───────────┘╲
             ╱               ╲
            ╱  ┌───────────┐  ╲
           ╱   │Repository │   ╲  ← 数据层测试
          ╱    │   Test    │    ╲
         ╱     └───────────┘     ╲
        ╱                         ╲
       ╱      ┌───────────┐        ╲
      ╱       │   Unit    │         ╲  ← 大量单元测试
     ╱        │   Tests   │          ╲
    ╱         └───────────┘           ╲

❌ 不使用: Playwright/Cypress E2E 浏览器测试
✅ 使用: Go test + MSW (前端 API mock)
```

### 5.2 TDD 流程

```
Phase 1: RED (写失败测试)
┌─────────────────────────────────────────────────────────────┐
│  1. stats_repository_test.go  - 测试数据库查询              │
│  2. stats_service_test.go     - 测试业务逻辑                │
│  3. stats_handler_test.go     - 测试 HTTP 端点              │
│  4. useStats.test.tsx         - 测试 React hook             │
│  5. ArticlePanel.test.tsx     - 测试组件渲染                │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
Phase 2: GREEN (实现代码)
┌─────────────────────────────────────────────────────────────┐
│  1. 实现 stats_repository.go                                │
│  2. 实现 stats_service.go                                   │
│  3. 实现 stats_handler.go                                   │
│  4. 实现 useStats.ts                                        │
│  5. 实现 ArticlePanel.tsx                                   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
Phase 3: REFACTOR (优化代码)
┌─────────────────────────────────────────────────────────────┐
│  1. 检查代码重复                                            │
│  2. 优化性能                                                │
│  3. 确保测试通过                                            │
└─────────────────────────────────────────────────────────────┘
```

### 5.3 后端测试示例

```go
// internal/handler/stats_handler_test.go

func TestStatsHandler_GetStats(t *testing.T) {
    tests := []struct {
        name       string
        userID     string
        wantStatus int
        wantBody   map[string]int64
    }{
        {
            name:       "returns stats for authenticated user",
            userID:     "user-1",
            wantStatus: http.StatusOK,
            wantBody: map[string]int64{
                "total": 10,
                "unread": 3,
                "starred": 2,
                "today": 1,
            },
        },
        {
            name:       "returns 401 for unauthenticated user",
            userID:     "",
            wantStatus: http.StatusUnauthorized,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/api/v1/stats", nil)
            if tt.userID != "" {
                req.Header.Set("X-User-ID", tt.userID)
            }

            w := httptest.NewRecorder()
            handler.GetStats(w, req)

            assert.Equal(t, tt.wantStatus, w.Code)
        })
    }
}
```

### 5.4 前端测试示例

```typescript
// web/src/hooks/__tests__/useStats.test.tsx

import { renderHook, waitFor } from '@testing-library/react'
import { rest } from 'msw'
import { setupServer } from 'msw/node'
import { createWrapper } from '@/test/utils'

const server = setupServer(
  rest.get('/api/v1/stats', (req, res, ctx) => {
    return res(ctx.json({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    }))
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('useStats', () => {
  it('should fetch stats from API', async () => {
    const wrapper = createWrapper()
    const { result } = renderHook(() => useStats(), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual({
      total: 100,
      unread: 25,
      starred: 10,
      today: 5,
    })
  })
})
```

### 5.5 验证命令

```bash
# 后端测试
go test ./internal/... -v -cover

# 前端测试
cd web && npm test -- --coverage

# API 端点验证
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/stats
```

### 5.6 手动验证清单

| # | 步骤 | 预期结果 |
|---|------|---------|
| 1 | 启动服务 `./dev.sh` | 服务正常运行 |
| 2 | 登录 | 跳转到 /items |
| 3 | 观察三栏布局 | Sidebar/ItemList/ArticlePanel 并排 |
| 4 | 查看计数 | All/Unread/Starred/Today 都有数字 |
| 5 | 点击 Today | 列表只显示今天文章 |
| 6 | 点击文章 | 右侧显示内容，URL 更新 |
| 7 | Star 操作 | 图标变黄，计数 +1 |
| 8 | 缩小窗口 <768px | 切换到单栏布局 |

---

## 6. Implementation Checklist

### Phase 1: 后端 API

- [ ] `internal/repository/stats_repository.go` - 新建
- [ ] `internal/service/stats_service.go` - 新建
- [ ] `internal/handler/stats_handler.go` - 新建
- [ ] `cmd/server/main.go` - 注册路由
- [ ] `internal/repository/item_repository.go` - 添加 published_today
- [ ] `internal/handler/item_handler.go` - 解析 published_today
- [ ] 后端单元测试

### Phase 2: 前端基础

- [ ] `web/src/types/feed.ts` - 添加 StatsResponse
- [ ] `web/src/hooks/useStats.ts` - 新建
- [ ] `web/src/components/items/ArticlePanel.tsx` - 新建
- [ ] 前端单元测试

### Phase 3: UI 整合

- [ ] `web/src/pages/items/ItemsPage.tsx` - 重构三栏布局
- [ ] `web/src/components/feed/Sidebar.tsx` - 添加计数 + Today
- [ ] `web/src/components/items/ItemList.tsx` - 添加选中样式
- [ ] 响应式样式

### Phase 4: 测试验证

- [ ] 所有单元测试通过
- [ ] 手动验证清单完成
- [ ] 代码审查

---

## 7. Appendix

### A. SQL 查询

```sql
-- Stats 查询
SELECT
  COUNT(*) as total,
  COUNT(CASE WHEN uis.is_read = false OR uis.is_read IS NULL THEN 1 END) as unread,
  COUNT(CASE WHEN uis.is_starred = true THEN 1 END) as starred,
  COUNT(CASE WHEN i.pub_date >= DATE('now') THEN 1 END) as today
FROM items i
JOIN user_feeds uf ON i.feed_id = uf.feed_id
LEFT JOIN user_item_states uis ON i.id = uis.item_id AND uis.user_id = ?
WHERE uf.user_id = ? AND uf.deleted_at IS NULL;
```

### B. 组件结构图

```
ItemsPage
├── Sidebar (w-64)
│   ├── Filters
│   │   ├── AllItemsFilter [stats.total]
│   │   ├── UnreadFilter [stats.unread]
│   │   ├── StarredFilter [stats.starred]
│   │   └── TodayFilter [stats.today]
│   └── FeedList
│       └── FeedCard[]
│
├── ItemList (w-80)
│   ├── ItemCard[] (selectedItemId highlight)
│   └── InfiniteScroll
│
└── ArticlePanel (flex-1)
    ├── ArticleHeader
    ├── ArticleContent
    └── ArticleSummary
```

### C. 响应式断点

| Breakpoint | Layout | Columns |
|------------|--------|---------|
| > 1024px | 三栏并排 | 3 |
| 768-1024px | 两栏 (Sidebar + Main) | 2 |
| < 768px | 单栏 + 抽屉 | 1 |
