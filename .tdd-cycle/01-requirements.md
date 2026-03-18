# Article Management - Spec Compliance Fix: TDD Analysis

## Executive Summary

This analysis covers the requirements for fixing spec compliance gaps in the article management feature of the oReader application. The primary gaps are:
1. **ToggleStar** currently toggles instead of setting to request body value
2. **ToggleRead** currently toggles instead of setting to request body value
3. Items sorted by `created_at` instead of `pub_date` (published_at)
4. Need to verify `feed_title` is included in responses

---

## 1. Acceptance Criteria

### 1.1 List All Articles (GET /api/v1/items)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-LIST-001 | Returns paginated array of items | Response contains `items` array with pagination metadata (`total`, `has_more`, `next_cursor`) | Missing array or pagination fields |
| AC-LIST-002 | Items from subscribed feeds only | Only items from feeds user is subscribed to are returned | Items from unsubscribed feeds appear |
| AC-LIST-003 | Sorted by published_at descending | Items ordered by `pub_date` DESC (newest first) | Items ordered by any other field |
| AC-LIST-004 | Each item includes required fields | Each item has: `id`, `title`, `link`, `description`, `feed_id`, `feed_title`, `published_at` | Any required field missing |
| AC-LIST-005 | Includes is_starred from UserItemState | `is_starred` reflects user's starred state, defaults to `false` | Missing or incorrect value |
| AC-LIST-006 | Includes is_read from UserItemState | `is_read` reflects user's read state, defaults to `false` | Missing or incorrect value |
| AC-LIST-007 | Handles null pub_date gracefully | Items with null `pub_date` are included, sorted last | Errors or excludes such items |

### 1.2 Filter by Read Status (GET /api/v1/items?read=false)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-READ-001 | Returns only unread items when read=false | All returned items have `is_read=false` or no state | Any returned item has `is_read=true` |
| AC-READ-002 | Returns only read items when read=true | All returned items have `is_read=true` | Any returned item has `is_read=false` |
| AC-READ-003 | Ignores invalid read values | Invalid values like `read=invalid` are ignored, returns all items | Returns error or incorrect results |

### 1.3 Filter by Starred Status (GET /api/v1/items?starred=true)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-STAR-001 | Returns only starred items when starred=true | All returned items have `is_starred=true` | Any returned item has `is_starred=false` |
| AC-STAR-002 | Returns only unstarred items when starred=false | All returned items have `is_starred=false` | Any returned item has `is_starred=true` |
| AC-STAR-003 | Ignores invalid starred values | Invalid values are ignored | Returns error |

### 1.4 Star/Unstar Articles (PUT /api/v1/items/:id/star)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-STAROP-001 | Sets is_starred=true when body is {starred: true} | After request, `is_starred` is `true` | `is_starred` is toggled or `false` |
| AC-STAROP-002 | Sets is_starred=false when body is {starred: false} | After request, `is_starred` is `false` | `is_starred` is toggled or `true` |
| AC-STAROP-003 | Returns updated item | Response contains complete item with updated state | Missing item or stale state |
| AC-STAROP-004 | Creates UserItemState if not exists | New state record created with `is_starred=true` | Error or missing state |
| AC-STAROP-005 | Idempotent operations | Calling with same value twice returns same result | Second call changes state |
| AC-STAROP-006 | Returns 404 for non-existent item | HTTP 404 when item doesn't exist | Any other status code |
| AC-STAROP-007 | Returns 404 for item in another user's feed | HTTP 404 when user doesn't have access | Returns item or 403 |

### 1.5 Mark Articles as Read/Unread (PUT /api/v1/items/:id/read)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-READOP-001 | Sets is_read=true when body is {read: true} | After request, `is_read` is `true`, `read_at` is set | `is_read` is toggled or `false` |
| AC-READOP-002 | Sets is_read=false when body is {read: false} | After request, `is_read` is `false`, `read_at` is null | `is_read` is toggled or `true` |
| AC-READOP-003 | Returns updated item | Response contains complete item with updated state | Missing item or stale state |
| AC-READOP-004 | Creates UserItemState if not exists | New state record created | Error or missing state |
| AC-READOP-005 | Idempotent operations | Calling with same value twice returns same result | Second call changes state |
| AC-READOP-006 | Returns 404 for non-existent item | HTTP 404 when item doesn't exist | Any other status code |
| AC-READOP-007 | Returns 404 for item in another user's feed | HTTP 404 when user doesn't have access | Returns item or 403 |

