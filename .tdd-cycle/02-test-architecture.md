# Test Architecture Design: Article Management Spec Compliance Fix

## Executive Summary

This document outlines a comprehensive test architecture for fixing spec compliance gaps in the oReader RSS application. The changes involve:
- Converting toggle operations to explicit set operations for star/read status
- Changing item sorting from `created_at DESC` to `pub_date DESC NULLS LAST`
- Ensuring `feed_title` is included in `ItemWithState` responses

---

## 1. Test Structure and Organization

### 1.1 Directory Layout

```
/data00/home/wangyang.backend/work/oReader/
├── internal/
│   ├── handler/
│   │   ├── item_handler.go
│   │   └── item_handler_test.go          # Handler unit tests (HTTP layer)
│   ├── service/
│   │   ├── item_service.go
│   │   ├── item_service_test.go          # Service unit tests (business logic)
│   │   └── interfaces.go
│   ├── repository/
│   │   ├── item_repository.go
│   │   ├── item_repository_test.go       # Repository integration tests (DB)
│   │   └── user_item_state_repository.go
│   └── model/
│       └── models.go
├── web/
│   └── tests/
│       ├── e2e/
│       │   ├── items.spec.ts             # E2E item tests
│       │   └── fixtures/
│       │       └── auth.ts               # Test fixtures
│       └── playwright.config.ts
└── internal/
    └── testutil/                          # NEW: Shared test utilities
        ├── fixtures.go                    # Test data factories
        ├── builders.go                    # Object builders
        ├── mocks.go                       # Mock implementations
        └── db.go                          # Database helpers
```

### 1.2 Naming Conventions

Following the existing pattern observed in the codebase:

**Go Tests:**
- `Test<Repository|Service|Handler>_<MethodName>_<Scenario>`
- Example: `TestItemService_SetStar_EnableStarred`

**Test Methods for New Requirements:**

| Layer | Test Name Pattern | Description |
|-------|-------------------|-------------|
| Repository | `TestItemRepository_ListByFeedID_SortsByPubDateDesc` | Verify sorting |
| Service | `TestItemService_SetStar_SetsToRequestValue` | Verify explicit set |
| Service | `TestItemService_SetRead_SetsToRequestValue` | Verify explicit set |
| Handler | `TestItemHandler_SetStar_AcceptsBooleanBody` | HTTP contract test |
| E2E | `SetStar → SetRead → Sorting` | End-to-end flow |

---

## 2. Fixture Design

### 2.1 Test Data Factory Pattern

Create a new `internal/testutil/fixtures.go`:

```go
package testutil

import (
    "time"
    "oreader/internal/model"
)

// ItemBuilder provides a fluent interface for creating test items
type ItemBuilder struct {
    item *model.Item
}

func NewItemBuilder() *ItemBuilder {
    now := time.Now()
    return &ItemBuilder{
        item: &model.Item{
            Base: model.Base{
                ID:        generateUUID(),
                CreatedAt: now,
                UpdatedAt: now,
            },
            FeedID:      "default-feed-id",
            GUID:        "default-guid",
            Title:       "Default Title",
            Link:        "https://example.com/default",
            Description: "Default description",
            Content:     "<p>Default content</p>",
        },
    }
}

// WithPubDate sets the publication date
func (b *ItemBuilder) WithPubDate(t time.Time) *ItemBuilder {
    b.item.PubDate = &t
    return b
}

// WithPubDateNil sets pub_date to NULL
func (b *ItemBuilder) WithPubDateNil() *ItemBuilder {
    b.item.PubDate = nil
    return b
}

// Build returns the constructed item
func (b *ItemBuilder) Build() *model.Item {
    return b.item
}

// FeedBuilder for creating test feeds
type FeedBuilder struct {
    feed *model.Feed
}

func NewFeedBuilder() *FeedBuilder {
    now := time.Now()
    return &FeedBuilder{
        feed: &model.Feed{
            Base: model.Base{
                ID:        generateUUID(),
                CreatedAt: now,
                UpdatedAt: now,
            },
            FeedURL: "https://example.com/feed.xml",
            Title:   "Test Feed",
        },
    }
}

// WithTitle sets the feed title (for feed_title verification)
func (b *FeedBuilder) WithTitle(title string) *FeedBuilder {
    b.feed.Title = title
    return b
}

// Build returns the constructed feed
func (b *FeedBuilder) Build() *model.Feed {
    return b.feed
}

// UserItemStateBuilder for creating test states
type UserItemStateBuilder struct {
    state *model.UserItemState
}

func NewUserItemStateBuilder() *UserItemStateBuilder {
    now := time.Now()
    return &UserItemStateBuilder{
        state: &model.UserItemState{
            Base: model.Base{
                ID:        generateUUID(),
                CreatedAt: now,
                UpdatedAt: now,
            },
            IsStarred: false,
            IsRead:    false,
        },
    }
}

// WithStarred sets the starred status
func (b *UserItemStateBuilder) WithStarred(starred bool) *UserItemStateBuilder {
    b.state.IsStarred = starred
    return b
}

// WithRead sets the read status
func (b *UserItemStateBuilder) WithRead(read bool) *UserItemStateBuilder {
    b.state.IsRead = read
    if read {
        now := time.Now()
        b.state.ReadAt = &now
    }
    return b
}

// Build returns the constructed state
func (b *UserItemStateBuilder) Build() *model.UserItemState {
    return b.state
}
```

