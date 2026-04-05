package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// Mock feed service for testing
type mockFeedService struct {
	subscribeErr error
	getErr       error
	listErr      error
	deleteErr    error
	refreshErr   error
}

func (m *mockFeedService) Subscribe(ctx context.Context, userID, feedURL string) (*service.SubscribeResult, error) {
	if m.subscribeErr != nil {
		return nil, m.subscribeErr
	}
	feed := &model.Feed{
		Title:   "Test Feed",
		FeedURL: feedURL,
	}
	feed.GenerateID()
	return &service.SubscribeResult{
		Feed:         feed,
		NewItemCount: 5,
	}, nil
}

func (m *mockFeedService) GetUserFeeds(ctx context.Context, userID string, opts service.ListOptions) ([]*service.FeedWithItemCount, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	feed := &model.Feed{
		Title:    "Test Feed",
		FeedURL:  "https://example.com/feed.xml",
		ImageURL: "https://example.com/icon.png",
	}
	feed.GenerateID()
	return []*service.FeedWithItemCount{
		{
			Feed:       feed,
			ItemCount:  10,
			CategoryID: nil,
		},
	}, 1, nil
}

func (m *mockFeedService) GetFeed(ctx context.Context, userID, feedID string) (*model.Feed, int, error) {
	if m.getErr != nil {
		return nil, 0, m.getErr
	}
	feed := &model.Feed{
		Title:    "Test Feed",
		FeedURL:  "https://example.com/feed.xml",
		ImageURL: "https://example.com/icon.png",
	}
	feed.GenerateID()
	return feed, 10, nil
}

func (m *mockFeedService) DeleteFeed(ctx context.Context, userID, feedID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *mockFeedService) RefreshFeed(ctx context.Context, userID, feedID string) (*service.RefreshResult, error) {
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	return &service.RefreshResult{
		NewItemCount: 3,
	}, nil
}

func TestFeedHandler_CreateFeed_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.POST("/feeds", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.CreateFeed)

	body := map[string]string{
		"feed_url": "https://example.com/feed.xml",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/feeds", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusCreated)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	feed := response["feed"].(map[string]interface{})
	if feed["title"] != "Test Feed" {
		t.Errorf("Feed title = %v, want 'Test Feed'", feed["title"])
	}
}

func TestFeedHandler_CreateFeed_InvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{
		subscribeErr: service.ErrInvalidFeedURL,
	}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.POST("/feeds", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.CreateFeed)

	body := map[string]string{
		"feed_url": "not-a-url",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/feeds", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFeedHandler_ListFeeds_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.GET("/feeds", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ListFeeds)

	req := httptest.NewRequest("GET", "/feeds", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	feeds := response["feeds"].([]interface{})
	if len(feeds) != 1 {
		t.Errorf("Feeds count = %d, want 1", len(feeds))
	}

	total := int(response["total"].(float64))
	if total != 1 {
		t.Errorf("Total = %d, want 1", total)
	}
}

func TestFeedHandler_GetFeed_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.GET("/feeds/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetFeed)

	req := httptest.NewRequest("GET", "/feeds/feed-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	feed := response["feed"].(map[string]interface{})
	if feed["title"] != "Test Feed" {
		t.Errorf("Feed title = %v, want 'Test Feed'", feed["title"])
	}

	itemCount := int(response["item_count"].(float64))
	if itemCount != 10 {
		t.Errorf("ItemCount = %d, want 10", itemCount)
	}
}

func TestFeedHandler_GetFeed_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{
		getErr: service.ErrFeedNotFound,
	}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.GET("/feeds/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetFeed)

	req := httptest.NewRequest("GET", "/feeds/non-existent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestFeedHandler_DeleteFeed_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.DELETE("/feeds/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.DeleteFeed)

	req := httptest.NewRequest("DELETE", "/feeds/feed-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestFeedHandler_RefreshFeed_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockFeedService{}
	handler := NewFeedHandler(mockService)

	router := gin.New()
	router.POST("/feeds/:id/refresh", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.RefreshFeed)

	req := httptest.NewRequest("POST", "/feeds/feed-1/refresh", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	newItemCount := int(response["new_item_count"].(float64))
	if newItemCount != 3 {
		t.Errorf("NewItemCount = %d, want 3", newItemCount)
	}
}