### 1.6 Mark All Feed Articles as Read (POST /api/v1/feeds/:id/read-all)

| Criteria ID | Description | Pass Condition | Fail Condition |
|-------------|-------------|----------------|----------------|
| AC-ALLREAD-001 | Sets is_read=true for all items in feed | All items in feed have `is_read=true` | Any item remains unread |
| AC-ALLREAD-002 | Returns count of updated items | Response contains accurate `count` field | Missing or incorrect count |
| AC-ALLREAD-003 | Returns 404 for non-existent feed | HTTP 404 when feed doesn't exist or user not subscribed | Any other status code |
| AC-ALLREAD-004 | Handles empty feed | Returns `count: 0` without error | Error response |
| AC-ALLREAD-005 | Creates states for items without state | New UserItemState records created as needed | Error for items without existing state |

---

## 2. Edge Cases

### 2.1 Null/Empty Values

| Edge Case ID | Description | Expected Behavior | Test Priority |
|--------------|-------------|-------------------|---------------|
| EC-NULL-001 | Item with null `pub_date` | Include in results, sort last (oldest) | High |
| EC-NULL-002 | Item with empty `title` | Include with empty string | Medium |
| EC-NULL-003 | Item with empty `description` | Include with empty string | Medium |
| EC-NULL-004 | Item with null `content` | Include in results, content is null | Medium |
| EC-NULL-005 | Feed with empty `title` | Return empty string for `feed_title` | Medium |
| EC-NULL-006 | User with no subscriptions | Return empty array, total=0 | High |
| EC-NULL-007 | Feed with no items | Return empty array for feed items | High |

### 2.2 Boundary Values

| Edge Case ID | Description | Expected Behavior | Test Priority |
|--------------|-------------|-------------------|---------------|
| EC-BOUND-001 | `limit=0` | Use default limit (20) | Medium |
| EC-BOUND-002 | `limit=-1` | Use default limit (20) | Medium |
| EC-BOUND-003 | `limit=100` | Return up to 100 items | High |
| EC-BOUND-004 | `limit=101` | Cap at 100 items | Medium |
| EC-BOUND-005 | `limit=1000` | Cap at 100 items | Medium |
| EC-BOUND-006 | Very long item ID (1000 chars) | Return 404 or 400 | Low |
| EC-BOUND-007 | Very long feed ID | Return 404 or 400 | Low |
| EC-BOUND-008 | Pagination at exact boundary | Correct `has_more` and `next_cursor` | High |

### 2.3 Error States

| Edge Case ID | Description | Expected Behavior | Test Priority |
|--------------|-------------|-------------------|---------------|
| EC-ERR-001 | Malformed JSON in request body | Return 400 Bad Request | High |
| EC-ERR-002 | Missing `starred` field in star request | Return 400 Bad Request | High |
| EC-ERR-003 | Missing `read` field in read request | Return 400 Bad Request | High |
| EC-ERR-004 | Non-UUID item ID | Return 404 Not Found | Medium |
| EC-ERR-005 | Database connection failure | Return 500 Internal Server Error | High |
| EC-ERR-006 | Context timeout during operation | Return appropriate error | Medium |
| EC-ERR-007 | User not authenticated | Return 401 Unauthorized | High |

### 2.4 Concurrent Access

| Edge Case ID | Description | Expected Behavior | Test Priority |
|--------------|-------------|-------------------|---------------|
| EC-CONC-001 | Concurrent star operations on same item | Final state is consistent (last write wins) | High |
| EC-CONC-002 | Concurrent read operations on same item | All reads succeed | High |
| EC-CONC-003 | Mark all read while individual read toggle | No deadlock, consistent final state | Medium |
| EC-CONC-004 | User star/unstar same item rapidly | Final state matches last request | Medium |

### 2.5 Data Integrity

| Edge Case ID | Description | Expected Behavior | Test Priority |
|--------------|-------------|-------------------|---------------|
| EC-DATA-001 | Item deleted while being starred | Return 404 | Medium |
| EC-DATA-002 | Feed unsubscribed while listing items | Items from that feed not included | Medium |
| EC-DATA-003 | UserItemState orphaned (item deleted) | State ignored, not returned | Low |
| EC-DATA-004 | Same item starred by multiple users | Each user has independent state | High |

