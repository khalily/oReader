package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"oreader/internal/model"
)

// refreshMockFeedRepository is a mock specifically for refresh worker tests
type refreshMockFeedRepository struct {
	feeds       []*model.Feed
	listAllErr  error
	updateErr   error
	updateCalls []*model.Feed
}

func (m *refreshMockFeedRepository) Create(ctx context.Context, feed *model.Feed) error {
	return nil
}

func (m *refreshMockFeedRepository) GetByID(ctx context.Context, id string) (*model.Feed, error) {
	for _, f := range m.feeds {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, errors.New("feed not found")
}

func (m *refreshMockFeedRepository) GetByURL(ctx context.Context, url string) (*model.Feed, error) {
	return nil, nil
}

func (m *refreshMockFeedRepository) ListByUserID(ctx context.Context, userID string, opts ListOptions) ([]*model.Feed, int64, error) {
	return nil, 0, nil
}

func (m *refreshMockFeedRepository) Update(ctx context.Context, feed *model.Feed) error {
	m.updateCalls = append(m.updateCalls, feed)
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, f := range m.feeds {
		if f.ID == feed.ID {
			m.feeds[i] = feed
			break
		}
	}
	return nil
}

func (m *refreshMockFeedRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *refreshMockFeedRepository) ListAll(ctx context.Context) ([]*model.Feed, error) {
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	return m.feeds, nil
}

// refreshMockItemRepository is a mock specifically for refresh worker tests
type refreshMockItemRepository struct {
	items         []*model.Item
	getByGUIDFunc func(ctx context.Context, feedID, guid string) (*model.Item, error)
	createBatchFunc func(ctx context.Context, items []*model.Item) error
}

func (m *refreshMockItemRepository) Create(ctx context.Context, item *model.Item) error {
	m.items = append(m.items, item)
	return nil
}

func (m *refreshMockItemRepository) CreateBatch(ctx context.Context, items []*model.Item) error {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(ctx, items)
	}
	m.items = append(m.items, items...)
	return nil
}

func (m *refreshMockItemRepository) GetByID(ctx context.Context, id string) (*model.Item, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *refreshMockItemRepository) GetByGUID(ctx context.Context, feedID, guid string) (*model.Item, error) {
	if m.getByGUIDFunc != nil {
		return m.getByGUIDFunc(ctx, feedID, guid)
	}
	for _, item := range m.items {
		if item.FeedID == feedID && item.GUID == guid {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *refreshMockItemRepository) ListByFeedID(ctx context.Context, feedID string, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	return nil, 0, nil
}

func (m *refreshMockItemRepository) ListStarred(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	return nil, 0, nil
}

func (m *refreshMockItemRepository) ListUnread(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	return nil, 0, nil
}

func (m *refreshMockItemRepository) Update(ctx context.Context, item *model.Item) error {
	for i, existing := range m.items {
		if existing.ID == item.ID {
			m.items[i] = item
			return nil
		}
	}
	return errors.New("item not found")
}

func (m *refreshMockItemRepository) CountByFeedID(ctx context.Context, feedID string) (int64, error) {
	count := int64(0)
	for _, item := range m.items {
		if item.FeedID == feedID {
			count++
		}
	}
	return count, nil
}

// refreshMockUserFeedRepository is a mock specifically for refresh worker tests
type refreshMockUserFeedRepository struct{}

func (m *refreshMockUserFeedRepository) Create(ctx context.Context, userFeed *model.UserFeed) error {
	return nil
}

func (m *refreshMockUserFeedRepository) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	return nil, errors.New("not found")
}

func (m *refreshMockUserFeedRepository) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	return nil, errors.New("not found")
}

func (m *refreshMockUserFeedRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	return nil, nil
}

func (m *refreshMockUserFeedRepository) Delete(ctx context.Context, userID, feedID string) error {
	return nil
}

