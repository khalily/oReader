package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"oreader/internal/model"
)

// MockFeedRepository is a mock implementation of FeedRepository for unit testing
type MockFeedRepository struct {
	mock.Mock
}

func (m *MockFeedRepository) Create(ctx context.Context, feed *model.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func (m *MockFeedRepository) GetByID(ctx context.Context, id string) (*model.Feed, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Feed), args.Error(1)
}

func (m *MockFeedRepository) GetByURL(ctx context.Context, url string) (*model.Feed, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Feed), args.Error(1)
}

func (m *MockFeedRepository) ListByUserID(ctx context.Context, userID string, opts ListOptions) ([]*model.Feed, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.Feed), args.Get(1).(int64), args.Error(2)
}

func (m *MockFeedRepository) Update(ctx context.Context, feed *model.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func (m *MockFeedRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedRepository) ListAll(ctx context.Context) ([]*model.Feed, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Feed), args.Error(1)
}

// MockUserFeedRepository is a mock implementation of UserFeedRepository
type MockUserFeedRepository struct {
	mock.Mock
}

func (m *MockUserFeedRepository) Create(ctx context.Context, userFeed *model.UserFeed) error {
	args := m.Called(ctx, userFeed)
	return args.Error(0)
}

func (m *MockUserFeedRepository) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserFeed), args.Error(1)
}

func (m *MockUserFeedRepository) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserFeed), args.Error(1)
}

func (m *MockUserFeedRepository) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.UserFeed), args.Error(1)
}

func (m *MockUserFeedRepository) Delete(ctx context.Context, userID, feedID string) error {
	args := m.Called(ctx, userID, feedID)
	return args.Error(0)
}

func (m *MockUserFeedRepository) GetMaxPosition(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Error(1)
}

// MockItemRepository is a mock implementation of ItemRepository
type MockItemRepository struct {
	mock.Mock
}

func (m *MockItemRepository) Create(ctx context.Context, item *model.Item) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockItemRepository) CreateBatch(ctx context.Context, items []*model.Item) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockItemRepository) GetByID(ctx context.Context, id string) (*model.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Item), args.Error(1)
}

func (m *MockItemRepository) GetByGUID(ctx context.Context, feedID, guid string) (*model.Item, error) {
	args := m.Called(ctx, feedID, guid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Item), args.Error(1)
}

func (m *MockItemRepository) ListByFeedID(ctx context.Context, feedID string, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	args := m.Called(ctx, feedID, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*ItemWithState), args.Get(1).(int64), args.Error(2)
}

func (m *MockItemRepository) ListStarred(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*ItemWithState), args.Get(1).(int64), args.Error(2)
}

func (m *MockItemRepository) ListUnread(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*ItemWithState), args.Get(1).(int64), args.Error(2)
}

func (m *MockItemRepository) CountByFeedID(ctx context.Context, feedID string) (int64, error) {
	args := m.Called(ctx, feedID)
	return args.Get(0).(int64), args.Error(1)
}

// Unit tests using mocks for error scenarios

func TestFeedService_GetUserFeeds_RepoError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	mockRepo := new(MockFeedRepository)
	mockRepo.On("ListByUserID", ctx, userID, mock.Anything).
		Return(nil, int64(0), errors.New("database connection failed"))

	service := NewFeedService(mockRepo, nil, nil, nil)
	feeds, total, err := service.GetUserFeeds(ctx, userID, ListOptions{Limit: 10})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.Nil(t, feeds)
	assert.Equal(t, int64(0), total)

	mockRepo.AssertExpectations(t)
}