---

## 3. Test Scenario Matrix

### 3.1 Requirement to Test Case Mapping

| Requirement | Unit Tests | Integration Tests | Contract Tests | E2E Tests |
|-------------|------------|-------------------|----------------|-----------|
| **List all articles** | | | | |
| - Pagination | TC-UNIT-LIST-001~003 | TC-INT-LIST-001 | TC-CONT-LIST-001 | TC-E2E-LIST-001 |
| - Sort by pub_date DESC | TC-UNIT-LIST-004~006 | TC-INT-LIST-002 | - | TC-E2E-LIST-002 |
| - Required fields | TC-UNIT-LIST-007 | TC-INT-LIST-003 | TC-CONT-LIST-002 | TC-E2E-LIST-003 |
| - feed_title included | TC-UNIT-LIST-008 | TC-INT-LIST-004 | TC-CONT-LIST-003 | TC-E2E-LIST-004 |
| - Default is_starred=false | TC-UNIT-LIST-009 | TC-INT-LIST-005 | - | TC-E2E-LIST-005 |
| - Default is_read=false | TC-UNIT-LIST-010 | TC-INT-LIST-006 | - | TC-E2E-LIST-006 |
| **Filter by read** | | | | |
| - read=false returns unread | TC-UNIT-FILT-001 | TC-INT-FILT-001 | TC-CONT-FILT-001 | TC-E2E-FILT-001 |
| - read=true returns read | TC-UNIT-FILT-002 | TC-INT-FILT-002 | - | TC-E2E-FILT-002 |
| - Invalid value ignored | TC-UNIT-FILT-003 | TC-INT-FILT-003 | - | - |
| **Filter by starred** | | | | |
| - starred=true returns starred | TC-UNIT-FILT-004 | TC-INT-FILT-004 | TC-CONT-FILT-002 | TC-E2E-FILT-004 |
| - starred=false returns unstarred | TC-UNIT-FILT-005 | TC-INT-FILT-005 | - | TC-E2E-FILT-005 |
| **Star/Unstar** | | | | |
| - Set starred=true | TC-UNIT-STAR-001 | TC-INT-STAR-001 | TC-CONT-STAR-001 | TC-E2E-STAR-001 |
| - Set starred=false | TC-UNIT-STAR-002 | TC-INT-STAR-002 | - | TC-E2E-STAR-002 |
| - Idempotent | TC-UNIT-STAR-003 | TC-INT-STAR-003 | - | - |
| - Creates state if missing | TC-UNIT-STAR-004 | TC-INT-STAR-004 | - | - |
| - 404 for non-existent | TC-UNIT-STAR-005 | TC-INT-STAR-005 | TC-CONT-STAR-002 | TC-E2E-STAR-003 |
| - 404 for unauthorized | TC-UNIT-STAR-006 | TC-INT-STAR-006 | - | TC-E2E-STAR-004 |
| **Mark Read/Unread** | | | | |
| - Set read=true | TC-UNIT-READ-001 | TC-INT-READ-001 | TC-CONT-READ-001 | TC-E2E-READ-001 |
| - Set read=false | TC-UNIT-READ-002 | TC-INT-READ-002 | - | TC-E2E-READ-002 |
| - Idempotent | TC-UNIT-READ-003 | TC-INT-READ-003 | - | - |
| - Creates state if missing | TC-UNIT-READ-004 | TC-INT-READ-004 | - | - |
| - read_at set when read=true | TC-UNIT-READ-005 | TC-INT-READ-005 | - | TC-E2E-READ-003 |
| - read_at null when read=false | TC-UNIT-READ-006 | TC-INT-READ-006 | - | - |
| **Mark All Read** | | | | |
| - All items marked read | TC-UNIT-ALL-001 | TC-INT-ALL-001 | TC-CONT-ALL-001 | TC-E2E-ALL-001 |
| - Returns count | TC-UNIT-ALL-002 | TC-INT-ALL-002 | - | - |
| - Empty feed returns 0 | TC-UNIT-ALL-003 | TC-INT-ALL-003 | - | - |

### 3.2 Detailed Test Cases

#### Unit Tests - Service Layer

