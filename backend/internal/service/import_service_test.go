package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/khalily/oreader/internal/model"
)

// MockImportJobRepository is a mock implementation of ImportJobRepository
type MockImportJobRepository struct {
	mock.Mock
}

func (m *MockImportJobRepository) Create(ctx context.Context, job *model.ImportJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockImportJobRepository) GetByID(ctx context.Context, id string) (*model.ImportJob, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ImportJob), args.Error(1)
}

func (m *MockImportJobRepository) GetByUserID(ctx context.Context, userID string) ([]*model.ImportJob, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ImportJob), args.Error(1)
}

func (m *MockImportJobRepository) Update(ctx context.Context, job *model.ImportJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

// MockFeedService is a mock implementation of FeedService for import testing
type MockFeedServiceForImport struct {
	mock.Mock
}

func (m *MockFeedServiceForImport) Subscribe(ctx context.Context, userID, feedURL string) (*SubscribeResult, error) {
	args := m.Called(ctx, userID, feedURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SubscribeResult), args.Error(1)
}

func (m *MockFeedServiceForImport) GetUserFeeds(ctx context.Context, userID string, opts ListOptions) ([]*model.Feed, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.Feed), args.Get(1).(int64), args.Error(2)
}

func (m *MockFeedServiceForImport) GetFeed(ctx context.Context, userID, feedID string) (*model.Feed, int, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int), args.Error(2)
	}
	return args.Get(0).(*model.Feed), args.Get(1).(int), args.Error(2)
}

func (m *MockFeedServiceForImport) DeleteFeed(ctx context.Context, userID, feedID string) error {
	args := m.Called(ctx, userID, feedID)
	return args.Error(0)
}

func (m *MockFeedServiceForImport) RefreshFeed(ctx context.Context, userID, feedID string) (*RefreshResult, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RefreshResult), args.Error(1)
}

func (m *MockFeedServiceForImport) RefreshAllFeeds(ctx context.Context, userID string) (int, int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int), args.Get(1).(int), args.Error(2)
}

func (m *MockFeedServiceForImport) GetFeedWithItemCount(ctx context.Context, userID, feedID string) (*FeedWithItemCount, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeedWithItemCount), args.Error(1)
}

// Unit tests for ImportService

func TestImportService_ParseOPML_Valid(t *testing.T) {
	ctx := context.Background()

	// Create service with nil dependencies (ParseOPML doesn't use them)
	service := NewImportService(nil, nil, nil, nil)

	// Note: OPML parser requires type="rss" attribute
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>My Feeds</title>
  </head>
  <body>
    <outline type="rss" text="Tech Blog" xmlUrl="https://example.com/tech.xml" htmlUrl="https://example.com"/>
    <outline type="rss" text="News" xmlUrl="https://example.com/news.xml" htmlUrl="https://example.com/news"/>
  </body>
</opml>`

	feeds, err := service.ParseOPML(ctx, opmlContent)

	require.NoError(t, err)
	assert.Len(t, feeds, 2)
	assert.Equal(t, "Tech Blog", feeds[0].Title)
	assert.Equal(t, "https://example.com/tech.xml", feeds[0].FeedURL)
	assert.Equal(t, "News", feeds[1].Title)
}

func TestImportService_ParseOPML_InvalidXML(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	invalidOPML := `this is not valid xml`

	feeds, err := service.ParseOPML(ctx, invalidOPML)

	require.Error(t, err)
	assert.Nil(t, feeds)
	assert.Contains(t, err.Error(), "failed to parse OPML")
}

func TestImportService_ParseOPML_Empty(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	feeds, err := service.ParseOPML(ctx, "")

	require.Error(t, err)
	assert.Nil(t, feeds)
}

func TestImportService_StartImport_RepoError(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feeds := []*FeedInfo{
		{Title: "Feed 1", FeedURL: "https://example.com/feed1.xml"},
	}

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("Create", ctx, mock.AnythingOfType("*model.ImportJob")).
		Return(errors.New("database error"))

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.StartImport(ctx, userID, feeds)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create import job")
	assert.Nil(t, job)

	mockJobRepo.AssertExpectations(t)
}

func TestImportService_GetJobStatus_Found(t *testing.T) {
	ctx := context.Background()
	jobID := "job1"

	expectedJob := &model.ImportJob{
		Base:       model.Base{ID: jobID},
		UserID:     "user1",
		Status:     model.ImportJobStatusCompleted,
		TotalFeeds: 5,
		Processed:  5,
		Failed:     0,
	}

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("GetByID", ctx, jobID).Return(expectedJob, nil)

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.GetJobStatus(ctx, jobID)

	require.NoError(t, err)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, model.ImportJobStatusCompleted, job.Status)
	assert.Equal(t, 5, job.TotalFeeds)

	mockJobRepo.AssertExpectations(t)
}

func TestImportService_GetJobStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	jobID := "nonexistent"

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("GetByID", ctx, jobID).Return(nil, errors.New("not found"))

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.GetJobStatus(ctx, jobID)

	require.Error(t, err)
	assert.Nil(t, job)

	mockJobRepo.AssertExpectations(t)
}

// =============================================================================
// ERROR PATH TESTS - Test error handling scenarios (TC-003)
// =============================================================================

func TestImportService_ParseOPML_NoFeeds(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// OPML with no feed outlines
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Empty Feeds</title>
  </head>
  <body>
  </body>
</opml>`

	feeds, err := service.ParseOPML(ctx, opmlContent)

	require.NoError(t, err)
	assert.Empty(t, feeds)
}