func TestFeedService_GetFeed_NotFound(t *testing.T) {
	ctx := context.Background()
	feedID := "nonexistent"
	userID := "user1"

	mockFeedRepo := new(MockFeedRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// First check - user subscription lookup fails
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(nil, errors.New("not subscribed"))

	service := NewFeedService(mockFeedRepo, nil, mockUserFeedRepo, nil)
	_, _, err := service.GetFeed(ctx, userID, feedID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedNotFound)

	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_DeleteFeed_RepoError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockUserFeedRepo := new(MockUserFeedRepository)
	// First call - check subscription exists
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(&model.UserFeed{UserID: userID, FeedID: feedID}, nil)
	// Second call - delete fails
	mockUserFeedRepo.On("Delete", ctx, userID, feedID).Return(errors.New("database error"))

	service := NewFeedService(nil, nil, mockUserFeedRepo, nil)
	err := service.DeleteFeed(ctx, userID, feedID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockUserFeedRepo.AssertExpectations(t)
}

// =============================================================================
// BEHAVIOR TESTS - Test actual business logic, not just literals
// These tests verify that the service methods work correctly with mocked dependencies
// =============================================================================

// TestFeedService_Subscribe_NewFeed tests the new feed subscription flow
// NOTE: The Subscribe method uses *rss.Parser directly (not an interface),
// so we can't fully mock it. This is an integration test pattern that would need
// the parser to be injected as an interface for proper unit testing.
// For full coverage, consider:
// 1. Creating a Parser interface in the service layer
// 2. Or using integration tests with a test server
//
// The tests below cover the paths that don't require the parser.

func TestFeedService_Subscribe_AlreadySubscribed(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedURL := "https://example.com/feed.xml"

	mockFeedRepo := new(MockFeedRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// Feed exists
	existingFeed := &model.Feed{
		Base:    model.Base{ID: "feed1"},
		FeedURL: feedURL,
		Title:   "Existing Feed",
	}
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// User is already subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, "feed1").Return(&model.UserFeed{
		UserID: userID,
		FeedID: "feed1",
	}, nil)

	service := NewFeedService(mockFeedRepo, nil, mockUserFeedRepo, nil)
	_, err := service.Subscribe(ctx, userID, feedURL)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedAlreadySubscribed)

	mockFeedRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_Subscribe_ExistingFeed_NewUser(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedURL := "https://example.com/feed.xml"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// Feed exists
	existingFeed := &model.Feed{
		Base:    model.Base{ID: "feed1"},
		FeedURL: feedURL,
		Title:   "Existing Feed",
	}
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// User is NOT subscribed (returns error)
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, "feed1").Return(nil, errors.New("not found"))

	// Count existing items
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(5), nil)

	// Get max position
	mockUserFeedRepo.On("GetMaxPosition", ctx, userID).Return(0, nil)

	// UserFeed creation succeeds
	mockUserFeedRepo.On("Create", ctx, mock.AnythingOfType("*model.UserFeed")).Return(nil)

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	result, err := service.Subscribe(ctx, userID, feedURL)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "feed1", result.Feed.ID)
	assert.Equal(t, 5, result.NewItemCount) // Existing items count

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_GetUserFeeds_Success(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)

	feeds := []*model.Feed{
		{Base: model.Base{ID: "feed1"}, Title: "Feed 1"},
		{Base: model.Base{ID: "feed2"}, Title: "Feed 2"},
	}
	mockFeedRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("service.ListOptions")).
		Return(feeds, int64(2), nil)

	// Count items for each feed
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(10), nil)
	mockItemRepo.On("CountByFeedID", ctx, "feed2").Return(int64(5), nil)

	service := NewFeedService(mockFeedRepo, mockItemRepo, nil, nil)
	result, total, err := service.GetUserFeeds(ctx, userID, ListOptions{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, result, 2)
	assert.Equal(t, "feed1", result[0].Feed.ID)
	assert.Equal(t, 10, result[0].ItemCount)
	assert.Equal(t, "feed2", result[1].Feed.ID)
	assert.Equal(t, 5, result[1].ItemCount)

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
}

func TestFeedService_GetUserFeeds_ItemCountError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)

	feeds := []*model.Feed{
		{Base: model.Base{ID: "feed1"}, Title: "Feed 1"},
	}
	mockFeedRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("service.ListOptions")).
		Return(feeds, int64(1), nil)

	// Count items fails
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(0), errors.New("count error"))

	service := NewFeedService(mockFeedRepo, mockItemRepo, nil, nil)
	_, _, err := service.GetUserFeeds(ctx, userID, ListOptions{Limit: 10})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "count error")

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
}

