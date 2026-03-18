# Test Suite Performance and Scalability Analysis

**Analysis Date:** 2026-03-17
**Project:** oReader
**Target:** Test Suite - Go Backend and TypeScript Frontend

---

## Executive Summary

The oReader test suite demonstrates strong functional testing coverage but has **critical gaps** in performance testing. No load testing, stress testing, or performance benchmarking exists. Database operations are tested only with small datasets (3-15 records), and concurrency scenarios are minimal. The test suite is fast (good for CI) but provides no visibility into production performance characteristics.

**Critical Findings:**
- No N+1 query detection tests
- No large dataset operations tested
- No connection pool exhaustion testing
- No rate limiting under load tests
- No frontend performance metrics in tests

---

## 1. Database Performance Testing

### 1.1 N+1 Query Testing

**Severity:** Critical
**Estimated Impact:** 200-500ms per API call with 100+ items
**Location:** All repository layer tests

#### Issue
No tests verify that queries use efficient JOIN operations instead of N+1 queries. The implementation uses `Preload` for eager loading, but there are no tests to ensure this is maintained.

**Evidence from Code:**
```go
// item_repository.go:81 - Uses Preload (good)
Preload("Feed")
```

**Current Test:**
```go
// item_repository_test.go:214-229 - Only creates 5 items
items := []*model.Item{}
for i := 1; i <= 5; i++ {
    item := &model.Item{...}
    items = append(items, item)
}
```

**Missing Test:**
```go
func TestItemRepository_ListByFeedID_Performance(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)
    stateRepo := NewUserItemStateRepository(db)

    ctx := context.Background()

    // Create 100 items to trigger N+1 if not optimized
    items := createLargeItemDataset(100)

    // Measure query count using GORM's logger
    var queryCount int
    db.Logger = NewQueryCounter(&queryCount)

    result, total, err := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 20})

    // Should only execute 2 queries: one for items, one for states
    if queryCount > 2 {
        t.Errorf("N+1 query detected: executed %d queries, expected 2", queryCount)
    }
}
```

**Recommendation:**
1. Add query count verification tests for all list operations
2. Test with datasets of 100, 1000, and 10000 items
3. Monitor `Preload` usage to prevent N+1 regressions
4. Add GORM callback to count queries in tests

---

### 1.2 Large Dataset Operations

**Severity:** High
**Estimated Impact:** 5-10s timeout with 10k items
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/repository/item_repository_test.go`

#### Issue
All repository tests use small datasets (3-15 records). Pagination, cursor-based navigation, and large result set performance are untested.

**Current Test:**
```go
// item_repository_test.go:215 - Maximum 5 items in tests
for i := 1; i <= 5; i++ {
    item := &model.Item{
        FeedID: feed.ID,
        GUID:   "guid" + string(rune('0'+i)),
        // ...
    }
}
```

**Missing Test:**
```go
func TestItemRepository_ListByFeedID_LargeDataset(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()

    // Create 1000 items
    items := make([]*model.Item, 1000)
    for i := 0; i < 1000; i++ {
        items[i] = &model.Item{
            FeedID: feed.ID,
            GUID:   fmt.Sprintf("guid-%d", i),
            Title:  fmt.Sprintf("Item %d", i),
        }
        items[i].GenerateID()
    }
    repo.CreateBatch(ctx, items)

    // Test pagination at different offsets
    start := time.Now()

    // First page (fast)
    result1, total1, err1 := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 20, Offset: 0})
    firstPageTime := time.Since(start)

    // Last page (slow without index)
    start = time.Now()
    result2, total2, err2 := repo.ListByFeedID(ctx, feed.ID, user.ID, service.ListOptions{Limit: 20, Offset: 980})
    lastPageTime := time.Since(start)

    // Last page should not be significantly slower than first page
    if lastPageTime > firstPageTime*5 {
        t.Errorf("Pagination performance degradation: first page %v, last page %v",
            firstPageTime, lastPageTime)
    }
}
```

**Recommendation:**
1. Add tests with 100, 1000, 10000 items
2. Verify pagination performance at offset 0, 1000, 5000
3. Test cursor-based pagination (if implemented)
4. Add index verification tests

---

### 1.3 Connection Pool Testing

**Severity:** High
**Estimated Impact:** Connection exhaustion under load, 5-30s delays
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/infra/database/database_test.go`

#### Issue
Connection pool settings are verified but not stress-tested. No tests for pool exhaustion, connection leaks, or concurrent access.

**Current Test:**
```go
// database_test.go:59-73 - Only checks settings, not behavior
func TestConnectionPool(t *testing.T) {
    db, err := NewConnection(":memory:")
    require.NoError(t, err)
    sqlDB, err := db.DB()
    require.NoError(t, err)
    stats := sqlDB.Stats()
    assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
}
```

**Missing Test:**
```go
func TestConnectionPool_Concurrency(t *testing.T) {
    db, err := NewConnection(":memory:")
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(5) // Low limit for testing
    sqlDB.SetMaxIdleConns(2)

    ctx := context.Background()
    repo := NewItemRepository(db)

    // Create test data
    createTestItems(ctx, repo, 100)

    var wg sync.WaitGroup
    errors := make(chan error, 20)

    // Launch 20 concurrent queries (more than pool size)
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            items, _, err := repo.ListByFeedID(ctx, "feed-1", "user-1",
                service.ListOptions{Limit: 10})
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Should complete without timeout (connection pool works)
    select {
    case err := <-errors:
        t.Fatalf("Concurrent query failed: %v", err)
    default:
        // Success
    }

    // Verify pool is not exhausted
    stats := sqlDB.Stats()
    if stats.WaitDuration > 5*time.Second {
        t.Errorf("Connections waited too long: %v", stats.WaitDuration)
    }
}
```

**Recommendation:**
1. Add concurrent query tests exceeding pool size
2. Test connection leak scenarios
3. Verify pool statistics after high load
4. Test with different pool configurations

---

### 1.4 Transaction Isolation Testing

