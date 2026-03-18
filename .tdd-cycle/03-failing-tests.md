# Failing Tests Summary - TDD RED Phase

## Test Files Created/Modified

### 1. Service Layer Tests (`internal/service/item_service_test.go`)

#### SetStar Tests (Not Toggle)
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemService_SetStar_SetsToTrue` | Verifies `SetStar(starred=true)` sets `is_starred=true` | Method `SetStar` doesn't exist (only `ToggleStar`) |
| `TestItemService_SetStar_SetsToFalse` | Verifies `SetStar(starred=false)` sets `is_starred=false` | Method `SetStar` doesn't exist |
| `TestItemService_SetStar_Idempotent` | Verifies calling `SetStar(starred=true)` twice returns `true` both times | Method `SetStar` doesn't exist |

#### SetRead Tests (Not Toggle)
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemService_SetRead_SetsToTrue` | Verifies `SetRead(read=true)` sets `is_read=true` and `read_at` | Method `SetRead` doesn't exist |
| `TestItemService_SetRead_SetsToFalse` | Verifies `SetRead(read=false)` sets `is_read=false` and clears `read_at` | Method `SetRead` doesn't exist |
| `TestItemService_SetRead_Idempotent` | Verifies calling `SetRead(read=true)` twice returns `true` both times | Method `SetRead` doesn't exist |

### 2. Repository Layer Tests (`internal/repository/item_repository_test.go`)

#### Sorting Tests
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemRepository_ListByFeedID_SortsByPubDateDescNullsLast` | Verifies items sorted by `pub_date DESC NULLS LAST` | Current implementation uses `created_at DESC` |
| `TestItemRepository_ListStarred_SortsByPubDateDescNullsLast` | Verifies starred items sorted by `pub_date DESC NULLS LAST` | Current implementation uses `created_at DESC` |
| `TestItemRepository_ListUnread_SortsByPubDateDescNullsLast` | Verifies unread items sorted by `pub_date DESC NULLS LAST` | Current implementation uses `created_at DESC` |

### 3. Handler Layer Tests (`internal/handler/item_handler_test.go`)

#### SetStar Handler Tests
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemHandler_SetStar_SetsToTrue` | Verifies handler passes `starred=true` to service | Handler method `SetStar` doesn't exist |
| `TestItemHandler_SetStar_SetsToFalse` | Verifies handler passes `starred=false` to service | Handler method `SetStar` doesn't exist |
| `TestItemHandler_SetStar_Idempotent` | Verifies handler calls are idempotent | Handler method `SetStar` doesn't exist |

#### SetRead Handler Tests
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemHandler_SetRead_SetsToTrue` | Verifies handler passes `read=true` to service | Handler method `SetRead` doesn't exist |
| `TestItemHandler_SetRead_SetsToFalse` | Verifies handler passes `read=false` to service | Handler method `SetRead` doesn't exist |

#### feed_title Tests
| Test Name | Description | Expected Failure Reason |
|-----------|-------------|------------------------|
| `TestItemHandler_ListItems_IncludesFeedTitle` | Verifies `feed_title` is in response | `ItemWithState` doesn't have `FeedTitle` field |

---

## Test Count Summary

| Layer | Test Count | Coverage Area |
|-------|------------|---------------|
| Service | 6 | SetStar (3), SetRead (3) |
| Repository | 3 | Sorting by pub_date |
| Handler | 6 | SetStar (3), SetRead (2), feed_title (1) |
| **Total** | **15** | |

---

## Expected Test Output

When running `go test ./internal/...`, all new tests should fail with:

```
--- FAIL: TestItemService_SetStar_SetsToTrue (0.00s)
    item_service_test.go:XXX: service.SetStar undefined

--- FAIL: TestItemService_SetRead_SetsToTrue (0.00s)
    item_service_test.go:XXX: service.SetRead undefined

--- FAIL: TestItemRepository_ListByFeedID_SortsByPubDateDescNullsLast (0.00s)
    item_repository_test.go:XXX: Position 0: expected "Jan 15th (newest)", got "..." (wrong order)

--- FAIL: TestItemHandler_SetStar_SetsToTrue (0.00s)
    item_handler_test.go:XXX: handler.SetStar undefined

--- FAIL: TestItemHandler_ListItems_IncludesFeedTitle (0.00s)
    item_handler_test.go:XXX: unknown field FeedTitle in struct literal
```

These failures are **intentional** - they define the spec-compliant behavior we need to implement.