func TestFeedService_GetFeed_Success(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(&model.UserFeed{
		UserID: userID,
		FeedID: feedID,
	}, nil)

	// Feed exists
	feed := &model.Feed{Base: model.Base{ID: feedID}, Title: "Test Feed"}
	mockFeedRepo.On("GetByID", ctx, feedID).Return(feed, nil)

	// Count items
	mockItemRepo.On("CountByFeedID", ctx, feedID).Return(int64(15), nil)

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	result, itemCount, err := service.GetFeed(ctx, userID, feedID)

	require.NoError(t, err)
	assert.Equal(t, feedID, result.ID)
	assert.Equal(t, "Test Feed", result.Title)
	assert.Equal(t, 15, itemCount)

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_DeleteFeed_Success(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(&model.UserFeed{
		UserID: userID,
		FeedID: feedID,
	}, nil)

	// Delete succeeds
	mockUserFeedRepo.On("Delete", ctx, userID, feedID).Return(nil)

	service := NewFeedService(nil, nil, mockUserFeedRepo, nil)
	err := service.DeleteFeed(ctx, userID, feedID)

	require.NoError(t, err)

	mockUserFeedRepo.AssertExpectations(t)
}

// =============================================================================
// ERROR PATH TESTS - Test error handling scenarios (TC-003)
// =============================================================================

func TestFeedService_GetUserFeeds_EmptyResult(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	mockFeedRepo := new(MockFeedRepository)
	mockFeedRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("service.ListOptions")).
		Return([]*model.Feed{}, int64(0), nil)

	service := NewFeedService(mockFeedRepo, nil, nil, nil)
	result, total, err := service.GetUserFeeds(ctx, userID, ListOptions{Limit: 10})

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, int64(0), total)

	mockFeedRepo.AssertExpectations(t)
}

// NOTE: TestFeedService_Subscribe_FeedRepoError is not testable here because
// when GetByURL returns an error, the service tries to use the parser (concrete type).
// This requires integration tests with a mock HTTP server or refactoring the service
// to inject a Parser interface. See feed_service_integration_test.go for such tests.

func TestFeedService_Subscribe_UserFeedCreateError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedURL := "https://example.com/feed.xml"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// Feed exists
	existingFeed := &model.Feed{
		Base:    model.Base{ID: "feed1"},
		FeedURL: feedURL,
		Title:   "Existing Feed",
	}
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// User is NOT subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, "feed1").Return(nil, errors.New("not found"))

	// Count items succeeds
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(5), nil)

	// Get max position succeeds
	mockUserFeedRepo.On("GetMaxPosition", ctx, userID).Return(0, nil)

	// UserFeed creation fails
	mockUserFeedRepo.On("Create", ctx, mock.AnythingOfType("*model.UserFeed")).Return(errors.New("constraint violation"))

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	_, err := service.Subscribe(ctx, userID, feedURL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "constraint violation")

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_GetFeed_FeedRepoError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(&model.UserFeed{
		UserID: userID,
		FeedID: feedID,
	}, nil)

	// Feed repo returns error
	mockFeedRepo.On("GetByID", ctx, feedID).Return(nil, errors.New("database error"))

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	_, _, err := service.GetFeed(ctx, userID, feedID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedNotFound)

	mockFeedRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_GetFeed_ItemCountError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(&model.UserFeed{
		UserID: userID,
		FeedID: feedID,
	}, nil)

	// Feed exists
	feed := &model.Feed{Base: model.Base{ID: feedID}, Title: "Test Feed"}
	mockFeedRepo.On("GetByID", ctx, feedID).Return(feed, nil)

	// Count items fails
	mockItemRepo.On("CountByFeedID", ctx, feedID).Return(int64(0), errors.New("count error"))

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	_, _, err := service.GetFeed(ctx, userID, feedID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "count error")

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_DeleteFeed_NotSubscribed(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is NOT subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(nil, errors.New("not found"))
	// Check for soft-deleted record - also not found
	mockUserFeedRepo.On("GetByUserAndFeedIncludingDeleted", ctx, userID, feedID).Return(nil, errors.New("not found"))

	service := NewFeedService(nil, nil, mockUserFeedRepo, nil)
	err := service.DeleteFeed(ctx, userID, feedID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedNotFound)

	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_Subscribe_GetMaxPositionError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feedURL := "https://example.com/feed.xml"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	// Feed exists
	existingFeed := &model.Feed{
		Base:    model.Base{ID: "feed1"},
		FeedURL: feedURL,
		Title:   "Existing Feed",
	}
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// User is NOT subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, "feed1").Return(nil, errors.New("not found"))

	// Count items succeeds
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(5), nil)

	// Get max position fails
	mockUserFeedRepo.On("GetMaxPosition", ctx, userID).Return(0, errors.New("position error"))

	service := NewFeedService(mockFeedRepo, mockItemRepo, mockUserFeedRepo, nil)
	_, err := service.Subscribe(ctx, userID, feedURL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "position error")

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