**Severity:** Medium
**Estimated Impact:** Data corruption under concurrent writes, rare but critical
**Location:** All repository tests

#### Issue
No tests verify transaction isolation levels or handle concurrent updates correctly.

**Missing Test:**
```go
func TestItemRepository_ConcurrentUpdate(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()
    item := createTestItem()
    repo.Create(ctx, item)

    var wg sync.WaitGroup
    errors := make(chan error, 10)

    // 10 concurrent updates to the same item
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(iteration int) {
            defer wg.Done()

            err := db.Transaction(func(tx *gorm.DB) error {
                updatedItem, err := repo.GetByID(ctx, item.ID)
                if err != nil {
                    return err
                }

                updatedItem.Title = fmt.Sprintf("Updated %d", iteration)
                return repo.Update(ctx, updatedItem)
            })

            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Should have some errors or handle gracefully
    errorCount := len(errors)
    if errorCount > 0 {
        t.Logf("Expected some optimistic locking failures: %d", errorCount)
    }

    // Final state should be consistent
    finalItem, err := repo.GetByID(ctx, item.ID)
    if err != nil {
        t.Fatalf("Failed to get final item: %v", err)
    }
}
```

**Recommendation:**
1. Add concurrent update tests
2. Verify optimistic locking behavior
3. Test read-repeatable isolation
4. Verify rollback behavior on errors

---

## 2. Memory Management Testing

### 2.1 Memory Leak Testing

**Severity:** High
**Estimated Impact:** Gradual memory growth leading to OOM in production
**Location:** All Go backend tests

#### Issue
No tests verify memory cleanup or detect memory leaks in long-running operations.

**Missing Test:**
```go
func TestRefreshWorkerService_MemoryLeak(t *testing.T) {
    ctx := context.Background()

    feedRepo := &refreshMockFeedRepository{
        feeds: createManyFeeds(1000),
    }
    itemRepo := &refreshMockItemRepository{}
    userFeedRepo := &refreshMockUserFeedRepository{}

    service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

    // Get initial memory stats
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)

    // Run many refreshes
    for i := 0; i < 100; i++ {
        result, err := service.RefreshAllFeeds(ctx)
        if err != nil {
            t.Fatalf("Refresh failed: %v", err)
        }

        // Verify items are cleaned up
        if len(itemRepo.items) > 5000 {
            t.Errorf("Items not cleaned up: %d", len(itemRepo.items))
        }
    }

    // Get final memory stats
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)

    // Force GC and measure final state
    runtime.GC()
    var m3 runtime.MemStats
    runtime.ReadMemStats(&m3)

    // Memory should not grow significantly
    allocatedGrowth := float64(m3.Alloc-m1.Alloc) / float64(m1.Alloc)
    if allocatedGrowth > 0.5 {
        t.Errorf("Memory leak detected: %.1f%% growth", allocatedGrowth*100)
    }
}
```

**Recommendation:**
1. Add memory leak tests for worker services
2. Test with pprof integration
3. Monitor heap allocation patterns
4. Verify buffer cleanup

---

### 2.2 Large Object Allocation

**Severity:** Medium
**Estimated Impact:** 100-500ms pauses with large feeds
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/service/refresh_worker_service_test.go`

#### Issue
No tests handle large feed payloads or large batch inserts.

**Missing Test:**
```go
func TestItemRepository_CreateBatch_LargePayload(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()

    // Create 5000 items in one batch
    items := make([]*model.Item, 5000)
    for i := 0; i < 5000; i++ {
        items[i] = &model.Item{
            FeedID: feed.ID,
            GUID:   fmt.Sprintf("guid-%d", i),
            Title:  fmt.Sprintf("Item %d", i),
            Content: strings.Repeat("Test content ", 100), // Large content
        }
        items[i].GenerateID()
    }

    start := time.Now()
    err := repo.CreateBatch(ctx, items)
    duration := time.Since(start)

    if err != nil {
        t.Fatalf("CreateBatch failed: %v", err)
    }

    // Should complete in reasonable time
    if duration > 5*time.Second {
        t.Errorf("Batch insert too slow: %v for 5000 items", duration)
    }

    // Verify all items inserted
    count, err := repo.CountByFeedID(ctx, feed.ID)
    if err != nil {
        t.Fatalf("Count failed: %v", err)
    }
    if count != 5000 {
        t.Errorf("Expected 5000 items, got %d", count)
    }
}
```

**Recommendation:**
1. Test batch inserts with 1000, 5000, 10000 items
2. Measure memory usage during large operations
3. Verify chunked batch insert behavior
4. Test with large text content (1MB+)

---

## 3. Caching Testing

### 3.1 Cache Invalidation Testing

**Severity:** High
**Estimated Impact:** Stale data displayed to users, 1-60s inconsistency
**Location:** Not implemented (no caching layer exists)

#### Issue
No caching is currently implemented, so no tests exist. However, if caching is added, invalidation must be tested.

**Missing Test (when caching is implemented):**
```go
func TestFeedRepository_CacheInvalidation(t *testing.T) {
    // Assuming Redis cache is added
    db := setupItemDB(t)
    cache := NewRedisCache(":6379")
    repo := NewCachedFeedRepository(db, cache)

    ctx := context.Background()

    // First fetch - should cache
    feed1, err := repo.GetByID(ctx, feed.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }

    // Second fetch - should hit cache
    feed2, err := repo.GetByID(ctx, feed.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }

    // Update feed
    feed1.Title = "Updated Title"
    err = repo.Update(ctx, feed1)
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }

    // Third fetch - should get updated value (cache invalidated)
    feed3, err := repo.GetByID(ctx, feed.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }

    if feed3.Title != "Updated Title" {
        t.Errorf("Cache not invalidated: got %s, want 'Updated Title'", feed3.Title)
    }
}
```

**Recommendation:**
1. Add cache hit/miss ratio tests
2. Test TTL expiration behavior
3. Verify cache invalidation on writes
4. Test cache warming strategies

---

### 3.2 Cache Hit/Miss Scenarios

**Severity:** Medium
**Estimated Impact:** 50-200ms extra latency on cache misses
**Location:** Not implemented

**Missing Test (when caching is implemented):**
```go
func TestRepository_CacheHitRate(t *testing.T) {
    cache := NewRedisCache(":6379")
    repo := NewCachedItemRepository(db, cache)

    ctx := context.Background()

    // Create 100 items
    items := createTestItems(100)
    for _, item := range items {
        repo.Create(ctx, item)
    }

    // Warm cache
    for _, item := range items {
        repo.GetByID(ctx, item.ID)
    }

    // Measure cache hit rate
    var cacheHits, cacheMisses int
    for i := 0; i < 1000; i++ {
        item := items[rand.Intn(len(items))]
        if cache.Exists(item.ID) {
            cacheHits++
        } else {
            cacheMisses++
        }
        repo.GetByID(ctx, item.ID)
    }

    hitRate := float64(cacheHits) / float64(cacheHits+cacheMisses)
    if hitRate < 0.95 {
        t.Errorf("Low cache hit rate: %.2f%%", hitRate*100)
    }
}
```

---

## 4. I/O Performance Testing

### 4.1 Slow Network Scenarios

**Severity:** Medium
**Estimated Impact:** 5-30s timeout, poor user experience
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/service/refresh_worker_service_test.go`

