package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khalily/oreader/internal/handler"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// ────────────────────────────────────────────────
// Spec loading (singleton)
// ────────────────────────────────────────────────

var (
	specDoc  *openapi3.T
	specOnce sync.Once
	specErr  error
)

// loadSpec loads and validates the OpenAPI spec once.
func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	specOnce.Do(func() {
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true
		specDoc, specErr = loader.LoadFromFile("../../docs/openapi.yaml")
	})
	require.NoError(t, specErr, "Failed to load OpenAPI spec")
	require.NotNil(t, specDoc, "OpenAPI spec is nil")
	return specDoc
}

// ────────────────────────────────────────────────
// Schema validation
// ────────────────────────────────────────────────

// validateResponseSchema validates a JSON response body against the OpenAPI
// schema for the given method, spec path, and status code.
//
// specPath uses OpenAPI parameter syntax, e.g. "/api/v1/feeds/{feedId}".
func validateResponseSchema(t *testing.T, doc *openapi3.T, method, specPath string, statusCode int, body []byte) {
	t.Helper()

	pathItem := doc.Paths.Value(specPath)
	require.NotNil(t, pathItem, "Path %s not found in OpenAPI spec", specPath)

	var op *openapi3.Operation
	switch method {
	case http.MethodGet:
		op = pathItem.Get
	case http.MethodPost:
		op = pathItem.Post
	case http.MethodPut:
		op = pathItem.Put
	case http.MethodDelete:
		op = pathItem.Delete
	case http.MethodPatch:
		op = pathItem.Patch
	}
	require.NotNil(t, op, "%s %s not defined in OpenAPI spec", method, specPath)

	respRef := op.Responses.Status(statusCode)
	require.NotNil(t, respRef, "Status %d not defined for %s %s in spec", statusCode, method, specPath)
	require.NotNil(t, respRef.Value, "Response reference has no value for %s %s status %d", method, specPath, statusCode)

	if respRef.Value.Content == nil {
		return // No content to validate (e.g. 204)
	}

	jsonContent := respRef.Value.Content.Get("application/json")
	if jsonContent == nil || jsonContent.Schema == nil {
		return
	}

	schema := jsonContent.Schema.Value
	require.NotNil(t, schema, "Schema value is nil for %s %s status %d", method, specPath, statusCode)

	var data interface{}
	require.NoError(t, json.Unmarshal(body, &data), "Failed to parse response JSON for %s %s", method, specPath)

	err := schema.VisitJSON(data)
	assert.NoError(t, err, "Schema validation failed for %s %s (status %d)", method, specPath, statusCode)
}

// ────────────────────────────────────────────────
// Router helpers
// ────────────────────────────────────────────────

// authMiddleware is a stub that injects user_id into gin context,
// bypassing real JWT/CSRF middleware for contract tests.
func authMiddleware(c *gin.Context) {
	c.Set("user_id", "test-user-001")
	c.Next()
}

// newTestEngine creates a gin.Engine in test mode with the health endpoint.
func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// setupProtected creates a gin.Engine with authMiddleware applied to the
// given method+path, forwarding to handlerFn.
func setupProtected(method, path string, handlerFn gin.HandlerFunc) *gin.Engine {
	r := newTestEngine()
	r.Handle(method, path, authMiddleware, handlerFn)
	return r
}