### 2.2 Edge Case Data Sets

```go
// internal/testutil/fixtures.go

// SortingTestData creates items with various pub_date scenarios
func SortingTestData(feedID string) []*model.Item {
    now := time.Now()
    return []*model.Item{
        // Recent item with pub_date
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Recent Published").
            WithPubDate(now.Add(-1 * time.Hour)).
            Build(),

        // Older item with pub_date
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Older Published").
            WithPubDate(now.Add(-24 * time.Hour)).
            Build(),

        // Item with NULL pub_date (should be last)
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("No PubDate").
            WithPubDateNil().
            Build(),

        // Very old item
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Very Old").
            WithPubDate(now.Add(-7 * 24 * time.Hour)).
            Build(),
    }
}
```

### 2.3 Database Setup and Teardown

```go
// internal/testutil/db.go

package testutil

import (
    "testing"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "oreader/internal/model"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }

    if err := db.AutoMigrate(
        &model.Feed{},
        &model.User{},
        &model.Item{},
        &model.UserItemState{},
        &model.UserFeed{},
    ); err != nil {
        t.Fatalf("Failed to migrate database: %v", err)
    }

    return db
}
```

---

## 3. Mock/Stub Strategy

### 3.1 What to Mock

| Layer | Mock Strategy | Rationale |
|-------|--------------|-----------|
| **Handler Tests** | Mock `ItemService` | Isolate HTTP layer, test request/response handling |
| **Service Tests** | Mock repositories | Isolate business logic, avoid DB dependencies |
| **Repository Tests** | Use real SQLite in-memory DB | Test actual SQL queries and sorting behavior |
| **E2E Tests** | Use real backend with test database | Full integration test |

### 3.2 Service Mock for Handler Tests

```go
// Enhanced mock for handler tests
type mockItemService struct {
    listErr      error
    getErr       error
    setStarErr   error
    setReadErr   error
    markAllErr   error

    // Track SetStar/SetRead calls to verify request body values are used
    SetStarCalls []SetStarCall
    SetReadCalls []SetReadCall
}

type SetStarCall struct {
    UserID   string
    ItemID   string
    Starred  bool  // The value from request body
}

type SetReadCall struct {
    UserID   string
    ItemID   string
    Read     bool  // The value from request body
}

// SetStar implements ItemService interface
func (m *mockItemService) SetStar(ctx context.Context, userID, itemID string, starred bool) (*service.ItemWithState, error) {
    m.SetStarCalls = append(m.SetStarCalls, SetStarCall{
        UserID:  userID,
        ItemID:  itemID,
        Starred: starred,
    })

    if m.setStarErr != nil {
        return nil, m.setStarErr
    }

    item := &model.Item{
        Base: model.Base{ID: itemID},
        FeedID: "feed-1",
        Title: "Test Item",
    }
    return &service.ItemWithState{
        Item:      item,
        IsStarred: starred,  // Use the provided value
        IsRead:    false,
    }, nil
}

// SetRead implements ItemService interface
func (m *mockItemService) SetRead(ctx context.Context, userID, itemID string, read bool) (*service.ItemWithState, error) {
    m.SetReadCalls = append(m.SetReadCalls, SetReadCall{
        UserID: userID,
        ItemID: itemID,
        Read:   read,
    })

    if m.setReadErr != nil {
        return nil, m.setReadErr
    }

    item := &model.Item{
        Base: model.Base{ID: itemID},
        FeedID: "feed-1",
        Title: "Test Item",
    }
    return &service.ItemWithState{
        Item:      item,
        IsStarred: false,
        IsRead:    read,  // Use the provided value
    }, nil
}
```