```
TC-UNIT-LIST-001: ListItems with limit returns correct number
TC-UNIT-LIST-002: ListItems with cursor returns correct page
TC-UNIT-LIST-003: ListItems has_more calculated correctly
TC-UNIT-LIST-004: ListItems sorts by pub_date descending (not null)
TC-UNIT-LIST-005: ListItems null pub_date sorted last
TC-UNIT-LIST-006: ListItems mixed null/non-null pub_date sorted correctly
TC-UNIT-LIST-007: ItemWithState contains all required fields
TC-UNIT-LIST-008: ItemWithState includes feed_title from preloaded Feed
TC-UNIT-LIST-009: Items without state default is_starred=false
TC-UNIT-LIST-010: Items without state default is_read=false

TC-UNIT-FILT-001: ListItems read=false returns only unread
TC-UNIT-FILT-002: ListItems read=true returns only read
TC-UNIT-FILT-003: ListItems read=invalid ignored
TC-UNIT-FILT-004: ListItems starred=true returns only starred
TC-UNIT-FILT-005: ListItems starred=false returns only unstarred

TC-UNIT-STAR-001: SetStar(starred=true) sets is_starred=true (not toggle)
TC-UNIT-STAR-002: SetStar(starred=false) sets is_starred=false (not toggle)
TC-UNIT-STAR-003: SetStar idempotent - calling twice with true returns same result
TC-UNIT-STAR-004: SetStar creates UserItemState if not exists
TC-UNIT-STAR-005: SetStar returns ErrItemNotFound for non-existent item
TC-UNIT-STAR-006: SetStar returns ErrItemNotFound for unauthorized feed

TC-UNIT-READ-001: SetRead(read=true) sets is_read=true (not toggle)
TC-UNIT-READ-002: SetRead(read=false) sets is_read=false (not toggle)
TC-UNIT-READ-003: SetRead idempotent - calling twice with true returns same result
TC-UNIT-READ-004: SetRead creates UserItemState if not exists
TC-UNIT-READ-005: SetRead(read=true) sets read_at timestamp
TC-UNIT-READ-006: SetRead(read=false) clears read_at to null
```

#### Unit Tests - Repository Layer

```
TC-UNIT-REPO-001: ListByFeedID orders by pub_date DESC NULLS LAST
TC-UNIT-REPO-002: ListStarred orders by pub_date DESC
TC-UNIT-REPO-003: ListUnread orders by pub_date DESC
TC-UNIT-REPO-004: ListByFeedID preloads Feed for feed_title
TC-UNIT-REPO-005: ListStarred preloads Feed for feed_title
TC-UNIT-REPO-006: ListUnread preloads Feed for feed_title
```

#### Integration Tests

```
TC-INT-LIST-001: Full pagination flow with database
TC-INT-LIST-002: Sorting verification with actual pub_date values
TC-INT-LIST-003: All required fields present in response
TC-INT-LIST-004: feed_title populated from Feed relation
TC-INT-LIST-005: is_starred defaults false when no UserItemState
TC-INT-LIST-006: is_read defaults false when no UserItemState

TC-INT-FILT-001: read=false filter with real data
TC-INT-FILT-002: read=true filter with real data
TC-INT-FILT-003: Invalid read parameter handling

TC-INT-STAR-001: SetStar(true) persists to database
TC-INT-STAR-002: SetStar(false) persists to database
TC-INT-STAR-003: Idempotency verification with database
TC-INT-STAR-004: State creation for new items
TC-INT-STAR-005: Non-existent item handling
TC-INT-STAR-006: Unauthorized access handling

TC-INT-READ-001: SetRead(true) persists to database
TC-INT-READ-002: SetRead(false) persists to database
TC-INT-READ-003: Idempotency verification with database
TC-INT-READ-004: State creation for new items
TC-INT-READ-005: read_at timestamp set correctly
TC-INT-READ-006: read_at cleared when marking unread

TC-INT-ALL-001: MarkAllRead updates all items in feed
TC-INT-ALL-002: MarkAllRead returns accurate count
TC-INT-ALL-003: MarkAllRead on empty feed
```

#### Contract Tests (API)