// doRequest is a test helper that executes an HTTP request and returns the
// response status code and body.
func doRequest(r *gin.Engine, method, path string, body interface{}) (int, []byte) {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

// ────────────────────────────────────────────────
// Mock: FeedService
// ────────────────────────────────────────────────

type contractFeedService struct {
	feed    *model.Feed
	feeds   []*service.FeedWithItemCount
	total   int64
	itemCnt int
}

func newContractFeedService() *contractFeedService {
	now := time.Now()
	return &contractFeedService{
		feed: &model.Feed{
			Base:         model.Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			FeedURL:      "https://example.com/feed.xml",
			Title:        "Test Feed",
			Description:  "A test feed for contract validation",
			LastFetchStatus: "ok",
		},
		total:   1,
		itemCnt: 5,
	}
}

func (s *contractFeedService) initFeeds() {
	s.feeds = []*service.FeedWithItemCount{
		{Feed: s.feed, ItemCount: s.itemCnt},
	}
}

func (s *contractFeedService) Subscribe(_ context.Context, _, _ string) (*service.SubscribeResult, error) {
	return &service.SubscribeResult{Feed: s.feed, NewItemCount: 3}, nil
}

func (s *contractFeedService) GetUserFeeds(_ context.Context, _ string, _ service.ListOptions) ([]*service.FeedWithItemCount, int64, error) {
	s.initFeeds()
	return s.feeds, s.total, nil
}

func (s *contractFeedService) GetFeed(_ context.Context, _, _ string) (*model.Feed, int, error) {
	return s.feed, s.itemCnt, nil
}

func (s *contractFeedService) DeleteFeed(_ context.Context, _, _ string) error {
	return nil
}

func (s *contractFeedService) RefreshFeed(_ context.Context, _, _ string) (*service.RefreshResult, error) {
	return &service.RefreshResult{NewItemCount: 2}, nil
}

// ────────────────────────────────────────────────
// Mock: ItemService
// ────────────────────────────────────────────────

type contractItemService struct {
	item  *service.ItemWithState
	items []*service.ItemWithState
	total int64
}

func newContractItemService() *contractItemService {
	now := time.Now()
	pubDate := now.Add(-2 * time.Hour)
	readAt := now.Add(-1 * time.Hour).Format(time.RFC3339)
	feedID := uuid.New().String()
	itemID := uuid.New().String()

	return &contractItemService{
		item: &service.ItemWithState{
			Item: &model.Item{
				Base:        model.Base{ID: itemID, CreatedAt: now, UpdatedAt: now},
				FeedID:      feedID,
				GUID:        "https://example.com/article-1",
				Title:       "Test Article",
				Link:        "https://example.com/article-1",
				Description: "Article description",
				Content:     "<p>Full article content</p>",
				PubDate:     &pubDate,
				Creator:     "Author Name",
			},
			Feed: &service.FeedResponse{
				ID:          feedID,
				Title:       "Test Feed",
				FeedURL:     "https://example.com/feed.xml",
				Description: "A test feed",
			},
			UserState: &service.UserItemStateResponse{
				ItemID:    itemID,
				IsStarred: false,
				IsRead:    false,
				ReadAt:    &readAt,
			},
		},
		total: 1,
	}
}

func (s *contractItemService) initItems() {
	s.items = []*service.ItemWithState{s.item}
}

func (s *contractItemService) ListItems(_ context.Context, _ string, _ service.ListItemOptions) (*service.ItemListResult, error) {
	s.initItems()
	return &service.ItemListResult{
		Items:      s.items,
		Total:      s.total,
		HasMore:    false,
		NextCursor: "",
	}, nil
}

func (s *contractItemService) GetItem(_ context.Context, _, _ string) (*service.ItemWithState, error) {
	return s.item, nil
}

func (s *contractItemService) ToggleStar(_ context.Context, _, _ string) (*service.ItemWithState, error) {
	return s.item, nil
}

func (s *contractItemService) ToggleRead(_ context.Context, _, _ string) (*service.ItemWithState, error) {
	return s.item, nil
}

func (s *contractItemService) SetStar(_ context.Context, _, _ string, starred bool) (*service.ItemWithState, error) {
	s.item.UserState.IsStarred = starred
	return s.item, nil
}

func (s *contractItemService) SetRead(_ context.Context, _, _ string, read bool) (*service.ItemWithState, error) {
	s.item.UserState.IsRead = read
	return s.item, nil
}

func (s *contractItemService) MarkAllRead(_ context.Context, _, _ string) (int, error) {
	return 7, nil
}

// ────────────────────────────────────────────────
// Mock: StatsService
// ────────────────────────────────────────────────

type contractStatsService struct{}

func (s *contractStatsService) GetUserStats(_ context.Context, _ string) (*service.UserStats, error) {
	return &service.UserStats{Total: 100, Unread: 25, Starred: 10, Today: 3}, nil
}

// ────────────────────────────────────────────────
// Mock: PaperService
// ────────────────────────────────────────────────

type contractPaperService struct {
	paper *model.Paper
}

func newContractPaperService() *contractPaperService {
	now := time.Now()
	return &contractPaperService{
		paper: &model.Paper{
			Base:              model.Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			UserID:            "test-user-001",
			Title:             "Attention Is All You Need",
			Authors:           `["Ashish Vaswani","Noam Shazeer"]`,
			Abstract:          "The dominant sequence transduction models are based on complex recurrent or convolutional neural networks.",
			Keywords:          `["transformer","attention","neural network"]`,
			PublishedYear:     "2017",
			DOI:               "10.48550/arXiv.1706.03762",
			PDFSize:           1048576,
			MarkdownContent:   "# Attention Is All You Need\n\nAbstract text...",
			CoverImage:        "uploads/covers/paper-cover.png",
			OriginalFilename:  "attention-is-all-you-need.pdf",
			Status:            model.PaperStatusCompleted,
		},
	}
}

func (s *contractPaperService) UploadPaper(_ context.Context, _ string, _ string, _ []byte) (*model.Paper, error) {
	p := *s.paper
	p.Status = model.PaperStatusPending
	p.MarkdownContent = ""
	return &p, nil
}

func (s *contractPaperService) GetPaper(_ context.Context, _, _ string) (*model.Paper, error) {
	return s.paper, nil
}

func (s *contractPaperService) ListPapers(_ context.Context, _ string, _ service.PaperListOptions) ([]*model.Paper, int64, error) {
	return []*model.Paper{s.paper}, 1, nil
}

func (s *contractPaperService) UpdatePaper(_ context.Context, _, _ string, updates map[string]interface{}) (*model.Paper, error) {
	p := *s.paper
	if v, ok := updates["title"]; ok {
		p.Title = v.(string)
	}
	return &p, nil
}

func (s *contractPaperService) DeletePaper(_ context.Context, _, _ string) error {
	return nil
}

func (s *contractPaperService) GetPaperStatus(_ context.Context, _, _ string) (*service.PaperStatusResponse, error) {
	return &service.PaperStatusResponse{
		ID:        s.paper.ID,
		Status:    s.paper.Status,
		Progress:  100,
		CreatedAt: s.paper.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.paper.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *contractPaperService) RetryPaper(_ context.Context, _, _ string) (*model.Paper, error) {
	p := *s.paper
	p.Status = model.PaperStatusPending
	return &p, nil
}

func (s *contractPaperService) ListTags(_ context.Context, _ string) ([]string, error) {
	return []string{"ai", "ml", "transformer"}, nil
}

func (s *contractPaperService) UpdateTags(_ context.Context, _, _ string, tags []string) error {
	return nil
}

func (s *contractPaperService) DownloadPaper(_ context.Context, _, _ string) (string, string, error) {
	return "/tmp/test-paper.pdf", "test-paper.pdf", nil
}

// ────────────────────────────────────────────────
// Contract tests: Health
// ────────────────────────────────────────────────

func TestContract_HealthEndpoint(t *testing.T) {
	doc := loadSpec(t)

	r := newTestEngine()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "2.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	validateResponseSchema(t, doc, http.MethodGet, "/health", 200, w.Body.Bytes())
}

// ────────────────────────────────────────────────
// Contract tests: Feed endpoints
// ────────────────────────────────────────────────

func TestContract_FeedEndpoints(t *testing.T) {
	doc := loadSpec(t)
	mockFeed := newContractFeedService()
	h := handler.NewFeedHandler(mockFeed)
	testID := mockFeed.feed.ID

	t.Run("GET /api/v1/feeds -> 200 FeedListResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/feeds", h.ListFeeds)
		code, body := doRequest(r, http.MethodGet, "/api/v1/feeds", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/feeds", 200, body)
	})

	t.Run("POST /api/v1/feeds -> 201 CreateFeedResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPost, "/api/v1/feeds", h.CreateFeed)
		code, body := doRequest(r, http.MethodPost, "/api/v1/feeds", map[string]string{
			"feed_url": "https://example.com/feed.xml",
		})
		assert.Equal(t, http.StatusCreated, code)
		validateResponseSchema(t, doc, http.MethodPost, "/api/v1/feeds", 201, body)
	})

	t.Run("GET /api/v1/feeds/:id -> 200 GetFeedResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/feeds/:id", h.GetFeed)
		code, body := doRequest(r, http.MethodGet, "/api/v1/feeds/"+testID, nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/feeds/{feedId}", 200, body)
	})

	t.Run("DELETE /api/v1/feeds/:id -> 204", func(t *testing.T) {
		r := setupProtected(http.MethodDelete, "/api/v1/feeds/:id", h.DeleteFeed)
		code, _ := doRequest(r, http.MethodDelete, "/api/v1/feeds/"+testID, nil)
		assert.Equal(t, http.StatusNoContent, code)
		// 204 has no body; nothing to validate against schema
	})

	t.Run("POST /api/v1/feeds/:id/refresh -> 200 RefreshFeedResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPost, "/api/v1/feeds/:id/refresh", h.RefreshFeed)
		code, body := doRequest(r, http.MethodPost, "/api/v1/feeds/"+testID+"/refresh", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodPost, "/api/v1/feeds/{feedId}/refresh", 200, body)
	})
}