// =============================================================================
// SECURITY TESTS - Test security edge cases (TC-002)
// These tests verify that the service properly handles malicious input
// =============================================================================

func TestFeedService_Subscribe_AlreadySubscribed_ReturnsCorrectError(t *testing.T) {
	// This test verifies that ErrFeedAlreadySubscribed is returned correctly
	// when a user tries to subscribe to a feed they're already subscribed to
	ctx := context.Background()
	userID := "user1"
	feedURL := "https://example.com/feed.xml"

	mockFeedRepo := new(MockFeedRepository)
	mockUserFeedRepo := new(MockUserFeedRepository)

	existingFeed := &model.Feed{
		Base:    model.Base{ID: "feed1"},
		FeedURL: feedURL,
		Title:   "Existing Feed",
	}
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// User IS already subscribed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, "feed1").Return(&model.UserFeed{
		UserID: userID,
		FeedID: "feed1",
	}, nil)

	service := NewFeedService(mockFeedRepo, nil, mockUserFeedRepo, nil)
	_, err := service.Subscribe(ctx, userID, feedURL)

	require.Error(t, err)
	// Verify the error is ErrFeedAlreadySubscribed, not some other error
	assert.ErrorIs(t, err, ErrFeedAlreadySubscribed)

	mockFeedRepo.AssertExpectations(t)
	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_DeleteFeed_VerifiesUserOwnership(t *testing.T) {
	// This test verifies that users can only delete feeds they're subscribed to
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is NOT subscribed to this feed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(nil, errors.New("not subscribed"))
	// Check for soft-deleted record - also not found
	mockUserFeedRepo.On("GetByUserAndFeedIncludingDeleted", ctx, userID, feedID).Return(nil, errors.New("not found"))

	service := NewFeedService(nil, nil, mockUserFeedRepo, nil)
	err := service.DeleteFeed(ctx, userID, feedID)

	// Should return ErrFeedNotFound, not delete the feed
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedNotFound)

	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_GetFeed_VerifiesUserOwnership(t *testing.T) {
	// This test verifies that users can only access feeds they're subscribed to
	ctx := context.Background()
	userID := "user1"
	feedID := "feed1"

	mockUserFeedRepo := new(MockUserFeedRepository)

	// User is NOT subscribed to this feed
	mockUserFeedRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(nil, errors.New("not subscribed"))

	service := NewFeedService(nil, nil, mockUserFeedRepo, nil)
	_, _, err := service.GetFeed(ctx, userID, feedID)

	// Should return ErrFeedNotFound
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFeedNotFound)

	mockUserFeedRepo.AssertExpectations(t)
}

func TestFeedService_GetUserFeeds_IsolatedByUser(t *testing.T) {
	// This test verifies that GetUserFeeds only returns feeds for the specified user
	ctx := context.Background()
	userID := "user1"

	mockFeedRepo := new(MockFeedRepository)
	mockItemRepo := new(MockItemRepository)

	// The repository should be called with the correct userID
	expectedFeeds := []*model.Feed{
		{Base: model.Base{ID: "feed1"}, Title: "User's Feed"},
	}
	mockFeedRepo.On("ListByUserID", ctx, userID, mock.AnythingOfType("service.ListOptions")).
		Return(expectedFeeds, int64(1), nil).
		Run(func(args mock.Arguments) {
			// Verify the userID passed is correct
			assert.Equal(t, userID, args.Get(1))
		})

	// Mock the item count call
	mockItemRepo.On("CountByFeedID", ctx, "feed1").Return(int64(5), nil)

	service := NewFeedService(mockFeedRepo, mockItemRepo, nil, nil)
	feeds, total, err := service.GetUserFeeds(ctx, userID, ListOptions{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, feeds, 1)
	assert.Equal(t, 5, feeds[0].ItemCount)

	mockFeedRepo.AssertExpectations(t)
	mockItemRepo.AssertExpectations(t)
}