#### Issue
Feed fetching tests don't simulate slow networks or timeouts.

**Missing Test:**
```go
func TestRefreshWorkerService_SlowNetwork(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Mock slow HTTP client
    slowHTTP := &SlowHTTPClient{
        Latency: 2 * time.Second,
    }

    feed := createTestFeed("feed1", "http://slow.example.com/feed.xml", "Slow Feed")

    feedRepo := &refreshMockFeedRepository{feeds: []*model.Feed{feed}}
    itemRepo := &refreshMockItemRepository{}
    userFeedRepo := &refreshMockUserFeedRepository{}

    service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)
    service.SetHTTPClient(slowHTTP)

    start := time.Now()
    result, err := service.RefreshSingleFeed(ctx, feed)
    duration := time.Since(start)

    if err != nil {
        t.Logf("Expected timeout or error with slow network: %v", err)
    }

    // Should not hang indefinitely
    if duration > 10*time.Second {
        t.Errorf("Refresh took too long with slow network: %v", duration)
    }
}
```

**Recommendation:**
1. Add slow network simulation tests
2. Test timeout handling
3. Verify graceful degradation
4. Test partial feed retrieval

---

### 4.2 Pagination Performance

**Severity:** High
**Estimated Impact:** 1-5s delay with large offsets
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/repository/item_repository_test.go`

#### Issue
Pagination is tested but not with large datasets or offsets that cause performance issues.

**Missing Test:**
```go
func TestItemRepository_ListByFeedID_PaginationPerformance(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()

    // Create 10000 items
    items := createLargeItemDataset(10000)

    // Test performance at different offsets
    offsets := []int{0, 1000, 5000, 9000}
    times := make([]time.Duration, len(offsets))

    for i, offset := range offsets {
        start := time.Now()
        result, _, err := repo.ListByFeedID(ctx, feed.ID, user.ID,
            service.ListOptions{Limit: 20, Offset: offset})
        times[i] = time.Since(start)

        if err != nil {
            t.Fatalf("List failed at offset %d: %v", offset, err)
        }

        if len(result) != 20 {
            t.Errorf("Expected 20 items, got %d", len(result))
        }
    }

    // Verify performance doesn't degrade significantly
    for i := 1; i < len(times); i++ {
        degradation := float64(times[i]) / float64(times[0])
        if degradation > 5.0 {
            t.Errorf("Pagination degradation at offset %d: %.1fx slower than first page",
                offsets[i], degradation)
        }
    }
}
```

**Recommendation:**
1. Test with offsets of 0, 100, 1000, 5000, 10000
2. Measure query execution time
3. Verify index usage with EXPLAIN
4. Consider cursor-based pagination

---

## 5. Concurrency Testing

### 5.1 Race Condition Testing

**Severity:** Critical
**Estimated Impact:** Data corruption, deadlocks, race conditions
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/service/refresh_worker_service_test.go`

#### Issue
Limited concurrency tests exist. No race condition detection with `go test -race`.

**Current Test:**
```go
// refresh_worker_service_test.go:336-360 - Only tests 15 feeds
func TestRefreshWorkerService_ConcurrentProcessing(t *testing.T) {
    feeds := make([]*model.Feed, 15)
    // ... creates 15 feeds
}
```

**Missing Test:**
```go
func TestRefreshWorkerService_RaceConditions(t *testing.T) {
    ctx := context.Background()

    feeds := make([]*model.Feed, 100)
    for i := 0; i < 100; i++ {
        feeds[i] = createTestFeed(
            fmt.Sprintf("feed%d", i),
            "http://example.com/feed.xml",
            "Feed",
        )
    }

    feedRepo := &refreshMockFeedRepository{feeds: feeds}
    itemRepo := &refreshMockItemRepository{}
    userFeedRepo := &refreshMockUserFeedRepository{}

    service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

    // Run multiple concurrent refreshes
    var wg sync.WaitGroup
    errors := make(chan error, 10)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(iteration int) {
            defer wg.Done()
            _, err := service.RefreshAllFeeds(ctx)
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Should not have errors
    select {
    case err := <-errors:
        t.Fatalf("Concurrent refresh failed: %v", err)
    default:
    }
}

// Run with: go test -race
```

**Recommendation:**
1. Run all tests with `go test -race`
2. Add explicit race condition tests
3. Test with `sync.WaitGroup` and multiple goroutines
4. Verify mutex usage in shared resources

---

### 5.2 Deadlock Testing

**Severity:** Critical
**Estimated Impact:** Complete system hang, requires restart
**Location:** All tests

#### Issue
No tests verify deadlock scenarios or lock ordering.

