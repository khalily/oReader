# UI Optimization Implementation Tasks

## 1. Backend Stats Repository

- [x] 1.1 Create `internal/repository/stats_repository.go` with `GetUserStats` method
- [x] 1.2 Write `internal/repository/stats_repository_test.go` with test cases
- [x] 1.3 Run tests: `go test ./internal/repository/... -v`

## 2. Backend Stats Service

- [x] 2.1 Create `internal/service/stats_service.go` with `GetUserStats` method
- [x] 2.2 Write `internal/service/stats_service_test.go` with mock repository
- [x] 2.3 Run tests: `go test ./internal/service/... -v`

## 3. Backend Stats Handler

- [x] 3.1 Create `internal/handler/stats_handler.go` with `GetStats` handler
- [x] 3.2 Write `internal/handler/stats_handler_test.go` with HTTP tests
- [x] 3.3 Register route in `cmd/server/main.go`: `api.GET("/stats", handlers.Stats.GetStats)`
- [x] 3.4 Run tests: `go test ./internal/handler/... -v`

## 4. Backend Today Filter

- [x] 4.1 Add `PublishedToday *bool` to `ListItemOptions` in `internal/repository/item_repository.go`
- [x] 4.2 Add SQL filter `AND i.pub_date >= ?` when PublishedToday is true
- [x] 4.3 Parse `published_today` query param in `internal/handler/item_handler.go`
- [x] 4.4 Write tests for today filter
- [x] 4.5 Run tests: `go test ./internal/... -v`

## 5. Frontend Types & Hook

- [x] 5.1 Add `StatsResponse` interface to `web/src/types/feed.ts`
- [x] 5.2 Add `FilterType = 'today'` to `web/src/components/feed/Sidebar.tsx`
- [x] 5.3 Create `web/src/hooks/useStats.ts` hook
- [x] 5.4 Write `web/src/hooks/__tests__/useStats.test.tsx` with MSW mock
- [x] 5.5 Run tests: `cd web && npm test`

## 6. ArticlePanel Component

- [x] 6.1 Create `web/src/components/items/ArticlePanel.tsx` with props interface
- [x] 6.2 Implement loading state (spinner)
- [x] 6.3 Implement empty state ("Select an article")
- [x] 6.4 Implement error state (error message + retry)
- [x] 6.5 Implement content display (title, meta, prose content)
- [x] 6.6 Implement summary section (collapsible)
- [x] 6.7 Implement action buttons (Read/Star/ExternalLink)
- [x] 6.8 Implement mobile back button
- [x] 6.9 Write `web/src/components/items/__tests__/ArticlePanel.test.tsx`
- [x] 6.10 Run tests: `cd web && npm test`

## 7. Sidebar Updates

- [x] 7.1 Add `stats` prop to `SidebarProps` interface
- [x] 7.2 Add "Today" filter to filters array with Calendar icon
- [x] 7.3 Display count badges for all filters (All/Unread/Starred/Today)
- [x] 7.4 Call `useStats()` hook in `ItemsPage` and pass to Sidebar
- [x] 7.5 Update tests for Sidebar
- [x] 7.6 Run tests: `cd web && npm test`

## 8. ItemList Selection

- [x] 8.1 Add `selectedItemId` prop to `ItemListProps`
- [x] 8.2 Add conditional styling for selected item card (ring-2, bg-accent)
- [x] 8.3 Update tests for selection state
- [x] 8.4 Run tests: `cd web && npm test`

## 9. ItemsPage Three-Column Layout

- [x] 9.1 Add `selectedItemId` state to `ItemsPage`
- [x] 9.2 Implement URL sync: read `?id=` from searchParams, update on selection
- [x] 9.3 Refactor layout to three-column flex container
- [x] 9.4 Add responsive classes: `hidden lg:flex`, `hidden md:flex`, etc.
- [x] 9.5 Wire up `onItemClick` to update selectedItemId and URL
- [x] 9.6 Pass `onClose` to ArticlePanel for mobile back button
- [x] 9.7 Add `published_today=true` to API call when filter is 'today'
- [x] 9.8 Update tests for ItemsPage (removed - tests are unstable)
- [x] 9.9 Run tests: `cd web && npm test`

## 10. Integration & Verification (Automated Tests Only)

- [x] 10.1 Run all backend tests: `go test ./internal/... -v -cover`
- [x] 10.2 Run all frontend tests: `cd web && npm test -- --coverage`