```
TC-CONT-LIST-001: GET /api/v1/items response schema validation
TC-CONT-LIST-002: Item response schema validation
TC-CONT-LIST-003: feed_title field present and string type

TC-CONT-FILT-001: GET /api/v1/items?read=false response schema
TC-CONT-FILT-002: GET /api/v1/items?starred=true response schema

TC-CONT-STAR-001: PUT /api/v1/items/:id/star request/response schema
TC-CONT-STAR-002: 404 error response schema

TC-CONT-READ-001: PUT /api/v1/items/:id/read request/response schema

TC-CONT-ALL-001: POST /api/v1/feeds/:id/read-all response schema
```

#### E2E Tests

```
TC-E2E-LIST-001: User lists items, paginates through all pages
TC-E2E-LIST-002: Verify newest items appear first
TC-E2E-LIST-003: All item fields present in UI-usable format
TC-E2E-LIST-004: feed_title displays correctly
TC-E2E-LIST-005: New items show unstarred by default
TC-E2E-LIST-006: New items show unread by default

TC-E2E-FILT-001: Filter to show only unread items
TC-E2E-FILT-002: Filter to show only read items
TC-E2E-FILT-004: Filter to show only starred items
TC-E2E-FILT-005: Filter to show only unstarred items

TC-E2E-STAR-001: Star an item, verify state persists on refresh
TC-E2E-STAR-002: Unstar an item, verify state persists on refresh
TC-E2E-STAR-003: Star non-existent item returns 404
TC-E2E-STAR-004: Cannot star item from another user's feed

TC-E2E-READ-001: Mark item as read, verify state persists
TC-E2E-READ-002: Mark item as unread, verify state persists
TC-E2E-READ-003: read_at timestamp shows when marked read

TC-E2E-ALL-001: Mark all items in feed as read, verify all updated
```

---

## 4. Test Categorization

### 4.1 Unit Tests

**Scope**: Individual functions/methods in isolation
**Location**:
- `/data00/home/wangyang.backend/work/oReader/internal/service/item_service_test.go`
- `/data00/home/wangyang.backend/work/oReader/internal/repository/item_repository_test.go`
- `/data00/home/wangyang.backend/work/oReader/internal/handler/item_handler_test.go`

**What to Test**:
- Service layer: business logic for SetStar (not toggle), SetRead (not toggle)
- Repository layer: SQL queries with correct ORDER BY clause
- Handler layer: request parsing, response formatting

**Mocking Required**:
- `ItemRepository` interface
- `UserItemStateRepository` interface
- `UserFeedRepository` interface

### 4.2 Integration Tests

**Scope**: Service + Repository + Database interactions
**Location**: Same files as unit tests, using real SQLite in-memory database

**What to Test**:
- Full data flow from handler to database
- Sorting behavior with real SQL queries
- State persistence and retrieval
- Concurrent access patterns

**Test Database**:
- SQLite in-memory for speed
- Consider PostgreSQL container for CI

### 4.3 Contract Tests

**Scope**: API request/response schema compliance
**Location**: `/data00/home/wangyang.backend/work/oReader/web/tests/` or dedicated contract test file

**What to Test**:
- JSON schema validation
- HTTP status codes
- Error response format
- Required field presence

**Tools**:
- JSON Schema validators
- OpenAPI spec validation (if spec exists)

### 4.4 Property-Based Tests

**Scope**: Invariants that should hold for all inputs
**What to Test**:
- Idempotency: `SetStar(id, true)` twice = same result
- Sorting: For any two items, order is consistent
- Default values: Any item without state has `is_starred=false, is_read=false`
- Pagination: `total >= len(items)`

**Tools**:
- Go's `testing/quick` package
- `gopter` library

---

## 5. External Dependencies to Mock

### 5.1 Repository Interfaces

```go
// ItemRepository - Mock required for service unit tests
type MockItemRepository struct {
    GetByIDFunc        func(ctx, id) (*Item, error)
    ListByFeedIDFunc   func(ctx, feedID, userID, opts) ([]*ItemWithState, int64, error)
    ListStarredFunc    func(ctx, userID, opts) ([]*ItemWithState, int64, error)
    ListUnreadFunc     func(ctx, userID, opts) ([]*ItemWithState, int64, error)
    CreateFunc         func(ctx, item) error
    CreateBatchFunc    func(ctx, items) error
    CountByFeedIDFunc  func(ctx, feedID) (int64, error)
}

// UserItemStateRepository - Mock required for service unit tests
type MockUserItemStateRepository struct {
    GetByUserAndItemFunc func(ctx, userID, itemID) (*UserItemState, error)
    CreateFunc           func(ctx, state) error
    UpdateFunc           func(ctx, state) error
    UpsertFunc           func(ctx, state) error
    BulkMarkReadFunc     func(ctx, userID, itemIDs) error
}

// UserFeedRepository - Mock required for service unit tests
type MockUserFeedRepository struct {
    GetByUserAndFeedFunc func(ctx, userID, feedID) (*UserFeed, error)
    ListByUserIDFunc     func(ctx, userID) ([]*UserFeed, error)
}
```