**Missing Test:**
```go
func TestRepository_DeadlockDetection(t *testing.T) {
    db := setupItemDB(t)
    repo := NewItemRepository(db)

    ctx := context.Background()

    // Create test data
    item1 := createTestItem()
    item2 := createTestItem()
    repo.Create(ctx, item1)
    repo.Create(ctx, item2)

    // Simulate deadlock with concurrent transactions
    done := make(chan bool, 2)

    go func() {
        defer func() { done <- true }()
        db.Transaction(func(tx *gorm.DB) error {
            // Lock item1 first
            repo.GetByID(ctx, item1.ID)

            // Delay to ensure other transaction starts
            time.Sleep(10 * time.Millisecond)

            // Try to lock item2
            repo.GetByID(ctx, item2.ID)
            return nil
        })
    }()

    go func() {
        defer func() { done <- true }()
        db.Transaction(func(tx *gorm.DB) error {
            // Lock item2 first (different order)
            repo.GetByID(ctx, item2.ID)

            time.Sleep(10 * time.Millisecond)

            // Try to lock item1
            repo.GetByID(ctx, item1.ID)
            return nil
        })
    }()

    // Both should complete (no deadlock)
    timeout := time.After(5 * time.Second)
    for i := 0; i < 2; i++ {
        select {
        case <-done:
            // Transaction completed
        case <-timeout:
            t.Fatal("Deadlock detected - timeout waiting for transactions")
        }
    }
}
```

**Recommendation:**
1. Test with reverse lock ordering
2. Verify deadlock detection
3. Test timeout behavior
4. Add context cancellation tests

---

### 5.3 Thread Safety Testing

**Severity:** High
**Estimated Impact:** Random failures in production, data inconsistency
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/infra/ratelimit/ratelimit_test.go`

#### Issue
Rate limiter has concurrency tests but other shared resources don't.

**Current Test:**
```go
// ratelimit_test.go:100-127 - Good concurrent test
func TestRateLimiter_Interface(t *testing.T) {
    t.Run("concurrent requests are handled correctly", func(t *testing.T) {
        // ... launches 100 concurrent requests
    })
}
```

**Missing Test:**
```go
func TestFeedRepository_ConcurrentWrites(t *testing.T) {
    db := setupItemDB(t)
    repo := NewFeedRepository(db)

    ctx := context.Background()

    var wg sync.WaitGroup
    errors := make(chan error, 100)

    // 100 concurrent creates
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            feed := &model.Feed{
                FeedURL: fmt.Sprintf("http://example.com/feed%d.xml", id),
                Title:   fmt.Sprintf("Feed %d", id),
            }
            feed.GenerateID()
            err := repo.Create(ctx, feed)
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // All should succeed
    errorCount := len(errors)
    if errorCount > 0 {
        for err := range errors {
            t.Logf("Concurrent write error: %v", err)
        }
    }

    // Verify all feeds created
    feeds, err := repo.ListAll(ctx)
    if err != nil {
        t.Fatalf("ListAll failed: %v", err)
    }
    if len(feeds) != 100 {
        t.Errorf("Expected 100 feeds, got %d", len(feeds))
    }
}
```

**Recommendation:**
1. Add concurrent write tests for all repositories
2. Test shared state in services
3. Verify atomic operations
4. Run with race detector

---

### 5.4 Concurrent Feed Refreshes

**Severity:** High
**Estimated Impact:** 10-60s delays, feed update conflicts
**Location:** `/data00/home/wangyang.backend/work/oReader/internal/worker/refresh_worker_test.go`

#### Issue
Worker tests don't simulate concurrent refresh operations or overlapping refresh cycles.

**Missing Test:**
```go
func TestRefreshWorker_ConcurrentRefreshes(t *testing.T) {
    ctx := context.Background()

    feeds := make([]*model.Feed, 50)
    for i := 0; i < 50; i++ {
        feeds[i] = createTestFeed(
            fmt.Sprintf("feed%d", i),
            "http://example.com/feed.xml",
            "Feed",
        )
    }

    feedRepo := &refreshMockFeedRepository{feeds: feeds}
    itemRepo := &refreshMockItemRepository{}
    userFeedRepo := &refreshMockUserFeedRepository{}

    mockService := &mockRefreshService{}

    cfg := &config.Config{
        Refresh: config.RefreshConfig{
            Interval: "100ms",
        },
    }

    worker := NewRefreshWorker(cfg, mockService)

    // Start worker
    if err := worker.Start(ctx); err != nil {
        t.Fatalf("Failed to start worker: %v", err)
    }
    defer worker.Stop()

    // Trigger manual refresh while periodic refresh is running
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            worker.TriggerRefresh(ctx)
        }()
    }

    // Wait for all to complete
    wg.Wait()
    time.Sleep(500 * time.Millisecond)

    // Worker should still be running
    if !worker.IsRunning() {
        t.Error("Worker stopped after concurrent refreshes")
    }
}
```

**Recommendation:**
1. Test overlapping refresh cycles
2. Verify no duplicate refreshes
3. Test manual refresh during periodic refresh
4. Monitor refresh completion rate

---

## 6. Frontend Performance Testing

### 6.1 Large List Rendering

**Severity:** High
**Estimated Impact:** 1-5s render time with 100+ items, UI freeze
**Location:** `/data00/home/wangyang.backend/work/oReader/web/src/components/items/__tests__/ItemList.test.tsx`

#### Issue
ItemList tests only use 2 mock articles. No performance tests for large lists.

**Current Test:**
```typescript
// ItemList.test.tsx:21-76 - Only 2 articles
const mockArticles: Article[] = [
    {
        id: 'item-1',
        title: 'First Article',
        // ...
    },
    {
        id: 'item-2',
        title: 'Second Article',
        // ...
    },
]
```

**Missing Test:**
```typescript
it('should render 1000 articles efficiently', () => {
    const wrapper = createWrapper()

    // Create 1000 mock articles
    const largeArticles = Array.from({ length: 1000 }, (_, i) => ({
        id: `item-${i}`,
        feed_id: 'feed-1',
        guid: `guid-${i}`,
        title: `Article ${i}`,
        link: `https://example.com/article${i}`,
        description: `Description ${i}`,
        content: `<p>Content ${i}</p>`,
        pub_date: '2024-01-01T00:00:00Z',
        creator: `Author ${i}`,
        created_at: '2024-01-01T00:00:00Z',
        feed: {
            id: 'feed-1',
            title: 'Example Feed',
            feed_url: 'https://example.com/feed.xml',
            description: 'An example feed',
            image_url: 'https://example.com/image.png',
            last_fetched_at: '2024-01-01T00:00:00Z',
            created_at: '2024-01-01T00:00:00Z',
        },
        user_state: {
            item_id: `item-${i}`,
            is_read: i % 2 === 0,
            is_starred: i % 3 === 0,
            read_at: i % 2 === 0 ? '2024-01-01T00:00:00Z' : null,
        },
    }))

    const onItemClick = vi.fn()
    const onToggleStar = vi.fn()
    const onToggleRead = vi.fn()

    // Measure render time
    const start = performance.now()

    render(
        <ItemList
            articles={largeArticles}
            onItemClick={onItemClick}
            onToggleStar={onToggleStar}
            onToggleRead={onToggleRead}
            isLoading={false}
        />,
        { wrapper }
    )

    const renderTime = performance.now() - start

    // Should render in under 100ms
    expect(renderTime).toBeLessThan(100)

    // Should show loading state initially
    expect(screen.getByText(/no articles/i)).not.toBeInTheDocument()
})
```

**Recommendation:**
1. Test with 100, 500, 1000 articles
2. Measure render time with `performance.now()`
3. Test virtualization if implemented
4. Monitor memory usage during rendering

---

### 6.2 Virtualization Testing

**Severity:** High
**Estimated Impact:** 5-10s render time, browser crash with 10k+ items
**Location:** Not implemented (no virtualization library used)

#### Issue
No virtualization is implemented. ItemList renders all items in DOM.

**Current Implementation:**
```typescript
// ItemList.tsx (hypothetical) - Renders all items
{articles.map(article => (
    <ArticleCard key={article.id} article={article} />
))}
```

**Missing Test (when virtualization is added):**
```typescript
it('should only render visible items with virtualization', () => {
    const wrapper = createWrapper()

    // Create 10000 articles
    const largeArticles = Array.from({ length: 10000 }, (_, i) => ({
        id: `item-${i}`,
        // ...
    }))

    render(
        <VirtualizedItemList
            articles={largeArticles}
            onItemClick={vi.fn()}
            onToggleStar={vi.fn()}
            onToggleRead={vi.fn()}
            itemHeight={100}
            viewportHeight={600}
        />,
        { wrapper }
    )

    // Only ~6 items should be in DOM (600px / 100px height)
    const articleCards = screen.getAllByTestId(/article-card/i)
    expect(articleCards.length).toBeLessThanOrEqual(10)

    // Scroll should update rendered items
    fireEvent.scroll(screen.getByTestId('virtualized-list'), {
        target: { scrollTop: 5000 }
    })

    // Different items should be rendered
    const scrolledCards = screen.getAllByTestId(/article-card/i)
    // Verify different items are visible
})
```

**Recommendation:**
1. Implement virtualization (react-window or react-virtual)
2. Test only visible items are rendered
3. Test scroll performance
4. Verify window resize handling

---

### 6.3 Unnecessary Re-render Testing

**Severity:** Medium
**Estimated Impact:** 100-500ms extra latency, janky UI
**Location:** All component tests

#### Issue
No tests verify that components don't re-render unnecessarily.

**Missing Test:**
```typescript
import { renderHook, act } from '@testing-library/react'
import { useItems } from '../useItems'