func TestImportService_ParseOPML_NoRSSType(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// OPML with outlines but no type="rss" attribute
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Non-RSS Feeds</title>
  </head>
  <body>
    <outline text="Folder">
      <outline text="Link" url="https://example.com"/>
    </outline>
  </body>
</opml>`

	feeds, err := service.ParseOPML(ctx, opmlContent)

	require.NoError(t, err)
	assert.Empty(t, feeds) // No RSS feeds found
}

func TestImportService_ParseOPML_NestedFolders(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// OPML with nested folder structure
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Nested Feeds</title>
  </head>
  <body>
    <outline text="Tech">
      <outline type="rss" text="Blog" xmlUrl="https://example.com/tech.xml"/>
    </outline>
    <outline type="rss" text="News" xmlUrl="https://example.com/news.xml"/>
  </body>
</opml>`

	feeds, err := service.ParseOPML(ctx, opmlContent)

	require.NoError(t, err)
	assert.Len(t, feeds, 2) // Both feeds should be found
}

func TestImportService_StartImport_EmptyFeedList(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	feeds := []*FeedInfo{} // Empty feed list

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("Create", ctx, mock.AnythingOfType("*model.ImportJob")).Return(nil)
	// Mock the async goroutine calls - processImportAsync calls GetByID then Update
	mockJobRepo.On("GetByID", mock.Anything, mock.AnythingOfType("string")).Return(&model.ImportJob{
		Base:       model.Base{ID: "test-job-id"},
		UserID:     userID,
		Status:     model.ImportJobStatusPending,
		TotalFeeds: 0,
	}, nil).Maybe()
	mockJobRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.ImportJob")).Return(nil).Maybe()

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.StartImport(ctx, userID, feeds)

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, 0, job.TotalFeeds)

	// Give the async goroutine time to complete (it should finish quickly with empty list)
	time.Sleep(100 * time.Millisecond)
}

func TestImportService_StartImport_NilFeedList(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("Create", ctx, mock.AnythingOfType("*model.ImportJob")).Return(nil)
	// Mock the async goroutine calls
	mockJobRepo.On("GetByID", mock.Anything, mock.AnythingOfType("string")).Return(&model.ImportJob{
		Base:       model.Base{ID: "test-job-id"},
		UserID:     userID,
		Status:     model.ImportJobStatusPending,
		TotalFeeds: 0,
	}, nil).Maybe()
	mockJobRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.ImportJob")).Return(nil).Maybe()

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.StartImport(ctx, userID, nil)

	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, 0, job.TotalFeeds)

	// Give the async goroutine time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestImportService_GetJobStatus_DatabaseError(t *testing.T) {
	ctx := context.Background()
	jobID := "job1"

	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("GetByID", ctx, jobID).Return(nil, errors.New("database connection failed"))

	service := NewImportService(nil, mockJobRepo, nil, nil)
	job, err := service.GetJobStatus(ctx, jobID)

	require.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "failed to get job status")

	mockJobRepo.AssertExpectations(t)
}

func TestImportService_ProcessImport_NotImplemented(t *testing.T) {
	ctx := context.Background()
	jobID := "job1"

	// Need to provide a mock repo since ProcessImport calls GetByID first
	mockJobRepo := new(MockImportJobRepository)
	mockJobRepo.On("GetByID", ctx, jobID).Return(&model.ImportJob{
		Base:   model.Base{ID: jobID},
		Status: model.ImportJobStatusPending,
	}, nil)

	service := NewImportService(nil, mockJobRepo, nil, nil)
	err := service.ProcessImport(ctx, jobID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "use StartImport")

	mockJobRepo.AssertExpectations(t)
}

// =============================================================================
// SECURITY TESTS - Test security edge cases (TC-002)
// =============================================================================