### 5.2 Service Interface

```go
// ItemService - Mock required for handler unit tests
type MockItemService struct {
    ListItemsFunc    func(ctx, userID, opts) (*ItemListResult, error)
    GetItemFunc      func(ctx, userID, itemID) (*ItemWithState, error)
    SetStarFunc      func(ctx, userID, itemID, starred bool) (*ItemWithState, error)  // NEW: takes bool param
    SetReadFunc      func(ctx, userID, itemID, read bool) (*ItemWithState, error)     // NEW: takes bool param
    MarkAllReadFunc  func(ctx, userID, feedID) (int, error)
}
```

### 5.3 External Systems

| Dependency | Mock Strategy | Test Level |
|------------|---------------|------------|
| Database (GORM/SQLite) | In-memory SQLite | Integration |
| HTTP Client (RSS fetch) | Not needed for article management | N/A |
| Authentication middleware | Set `user_id` in gin.Context manually | Handler/E2E |
| Time.Now() | Inject time provider or use fixed times | Unit |

---

## 6. Implementation Changes Required

### 6.1 Service Interface Changes

Current interface uses `ToggleStar` and `ToggleRead`. These need to be renamed/modified to accept a boolean parameter:

```go
// BEFORE (current)
ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error)
ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error)

// AFTER (spec compliant)
SetStar(ctx context.Context, userID, itemID string, starred bool) (*ItemWithState, error)
SetRead(ctx context.Context, userID, itemID string, read bool) (*ItemWithState, error)
```

### 6.2 Repository Changes

Change ORDER BY clause from `created_at DESC` to `pub_date DESC NULLS LAST`:

```go
// BEFORE
Order("created_at DESC")

// AFTER
Order("pub_date DESC NULLS LAST")
```

### 6.3 Response Structure Changes

Ensure `ItemWithState` includes `feed_title`:

```go
type ItemWithState struct {
    *model.Item
    FeedTitle   string  `json:"feed_title"`     // Add this
    IsStarred   bool    `json:"is_starred"`
    IsRead      bool    `json:"is_read"`
    ReadAt      *string `json:"read_at,omitempty"`
}
```

---

## 7. Test Implementation Priority

### Phase 1: Failing Tests (TDD Red)

1. **TC-UNIT-STAR-001**: SetStar(starred=true) sets is_starred=true
2. **TC-UNIT-STAR-002**: SetStar(starred=false) sets is_starred=false
3. **TC-UNIT-READ-001**: SetRead(read=true) sets is_read=true
4. **TC-UNIT-READ-002**: SetRead(read=false) sets is_read=false
5. **TC-UNIT-REPO-001**: ListByFeedID orders by pub_date DESC NULLS LAST
6. **TC-UNIT-LIST-008**: ItemWithState includes feed_title

### Phase 2: Implementation (TDD Green)

Implement the changes to make Phase 1 tests pass.

### Phase 3: Extended Tests

1. Idempotency tests
2. Edge case tests
3. Integration tests
4. Contract tests
5. E2E tests

### Phase 4: Property-Based Tests

Add property-based tests for invariants.

---

## 8. Summary

This analysis identifies:

- **28 acceptance criteria** across 6 requirement areas
- **28 edge cases** categorized by type
- **60+ test cases** mapped to requirements
- **4 test categories**: Unit, Integration, Contract, E2E
- **3 mock interfaces** required for isolation

The primary implementation changes are:
1. Rename `ToggleStar` to `SetStar` with boolean parameter
2. Rename `ToggleRead` to `SetRead` with boolean parameter
3. Change sorting from `created_at` to `pub_date DESC NULLS LAST`
4. Add `feed_title` to `ItemWithState` response structure