// ────────────────────────────────────────────────
// Contract tests: MarkAllRead (on feed path)
// ────────────────────────────────────────────────

func TestContract_MarkAllReadEndpoint(t *testing.T) {
	doc := loadSpec(t)
	mockItem := newContractItemService()
	h := handler.NewItemHandler(mockItem)
	testFeedID := mockItem.item.FeedID

	r := setupProtected(http.MethodPost, "/api/v1/feeds/:id/mark-all-read", h.MarkAllRead)
	code, body := doRequest(r, http.MethodPost, "/api/v1/feeds/"+testFeedID+"/mark-all-read", nil)
	assert.Equal(t, http.StatusOK, code)
	validateResponseSchema(t, doc, http.MethodPost, "/api/v1/feeds/{feedId}/mark-all-read", 200, body)
}

// ────────────────────────────────────────────────
// Contract tests: Item endpoints
// ────────────────────────────────────────────────

func TestContract_ItemEndpoints(t *testing.T) {
	doc := loadSpec(t)
	mockItem := newContractItemService()
	h := handler.NewItemHandler(mockItem)
	testItemID := mockItem.item.ID

	t.Run("GET /api/v1/items -> 200 ItemListResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/items", h.ListItems)
		code, body := doRequest(r, http.MethodGet, "/api/v1/items", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/items", 200, body)
	})

	t.Run("GET /api/v1/items/:id -> 200 ItemResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/items/:id", h.GetItem)
		code, body := doRequest(r, http.MethodGet, "/api/v1/items/"+testItemID, nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/items/{itemId}", 200, body)
	})

	t.Run("PUT /api/v1/items/:id/star -> 200 ItemResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPut, "/api/v1/items/:id/star", h.SetStar)
		code, body := doRequest(r, http.MethodPut, "/api/v1/items/"+testItemID+"/star", map[string]bool{
			"starred": true,
		})
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodPut, "/api/v1/items/{itemId}/star", 200, body)
	})

	t.Run("PUT /api/v1/items/:id/read -> 200 ItemResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPut, "/api/v1/items/:id/read", h.SetRead)
		code, body := doRequest(r, http.MethodPut, "/api/v1/items/"+testItemID+"/read", map[string]bool{
			"read": true,
		})
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodPut, "/api/v1/items/{itemId}/read", 200, body)
	})
}