---

## 4. Test Data Strategy

### 4.1 Sorting Test Data Generator

```go
// GenerateItemsForSorting creates a predictable dataset for sorting verification
func GenerateItemsForSorting(feedID string) []*model.Item {
    baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

    return []*model.Item{
        // Item 1: pub_date = 2024-01-01 12:00 (should be 2nd)
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Jan 1st").
            WithPubDate(baseTime).
            WithGUID("item-1").
            Build(),

        // Item 2: pub_date = 2024-01-02 12:00 (should be 1st)
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Jan 2nd").
            WithPubDate(baseTime.Add(24 * time.Hour)).
            WithGUID("item-2").
            Build(),

        // Item 3: pub_date = NULL (should be last)
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("No Date").
            WithPubDateNil().
            WithGUID("item-3").
            Build(),

        // Item 4: pub_date = 2023-12-31 12:00 (should be 3rd)
        NewItemBuilder().
            WithFeedID(feedID).
            WithTitle("Dec 31st").
            WithPubDate(baseTime.Add(-24 * time.Hour)).
            WithGUID("item-4").
            Build(),
    }
}
```

---

## 5. Test Execution Order and Parallelization

### 5.1 Test Execution Strategy

```
Phase 1: Unit Tests (Fast, Mocked)
├── Handler Tests (parallel)
│   ├── TestItemHandler_SetStar_*
│   ├── TestItemHandler_SetRead_*
│   └── TestItemHandler_ListItems_*
├── Service Tests (parallel)
│   ├── TestItemService_SetStar_*
│   ├── TestItemService_SetRead_*
│   └── TestItemService_ListItems_*
└── Repository Tests (parallel, isolated DB per test)
    ├── TestItemRepository_ListByFeedID_Sorting*
    ├── TestItemRepository_ListStarred_*
    └── TestItemRepository_ListUnread_*

Phase 2: Integration Tests (Slower, Real DB)
├── Repository Integration (parallel with shared DB)
│   ├── TestItemRepository_LargeDataset_*
│   └── TestItemRepository_RaceCondition_*
└── Service Integration (sequential)

Phase 3: E2E Tests (Slowest, Full Stack)
└── Playwright Tests (parallel by default)
    ├── items-setstar.spec.ts
    ├── items-setread.spec.ts
    └── items-sorting.spec.ts
```

### 5.2 Run Commands

```bash
# Run all tests in parallel
go test -parallel 4 ./internal/...

# Run with race detector (important for concurrent operations)
go test -race ./internal/repository/...

# Run specific package
go test -v ./internal/service/...
```

---

## 6. Test Implementation Guide

### 6.1 Repository Layer Tests (Sorting)

```go
// internal/repository/item_repository_test.go

func TestItemRepository_ListByFeedID_SortsByPubDateDescNullsLast(t *testing.T) {
    db := testutil.SetupTestDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()

    // Create user and feed
    user := &model.User{Email: "user@example.com", PasswordHash: "hash"}
    user.GenerateID()
    db.Create(user)

    feed := &model.Feed{FeedURL: "https://example.com/feed.xml", Title: "Test Feed"}
    feed.GenerateID()
    db.Create(feed)

    // Create items with varying pub_dates
    items := testutil.GenerateItemsForSorting(feed.ID)
    for _, item := range items {
        item.GenerateID()
    }
    repo.CreateBatch(ctx, items)

    // List items
    result, total, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 10})
    if err != nil {
        t.Fatalf("ListByFeedID() error: %v", err)
    }

    // Verify specific order: Jan 2nd, Jan 1st, Dec 31st, No Date
    expectedOrder := []string{"Jan 2nd", "Jan 1st", "Dec 31st", "No Date"}
    for i, item := range result {
        if item.Title != expectedOrder[i] {
            t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], item.Title)
        }
    }
}
```

### 6.2 Service Layer Tests (SetStar/SetRead)