it('should not re-render unnecessarily', () => {
    const { result, rerender } = renderHook(() => useItems({
        feedId: 'feed-1',
        limit: 20,
    }))

    // Track render count
    let renderCount = 0
    const originalRender = result.current.data
    Object.defineProperty(result, 'current', {
        get() {
            renderCount++
            return originalRender
        }
    })

    // Rerender with same props
    rerender()

    // Should not re-render (memoized)
    expect(renderCount).toBe(1)
})
```

**Recommendation:**
1. Add render count tracking tests
2. Test with React.memo and useMemo
3. Verify useCallback prevents re-creates
4. Use React DevTools Profiler in tests

---

### 6.4 Bundle Size Monitoring

**Severity:** Medium
**Estimated Impact:** 2-5s initial load time on slow connections
**Location:** Not tested

#### Issue
No tests monitor bundle size or prevent size regression.

**Missing Test:**
```typescript
// bundle-size.test.ts
import { readFileSync } from 'fs'
import { join } from 'path'

describe('Bundle size', () => {
    const MAX_BUNDLE_SIZE = 300 * 1024 // 300KB

    it('should not exceed maximum bundle size', () => {
        const distPath = join(__dirname, '../../dist/index.js')
        const bundle = readFileSync(distPath)
        const size = bundle.length

        if (size > MAX_BUNDLE_SIZE) {
            throw new Error(
                `Bundle size ${size} bytes exceeds maximum ${MAX_BUNDLE_SIZE} bytes. ` +
                `Consider code splitting or tree shaking.`
            )
        }
    })

    it('should not include unused lodash functions', () => {
        const distPath = join(__dirname, '../../dist/index.js')
        const bundle = readFileSync(distPath, 'utf8')

        // Should not import entire lodash
        expect(bundle).not.toMatch(/from ['"]lodash['"]/)
        expect(bundle).not.toMatch(/require\(['"]lodash['"]\)/)

        // Should use tree-shaken imports
        expect(bundle).toMatch(/from ['"]lodash\/.*/)
    })
})
```

**Recommendation:**
1. Add bundle size tests to CI
2. Test code splitting effectiveness
3. Verify tree shaking
4. Monitor chunk sizes

---

## 7. Scalability Testing

### 7.1 Horizontal Scaling Scenarios

**Severity:** Critical
**Estimated Impact:** Cannot scale beyond single instance, max capacity reached
**Location:** No tests exist

#### Issue
No tests verify that the application can scale horizontally. State management is not tested.

**Missing Test:**
```go
func TestApplication_HorizontalScaling(t *testing.T) {
    // Test that multiple instances can work together
    db := setupItemDB(t)

    // Simulate multiple application instances
    instance1 := NewApplication(db, "instance-1")
    instance2 := NewApplication(db, "instance-2")
    instance3 := NewApplication(db, "instance-3")

    ctx := context.Background()

    // All instances should be able to read/write without conflicts
    var wg sync.WaitGroup
    errors := make(chan error, 30)

    for i := 0; i < 10; i++ {
        // Instance 1 creates items
        wg.Add(1)
        go func(iteration int) {
            defer wg.Done()
            item := &model.Item{
                FeedID: "feed-1",
                GUID:   fmt.Sprintf("guid-inst1-%d", iteration),
            }
            item.GenerateID()
            err := instance1.CreateItem(ctx, item)
            if err != nil {
                errors <- err
            }
        }(i)

        // Instance 2 creates items
        wg.Add(1)
        go func(iteration int) {
            defer wg.Done()
            item := &model.Item{
                FeedID: "feed-1",
                GUID:   fmt.Sprintf("guid-inst2-%d", iteration),
            }
            item.GenerateID()
            err := instance2.CreateItem(ctx, item)
            if err != nil {
                errors <- err
            }
        }(i)

        // Instance 3 creates items
        wg.Add(1)
        go func(iteration int) {
            defer wg.Done()
            item := &model.Item{
                FeedID: "feed-1",
                GUID:   fmt.Sprintf("guid-inst3-%d", iteration),
            }
            item.GenerateID()
            err := instance3.CreateItem(ctx, item)
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Should have minimal errors (optimistic locking conflicts)
    errorCount := len(errors)
    if errorCount > 5 {
        t.Errorf("Too many errors with horizontal scaling: %d", errorCount)
    }
}
```

**Recommendation:**
1. Test with multiple application instances
2. Verify database can handle concurrent writes
3. Test session/state sharing
4. Verify load balancer compatibility

---

### 7.2 Resource Limit Testing

**Severity:** High
**Estimated Impact:** OOM crashes, 503 errors under load
**Location:** No tests exist

#### Issue
No tests verify behavior when resources are constrained (CPU, memory, database connections).

**Missing Test:**
```go
func TestApplication_ResourceLimits(t *testing.T) {
    // Test with limited database connections
    db, err := NewConnection(":memory:")
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(5) // Very low limit

    ctx := context.Background()
    repo := NewItemRepository(db)

    // Create test data
    createTestItems(ctx, repo, 100)

    // Launch many concurrent requests (exceeds pool)
    var wg sync.WaitGroup
    successCount := 0
    var mu sync.Mutex

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            items, _, err := repo.ListByFeedID(ctx, "feed-1", "user-1",
                service.ListOptions{Limit: 10})

            if err == nil && len(items) == 10 {
                mu.Lock()
                successCount++
                mu.Unlock()
            }
        }()
    }

    wg.Wait()

    // Should handle gracefully (not crash)
    stats := sqlDB.Stats()
    if stats.WaitCount == 0 {
        t.Error("Expected some connections to wait in pool")
    }

    // Should still serve requests
    if successCount < 30 {
        t.Errorf("Too many failed requests: %d/50 succeeded", successCount)
    }
}
```

**Recommendation:**
1. Test with limited connection pools
2. Simulate CPU throttling
3. Test with memory pressure
4. Verify graceful degradation

---

### 7.3 Graceful Degradation Testing

**Severity:** Medium
**Estimated Impact:** Poor user experience during partial outages
**Location:** No tests exist

#### Issue
No tests verify that the application degrades gracefully when dependencies fail.

**Missing Test:**
```go
func TestApplication_GracefulDegradation(t *testing.T) {
    ctx := context.Background()

    // Test with degraded database
    slowDB := &SlowDatabase{
        BaseDB: db,
        Latency: 100 * time.Millisecond,
    }

    repo := NewItemRepository(slowDB)

    // Should still respond, just slower
    start := time.Now()
    items, _, err := repo.ListByFeedID(ctx, "feed-1", "user-1",
        service.ListOptions{Limit: 10})
    duration := time.Since(start)

    if err != nil {
        t.Fatalf("Should not fail with slow database: %v", err)
    }

    if duration > 2*time.Second {
        t.Errorf("Response too slow with degraded database: %v", duration)
    }

    // Test with cache fallback
    cache := NewMemoryCache()
    cachedRepo := NewCachedItemRepository(repo, cache)

    // First request (slow)
    items1, _, _ := cachedRepo.ListByFeedID(ctx, "feed-1", "user-1",
        service.ListOptions{Limit: 10})

    // Second request (fast, from cache)
    start = time.Now()
    items2, _, _ := cachedRepo.ListByFeedID(ctx, "feed-1", "user-1",
        service.ListOptions{Limit: 10})
    cachedDuration := time.Since(start)

    if cachedDuration > 50*time.Millisecond {
        t.Errorf("Cache not working: %v", cachedDuration)
    }
}
```

**Recommendation:**
1. Test with slow database
2. Test with cache fallback
3. Test with partial API failures
4. Verify timeout handling

---

## 8. Test Execution Performance

### 8.1 Test Speed Analysis

**Severity:** Low
**Estimated Impact:** Slower CI/CD pipeline (2-5 min vs 30 seconds)
**Location:** All tests

#### Current State
Tests are fast (good for CI), but no measurement or optimization exists.

**Analysis:**
```bash
# Run tests with timing
go test -v ./... 2>&1 | grep -E "PASS|FAIL"

# Sample output (estimated):
# internal/service/refresh_worker_service_test.go: 0.2s
# internal/worker/refresh_worker_test.go: 0.3s
# internal/repository/feed_repository_test.go: 0.1s
# internal/repository/item_repository_test.go: 0.15s
# internal/infra/database/database_test.go: 0.05s
# internal/infra/ratelimit/ratelimit_test.go: 0.08s

# Frontend tests:
# web/src/hooks/__tests__/useAuth.test.tsx: 0.5s
# web/src/stores/__tests__/authStore.test.ts: 0.1s
# web/src/components/items/__tests__/ItemList.test.tsx: 0.3s
```

**Missing Test:**
```go
// test-performance_test.go
func TestPerformance_Benchmark(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping benchmark in short mode")
    }

    benchmarks := []struct {
        name string
        fn   func(b *testing.B)
    }{
        {"ItemRepository_Create", BenchmarkItemRepository_Create},
        {"ItemRepository_List", BenchmarkItemRepository_List},
        {"FeedRepository_Create", BenchmarkFeedRepository_Create},
        {"RateLimiter_Allow", BenchmarkRateLimiter_Allow},
    }

    for _, bm := range benchmarks {
        t.Run(bm.name, func(t *testing.T) {
            b := testing.Benchmark(func() {
                bm.fn(b)
            })

            // Benchmark should complete in reasonable time
            if b.T > 1*time.Second {
                t.Errorf("Benchmark too slow: %v", b.T)
            }

            // Verify performance doesn't degrade
            if b.N > 1000 && b.T > 2*time.Second {
                t.Errorf("Performance regression: %v for %d iterations", b.T, b.N)
            }
        })
    }
}

func BenchmarkItemRepository_Create(b *testing.B) {
    db := setupItemDB(b)
    repo := NewItemRepository(db)

    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        item := &model.Item{
            FeedID: "feed-1",
            GUID:   fmt.Sprintf("guid-%d", i),
            Title:  fmt.Sprintf("Item %d", i),
        }
        item.GenerateID()
        repo.Create(ctx, item)
    }
}
```

**Recommendation:**
1. Add benchmark tests for critical paths
2. Monitor test execution time
3. Optimize slow tests
4. Use `t.Parallel()` where appropriate

---

### 8.2 Parallel Test Execution

**Severity:** Low
**Estimated Impact:** 2-5x faster test execution
**Location:** All tests

#### Issue
Most tests run sequentially. No parallelization optimization.

**Current State:**
```go
// All tests run sequentially
func TestItemRepository_Create(t *testing.T) { ... }
func TestItemRepository_List(t *testing.T) { ... }
```

**Optimization:**
```go
func TestItemRepository_Create(t *testing.T) {
    t.Parallel() // Run in parallel with other tests
    // ...
}

func TestItemRepository_List(t *testing.T) {
    t.Parallel()
    // ...
}
```

**Recommendation:**
1. Add `t.Parallel()` to independent tests
2. Use separate databases for parallel tests
3. Verify no race conditions with parallel execution
4. Measure speedup from parallelization

---

### 8.3 Slow Test Identification

**Severity:** Low
**Estimated Impact:** Wastes developer time, slows CI
**Location:** All tests

#### Issue
No mechanism to identify and track slow tests.

**Missing Test:**
```go
// test-registry_test.go
var slowTests = map[string]time.Duration{}

func trackSlowTest(name string) func() {
    start := time.Now()
    return func() {
        duration := time.Since(start)
        if duration > 500*time.Millisecond {
            slowTests[name] = duration
        }
    }
}

func TestSlowTests_Report(t *testing.T) {
    if len(slowTests) == 0 {
        return
    }

    t.Log("Slow tests detected (>500ms):")
    for name, duration := range slowTests {
        t.Logf("  %s: %v", name, duration)
    }

    // Fail if too many slow tests
    if len(slowTests) > 5 {
        t.Errorf("Too many slow tests: %d", len(slowTests))
    }
}

// Usage in tests:
func TestItemRepository_List(t *testing.T) {
    defer trackSlowTest("TestItemRepository_List")()
    // ... test code
}
```

**Recommendation:**
1. Add slow test tracking
2. Set test time limits
3. Optimize slow tests
4. Document expected test durations

---

## 9. Performance Summary

### 9.1 Critical Issues (Immediate Action Required)

| Issue | Impact | Location | Priority |
|-------|--------|-----------|----------|
| No N+1 query detection | 200-500ms latency | All repository tests | P0 |
| No large dataset tests | 5-10s timeout | `item_repository_test.go` | P0 |
| No race condition tests | Data corruption | All tests | P0 |
| No deadlock tests | System hang | All tests | P0 |
| No concurrent refresh tests | 10-60s delays | `refresh_worker_test.go` | P0 |

### 9.2 High Priority Issues

| Issue | Impact | Location | Priority |
|-------|--------|-----------|----------|
| No connection pool testing | Connection exhaustion | `database_test.go` | P1 |
| No large list rendering tests | 1-5s render time | `ItemList.test.tsx` | P1 |
| No virtualization | 5-10s render time | `ItemList.test.tsx` | P1 |
| No memory leak tests | OOM crashes | All tests | P1 |
| No slow network tests | 5-30s timeout | `refresh_worker_service_test.go` | P1 |

### 9.3 Medium Priority Issues

| Issue | Impact | Location | Priority |
|-------|--------|-----------|----------|
| No transaction isolation tests | Data corruption | All tests | P2 |
| No pagination performance tests | 1-5s delay | `item_repository_test.go` | P2 |
| No cache invalidation tests | Stale data | Not implemented | P2 |
| No bundle size tests | 2-5s load time | Frontend tests | P2 |
| No graceful degradation tests | Poor UX | All tests | P2 |

### 9.4 Low Priority Issues

| Issue | Impact | Location | Priority |
|-------|--------|-----------|----------|
| No test speed tracking | Slower CI | All tests | P3 |
| No parallel test execution | 2-5x slower tests | All tests | P3 |
| No unnecessary re-render tests | 100-500ms latency | Frontend tests | P3 |

---

## 10. Recommendations

### 10.1 Immediate Actions (Week 1-2)

1. **Add N+1 Query Detection**
   - Implement query counting in tests
   - Add tests with 100+ records
   - Run in CI pipeline

2. **Add Race Condition Testing**
   - Run all tests with `go test -race`
   - Add explicit concurrency tests
   - Fix any detected race conditions

3. **Add Large Dataset Tests**
   - Create test data factories for 100, 1000, 10000 records
   - Test pagination at large offsets
   - Measure query performance

### 10.2 Short-term Actions (Week 3-4)

4. **Add Connection Pool Tests**
   - Test with limited pool size
   - Verify concurrent access
   - Test pool exhaustion recovery

5. **Add Large List Rendering Tests**
   - Test frontend with 100+ items
   - Measure render time
   - Consider virtualization

6. **Add Memory Leak Tests**
   - Use `runtime.MemStats`
   - Test long-running operations
   - Monitor heap allocation

### 10.3 Medium-term Actions (Month 2)

7. **Implement Virtualization**
   - Use react-window or react-virtual
   - Test scroll performance
   - Verify only visible items render

8. **Add Caching Layer**
   - Implement Redis cache
   - Add cache hit/miss tests
   - Test cache invalidation

9. **Add Load Testing**
   - Use k6 or JMeter
   - Simulate 100+ concurrent users
   - Test API endpoints under load

### 10.4 Long-term Actions (Month 3+)

10. **Add Performance Monitoring**
    - Integrate APM (Datadog, New Relic)
    - Set up performance dashboards
    - Add SLO monitoring

11. **Optimize Critical Paths**
    - Profile hotspots with pprof
    - Optimize database queries
    - Implement caching strategies

12. **Add Performance Regression Tests**
    - Benchmark critical operations
    - Track performance over time
    - Alert on regressions

---

## 11. Recommended SLOs and Budgets

### 11.1 API Performance Budgets

| Endpoint | P50 | P95 | P99 | Timeout |
|----------|-----|-----|-----|---------|
| GET /api/v1/feeds | 50ms | 100ms | 200ms | 1s |
| GET /api/v1/items | 100ms | 200ms | 500ms | 2s |
| POST /api/v1/feeds | 200ms | 500ms | 1s | 5s |
| PUT /api/v1/items/:id/star | 50ms | 100ms | 200ms | 1s |

### 11.2 Database Performance Budgets

| Operation | Target | Max |
|-----------|--------|-----|
| Single record read | 10ms | 50ms |
| List 20 items | 20ms | 100ms |
| Batch insert (100 items) | 100ms | 500ms |
| Count query | 10ms | 50ms |

### 11.3 Frontend Performance Budgets

| Metric | Target | Max |
|--------|--------|-----|
| Initial page load | 2s | 5s |
| First contentful paint | 1s | 2s |
| Time to interactive | 3s | 5s |
| List render (100 items) | 100ms | 500ms |
| Interaction response | 50ms | 200ms |

### 11.4 Scalability Targets

| Metric | Target |
|--------|--------|
| Concurrent users | 1000 |
| Requests per second | 1000 |
| Database connections | 100 |
| Memory per instance | 1GB |
| CPU utilization | 70% |

---

## 12. Conclusion

The oReader test suite provides excellent functional coverage but has **critical gaps in performance testing**. The absence of load testing, large dataset tests, and concurrency verification means production performance issues will likely go undetected until they impact users.

**Key Takeaways:**

1. **Database performance is untested** - No N+1 query detection, no large dataset tests, no connection pool testing
2. **Concurrency is minimally tested** - Limited race condition tests, no deadlock testing, no concurrent refresh tests
3. **Frontend performance is untested** - No large list rendering tests, no virtualization, no bundle size monitoring
4. **No scalability verification** - No horizontal scaling tests, no resource limit tests, no graceful degradation tests

**Next Steps:**

1. Implement the top 3 priority optimizations immediately
2. Add performance testing to CI/CD pipeline
3. Establish performance budgets and SLOs
4. Monitor performance in production and iterate

By addressing these gaps, the test suite will provide confidence that the application performs well under load and scales as the user base grows.

---

## Appendix A: Test Coverage Summary

### Go Backend Tests

| Test File | Tests | Performance Tests | Missing Performance Tests |
|-----------|-------|------------------|-------------------------|
| `refresh_worker_service_test.go` | 10 | 1 (concurrent) | N+1, large dataset, memory leak |
| `refresh_worker_test.go` | 10 | 0 | Slow network, concurrent refreshes |
| `feed_repository_test.go` | 8 | 0 | Large dataset, pagination, pool |
| `item_repository_test.go` | 8 | 0 | N+1, large dataset, pagination |
| `database_test.go` | 2 | 1 (basic) | Pool exhaustion, concurrent access |
| `ratelimit_test.go` | 5 | 1 (concurrent) | Large dataset, hit rate |

### TypeScript Frontend Tests

| Test File | Tests | Performance Tests | Missing Performance Tests |
|-----------|-------|------------------|-------------------------|
| `useAuth.test.tsx` | 9 | 0 | Loading state, error handling |
| `authStore.test.ts` | 8 | 0 | Memory leak, large state |
| `ItemList.test.tsx` | 10 | 0 | Large list, re-render, virtualization |
| `useItems.test.tsx` | ~5 | 0 | Pagination, caching, error handling |

---

## Appendix B: Tools and Libraries

### Recommended Performance Testing Tools

**Go Backend:**
- `go test -race` - Race condition detection
- `testing/benchmark` - Benchmark tests
- `net/http/httptest` - HTTP mocking
- `github.com/stretchr/testify` - Assertions
- `github.com/prometheus/client_golang` - Metrics

**TypeScript Frontend:**
- `@testing-library/react` - Component testing
- `vitest` - Test runner
- `msw` - API mocking
- `react-window` or `react-virtual` - Virtualization
- `@tanstack/react-query` - Query caching

**Load Testing:**
- `k6` - Load testing
- `JMeter` - Load testing
- `Locust` - Load testing
- `Artillery` - Load testing

**Performance Monitoring:**
- `pprof` - Go profiling
- `React DevTools Profiler` - React profiling
- `Lighthouse` - Frontend performance
- `Datadog` / `New Relic` - APM

---

**End of Analysis**