// ────────────────────────────────────────────────
// Contract tests: Stats endpoint
// ────────────────────────────────────────────────

func TestContract_StatsEndpoint(t *testing.T) {
	doc := loadSpec(t)
	mockStats := &contractStatsService{}
	h := handler.NewStatsHandler(mockStats)

	r := setupProtected(http.MethodGet, "/api/v1/stats", h.GetStats)
	code, body := doRequest(r, http.MethodGet, "/api/v1/stats", nil)
	assert.Equal(t, http.StatusOK, code)
	validateResponseSchema(t, doc, http.MethodGet, "/api/v1/stats", 200, body)
}

// ────────────────────────────────────────────────
// Contract tests: Paper endpoints
// ────────────────────────────────────────────────

func TestContract_PaperEndpoints(t *testing.T) {
	doc := loadSpec(t)
	mockPaper := newContractPaperService()
	h := handler.NewPaperHandler(mockPaper, 50*1024*1024)
	testPaperID := mockPaper.paper.ID

	t.Run("GET /api/v1/papers -> 200 PaperListResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/papers", h.ListPapers)
		code, body := doRequest(r, http.MethodGet, "/api/v1/papers", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/papers", 200, body)
	})

	t.Run("GET /api/v1/papers/tags -> 200 tags", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/papers/tags", h.ListTags)
		code, body := doRequest(r, http.MethodGet, "/api/v1/papers/tags", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/papers/tags", 200, body)
	})

	t.Run("GET /api/v1/papers/:id -> 200 PaperResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/papers/:id", h.GetPaper)
		code, body := doRequest(r, http.MethodGet, "/api/v1/papers/"+testPaperID, nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/papers/{paperId}", 200, body)
	})

	t.Run("PUT /api/v1/papers/:id -> 200 PaperResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPut, "/api/v1/papers/:id", h.UpdatePaper)
		code, body := doRequest(r, http.MethodPut, "/api/v1/papers/"+testPaperID, map[string]string{
			"title": "Updated Paper Title",
		})
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodPut, "/api/v1/papers/{paperId}", 200, body)
	})

	t.Run("PUT /api/v1/papers/:id/tags -> 200 TagsResponse", func(t *testing.T) {
		r := setupProtected(http.MethodPut, "/api/v1/papers/:id/tags", h.UpdateTags)
		code, body := doRequest(r, http.MethodPut, "/api/v1/papers/"+testPaperID+"/tags", map[string]interface{}{
			"tags": []string{"tag1", "tag2"},
		})
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodPut, "/api/v1/papers/{paperId}/tags", 200, body)
	})

	t.Run("GET /api/v1/papers/:id/status -> 200 PaperStatusResponse", func(t *testing.T) {
		r := setupProtected(http.MethodGet, "/api/v1/papers/:id/status", h.GetPaperStatus)
		code, body := doRequest(r, http.MethodGet, "/api/v1/papers/"+testPaperID+"/status", nil)
		assert.Equal(t, http.StatusOK, code)
		validateResponseSchema(t, doc, http.MethodGet, "/api/v1/papers/{paperId}/status", 200, body)
	})

	t.Run("DELETE /api/v1/papers/:id -> 204", func(t *testing.T) {
		r := setupProtected(http.MethodDelete, "/api/v1/papers/:id", h.DeletePaper)
		code, _ := doRequest(r, http.MethodDelete, "/api/v1/papers/"+testPaperID, nil)
		assert.Equal(t, http.StatusNoContent, code)
		// 204 has no body; nothing to validate against schema
	})
}