```go
// internal/service/item_service_test.go

func TestItemService_SetStar_SetsToRequestValue(t *testing.T) {
    ctx := context.Background()
    userID := "user-1"
    itemID := "item-1"
    feedID := "feed-1"

    testCases := []struct {
        name           string
        initialState   bool
        requestValue   bool
        expectedResult bool
    }{
        {"unstarred to starred", false, true, true},
        {"starred to starred (idempotent)", true, true, true},
        {"starred to unstarred", true, false, false},
        {"unstarred to unstarred (idempotent)", false, false, false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Setup mocks
            itemRepo := &mockItemRepository{
                items: []*model.Item{
                    {Base: model.Base{ID: itemID}, FeedID: feedID},
                },
            }

            stateRepo := &mockUserItemStateRepository{
                states: []*model.UserItemState{
                    {UserID: userID, ItemID: itemID, IsStarred: tc.initialState},
                },
            }

            userFeedRepo := &mockUserFeedRepository{
                userFeeds: []*model.UserFeed{
                    {UserID: userID, FeedID: feedID},
                },
            }

            service := NewItemService(itemRepo, stateRepo, userFeedRepo)

            // Execute
            result, err := service.SetStar(ctx, userID, itemID, tc.requestValue)

            // Verify
            if err != nil {
                t.Fatalf("SetStar() error: %v", err)
            }

            if result.IsStarred != tc.expectedResult {
                t.Errorf("IsStarred = %v, want %v", result.IsStarred, tc.expectedResult)
            }
        })
    }
}
```

### 6.3 Handler Layer Tests (HTTP Contract)

```go
// internal/handler/item_handler_test.go

func TestItemHandler_SetStar_AcceptsBooleanBody(t *testing.T) {
    gin.SetMode(gin.TestMode)

    userID := "test-user-id"
    itemID := "item-1"

    mockService := &mockItemService{}
    handler := NewItemHandler(mockService)

    router := gin.New()
    router.PUT("/items/:id/star", func(c *gin.Context) {
        c.Set("user_id", userID)
        c.Next()
    }, handler.SetStar)

    testCases := []struct {
        name     string
        body     map[string]bool
        wantErr  bool
    }{
        {"starred=true", map[string]bool{"starred": true}, false},
        {"starred=false", map[string]bool{"starred": false}, false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            jsonBody, _ := json.Marshal(tc.body)
            req := httptest.NewRequest("PUT", "/items/"+itemID+"/star", bytes.NewReader(jsonBody))
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()

            router.ServeHTTP(w, req)

            if w.Code != http.StatusOK {
                t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
            }

            // Verify the service was called with the request body value
            if len(mockService.SetStarCalls) == 0 {
                t.Fatal("SetStar was not called")
            }

            call := mockService.SetStarCalls[len(mockService.SetStarCalls)-1]
            if call.Starred != tc.body["starred"] {
                t.Errorf("Service called with starred=%v, want %v", call.Starred, tc.body["starred"])
            }
        })
    }
}
```

---

## 7. Summary

This test architecture provides:

1. **Isolation**: Each layer tested independently with appropriate mocks
2. **Speed**: Unit tests < 10ms, integration tests < 100ms
3. **Reliability**: Deterministic test data, proper cleanup
4. **Coverage**: Comprehensive coverage of sorting, star/read operations
5. **Maintainability**: Reusable fixtures, clear naming conventions
6. **CI-Ready**: Parallelizable, fast feedback loop

### Key Implementation Changes Required

1. **Service Interface Changes**:
   ```go
   // BEFORE (current)
   ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error)
   ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error)

   // AFTER (spec compliant)
   SetStar(ctx context.Context, userID, itemID string, starred bool) (*ItemWithState, error)
   SetRead(ctx context.Context, userID, itemID string, read bool) (*ItemWithState, error)
   ```

2. **Repository Changes**:
   ```go
   // BEFORE
   Order("created_at DESC")

   // AFTER
   Order("pub_date DESC NULLS LAST")
   ```

3. **Response Structure Changes**:
   ```go
   type ItemWithState struct {
       *model.Item
       FeedTitle   string  `json:"feed_title"`     // Add this
       IsStarred   bool    `json:"is_starred"`
       IsRead      bool    `json:"is_read"`
       ReadAt      *string `json:"read_at,omitempty"`
   }
   ```