func TestImportService_ParseOPML_XXEAttack(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// XXE attack attempt - try to read /etc/passwd
	xxePayload := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<opml version="1.0">
  <head>
    <title>XXE Attack</title>
  </head>
  <body>
    <outline type="rss" text="&xxe;" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`

	// The parser should either reject the XXE or not expand the entity
	feeds, err := service.ParseOPML(ctx, xxePayload)

	// Either an error is returned, or the entity is not expanded (safe)
	// The key is that /etc/passwd content should NOT appear in the result
	if err == nil {
		// If no error, verify the entity was not expanded
		for _, feed := range feeds {
			// The title should NOT contain actual /etc/passwd content
			assert.NotContains(t, feed.Title, "root:")
			assert.NotContains(t, feed.Title, "nobody:")
		}
	}
	// If error is returned, that's also acceptable (XXE blocked)
}

func TestImportService_ParseOPML_BillionLaughs(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// Billion laughs attack - tries to cause memory exhaustion
	billionLaughsPayload := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
]>
<opml version="1.0">
  <head>
    <title>&lol3;</title>
  </head>
  <body>
    <outline type="rss" text="Test" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`

	// The parser should either reject this or handle it safely
	// We don't test for specific behavior, just that it doesn't crash
	_, _ = service.ParseOPML(ctx, billionLaughsPayload)
	// Test passes if we reach here without panic/timeout
}

func TestImportService_ParseOPML_ExternalEntity(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// External entity attack - try to access network resources
	externalEntityPayload := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE foo [
  <!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">
]>
<opml version="1.0">
  <head>
    <title>External Entity</title>
  </head>
  <body>
    <outline type="rss" text="&xxe;" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`

	// The parser should not make network requests during parsing
	// Either reject or don't expand the entity
	feeds, err := service.ParseOPML(ctx, externalEntityPayload)

	if err == nil {
		// If no error, verify no SSRF occurred
		for _, feed := range feeds {
			// Should not contain AWS metadata
			assert.NotContains(t, feed.Title, "ami-id")
			assert.NotContains(t, feed.Title, "instance-id")
		}
	}
}

func TestImportService_ParseOPML_MaliciousURLInFeed(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// OPML with malicious URLs (javascript:, file:, etc.)
	// Note: XML special characters in attributes must be escaped
	maliciousOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
<head>
<title>Malicious URLs</title>
</head>
<body>
<outline type="rss" text="JavaScript" xmlUrl="javascript:alert(1)"/>
<outline type="rss" text="File" xmlUrl="file:///etc/passwd"/>
<outline type="rss" text="Data" xmlUrl="data:text/plain,malicious"/>
</body>
</opml>`

	feeds, err := service.ParseOPML(ctx, maliciousOPML)

	// The parser parses whatever is in the OPML
	// URL validation happens at the FeedService.Subscribe level
	require.NoError(t, err)
	assert.Len(t, feeds, 3)
	// These URLs should be present but will be validated later
	assert.Equal(t, "javascript:alert(1)", feeds[0].FeedURL)
	assert.Equal(t, "file:///etc/passwd", feeds[1].FeedURL)
}

func TestImportService_ParseOPML_LargeInput(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// Create a large OPML with many feeds (but not so large it times out)
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Large OPML</title>
  </head>
  <body>
`)
	// Add 100 feeds
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&body, `    <outline type="rss" text="Feed %d" xmlUrl="https://example%d.com/feed.xml"/>
`, i, i)
	}
	body.WriteString(`  </body>
</opml>`)

	feeds, err := service.ParseOPML(ctx, body.String())

	require.NoError(t, err)
	assert.Len(t, feeds, 100)
}

func TestImportService_ParseOPML_UnicodeHandling(t *testing.T) {
	ctx := context.Background()
	service := NewImportService(nil, nil, nil, nil)

	// OPML with various unicode characters
	unicodeOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Unicode 测试 Тест テスト</title>
  </head>
  <body>
    <outline type="rss" text="中文标题" xmlUrl="https://example.com/中文.xml"/>
    <outline type="rss" text="العربية" xmlUrl="https://example.com/arabic.xml"/>
    <outline type="rss" text="עברית" xmlUrl="https://example.com/hebrew.xml"/>
    <outline type="rss" text="日本語" xmlUrl="https://example.com/japanese.xml"/>
  </body>
</opml>`

	feeds, err := service.ParseOPML(ctx, unicodeOPML)

	require.NoError(t, err)
	assert.Len(t, feeds, 4)
	assert.Equal(t, "中文标题", feeds[0].Title)
	assert.Equal(t, "العربية", feeds[1].Title)
}