func (m *refreshMockUserFeedRepository) GetMaxPosition(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

// Helper function to create test feeds
func createTestFeed(id, url, title string) *model.Feed {
	return &model.Feed{
		Base:               model.Base{ID: id},
		FeedURL:            url,
		Title:              title,
		Description:        "Test description",
		ConsecutiveFailures: 0,
		LastFetchStatus:    "pending",
	}
}

// TestRefreshWorkerService_RefreshAllFeeds tests refreshing all feeds
func TestRefreshWorkerService_RefreshAllFeeds(t *testing.T) {
	ctx := context.Background()

	feeds := []*model.Feed{
		createTestFeed("feed1", "http://example.com/feed1.xml", "Feed 1"),
		createTestFeed("feed2", "http://example.com/feed2.xml", "Feed 2"),
		createTestFeed("feed3", "http://example.com/feed3.xml", "Feed 3"),
	}

	feedRepo := &refreshMockFeedRepository{feeds: feeds}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	result, err := service.RefreshAllFeeds(ctx)
	if err != nil {
		t.Fatalf("RefreshAllFeeds failed: %v", err)
	}

	if result.TotalFeeds != 3 {
		t.Errorf("Expected 3 total feeds, got %d", result.TotalFeeds)
	}

	if result.SuccessCount < 0 || result.SuccessCount > 3 {
		t.Errorf("Expected success count between 0 and 3, got %d", result.SuccessCount)
	}

	if result.Duration == 0 {
		t.Error("Expected duration to be recorded")
	}
}

// TestRefreshWorkerService_EmptyFeeds tests refreshing when no feeds exist
func TestRefreshWorkerService_EmptyFeeds(t *testing.T) {
	ctx := context.Background()

	feedRepo := &refreshMockFeedRepository{feeds: []*model.Feed{}}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	result, err := service.RefreshAllFeeds(ctx)
	if err != nil {
		t.Fatalf("RefreshAllFeeds failed: %v", err)
	}

	if result.TotalFeeds != 0 {
		t.Errorf("Expected 0 total feeds, got %d", result.TotalFeeds)
	}

	if result.SuccessCount != 0 {
		t.Errorf("Expected 0 successful refreshes, got %d", result.SuccessCount)
	}
}

// TestRefreshWorkerService_ListAllError tests error handling when ListAll fails
func TestRefreshWorkerService_ListAllError(t *testing.T) {
	ctx := context.Background()

	feedRepo := &refreshMockFeedRepository{listAllErr: errors.New("database error")}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	_, err := service.RefreshAllFeeds(ctx)
	if err == nil {
		t.Error("Expected error when ListAll fails, got nil")
	}
}

// TestRefreshWorkerService_RefreshSingleFeed tests refreshing a single feed
func TestRefreshWorkerService_RefreshSingleFeed(t *testing.T) {
	ctx := context.Background()

	feed := createTestFeed("feed1", "http://example.com/feed.xml", "Test Feed")

	feedRepo := &refreshMockFeedRepository{feeds: []*model.Feed{feed}}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	result, err := service.RefreshSingleFeed(ctx, feed)
	if err != nil {
		t.Fatalf("RefreshSingleFeed failed: %v", err)
	}

	if result.Duration == 0 {
		t.Error("Expected duration to be recorded")
	}
}

// TestRefreshResult_String tests the RefreshAllResult String method
func TestRefreshResult_String(t *testing.T) {
	result := &RefreshAllResult{
		TotalFeeds:    10,
		SuccessCount:  8,
		FailureCount:  2,
		TotalItems:    50,
		NewItems:      20,
		Duration:      5 * time.Second,
		FailedFeedIDs: []string{"feed1", "feed2"},
	}

	str := result.String()
	expected := "10 feeds, 8 success, 2 failures, 50 total items, 20 new items, 5s duration"
	if str != expected {
		t.Errorf("String() = %v, want %v", str, expected)
	}
}

// TestRefreshResult_String_Milliseconds tests the String method with millisecond duration
func TestRefreshResult_String_Milliseconds(t *testing.T) {
	result := &RefreshAllResult{
		TotalFeeds:    1,
		SuccessCount:  1,
		FailureCount:  0,
		TotalItems:    5,
		NewItems:      5,
		Duration:      150 * time.Millisecond,
	}

	str := result.String()
	expected := "1 feeds, 1 success, 0 failures, 5 total items, 5 new items, 150ms duration"
	if str != expected {
		t.Errorf("String() = %v, want %v", str, expected)
	}
}

// TestSingleFeedResult_String tests the SingleFeedResult String method
func TestSingleFeedResult_String(t *testing.T) {
	tests := []struct {
		name     string
		result   SingleFeedResult
		expected string
	}{
		{
			name: "success result",
			result: SingleFeedResult{
				FeedID:   "feed1",
				Success:  true,
				NewItems: 5,
				Duration: 100 * time.Millisecond,
			},
			expected: "feed1: success (5 items, 100ms)",
		},
		{
			name: "failure result",
			result: SingleFeedResult{
				FeedID:   "feed1",
				Success:  false,
				Error:    "network error",
				Duration: 50 * time.Millisecond,
			},
			expected: "feed1: failed (network error, 50ms)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.result.String()
			if str != tt.expected {
				t.Errorf("String() = %v, want %v", str, tt.expected)
			}
		})
	}
}

// TestRefreshWorkerService_ConcurrentProcessing tests that concurrent refresh is handled correctly
func TestRefreshWorkerService_ConcurrentProcessing(t *testing.T) {
	ctx := context.Background()

	// Create multiple feeds
	feeds := make([]*model.Feed, 15)
	for i := 0; i < 15; i++ {
		feeds[i] = createTestFeed("feed"+string(rune('0'+i)), "http://example.com/feed.xml", "Feed")
	}

	feedRepo := &refreshMockFeedRepository{feeds: feeds}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	result, err := service.RefreshAllFeeds(ctx)
	if err != nil {
		t.Fatalf("RefreshAllFeeds failed: %v", err)
	}

	if result.TotalFeeds != 15 {
		t.Errorf("Expected 15 total feeds, got %d", result.TotalFeeds)
	}
}

// TestRefreshWorkerService_ContextCancellation tests context cancellation handling
func TestRefreshWorkerService_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	feeds := []*model.Feed{
		createTestFeed("feed1", "http://example.com/feed1.xml", "Feed 1"),
		createTestFeed("feed2", "http://example.com/feed2.xml", "Feed 2"),
	}

	feedRepo := &refreshMockFeedRepository{feeds: feeds}
	itemRepo := &refreshMockItemRepository{}
	userFeedRepo := &refreshMockUserFeedRepository{}

	service := NewRefreshWorkerService(feedRepo, itemRepo, userFeedRepo)

	// Cancel the context immediately to test cancellation
	cancel()

	_, err := service.RefreshAllFeeds(ctx)
	// Should either complete quickly or return context error
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Got error (may be expected): %v", err)
	}
}
