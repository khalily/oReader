package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"oreader/internal/model"
	"oreader/internal/service"
)

// Mock item service for testing
type mockItemService struct {
	listErr      error
	getErr       error
	toggleStarErr error
	toggleReadErr error
	markAllErr   error
}

func (m *mockItemService) ListItems(ctx context.Context, userID string, opts service.ListItemOptions) (*service.ItemListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	item1 := &model.Item{
		Base: model.Base{ID: "item-1"},
		FeedID: "feed-1",
		Title: "Test Item 1",
		Link: "https://example.com/1",
	}
	item2 := &model.Item{
		Base: model.Base{ID: "item-2"},
		FeedID: "feed-1",
		Title: "Test Item 2",
		Link: "https://example.com/2",
	}
	return &service.ItemListResult{
		Items: []*service.ItemWithState{
			{Item: item1, IsStarred: false, IsRead: false},
			{Item: item2, IsStarred: true, IsRead: true},
		},
		Total:   2,
		HasMore: false,
	}, nil
}

func (m *mockItemService) GetItem(ctx context.Context, userID, itemID string) (*service.ItemWithState, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	item := &model.Item{
		Base: model.Base{ID: itemID},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
		Description: "Test description",
		Content: "<p>Test content</p>",
	}
	return &service.ItemWithState{
		Item:     item,
		IsStarred: true,
		IsRead:   false,
	}, nil
}

func (m *mockItemService) ToggleStar(ctx context.Context, userID, itemID string) (*service.ItemWithState, error) {
	if m.toggleStarErr != nil {
		return nil, m.toggleStarErr
	}
	item := &model.Item{
		Base: model.Base{ID: itemID},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
	}
	return &service.ItemWithState{
		Item:     item,
		IsStarred: true,
		IsRead:   false,
	}, nil
}

func (m *mockItemService) ToggleRead(ctx context.Context, userID, itemID string) (*service.ItemWithState, error) {
	if m.toggleReadErr != nil {
		return nil, m.toggleReadErr
	}
	item := &model.Item{
		Base: model.Base{ID: itemID},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
	}
	return &service.ItemWithState{
		Item:     item,
		IsStarred: false,
		IsRead:   true,
	}, nil
}

func (m *mockItemService) MarkAllRead(ctx context.Context, userID, feedID string) (int, error) {
	if m.markAllErr != nil {
		return 0, m.markAllErr
	}
	return 5, nil
}

// Test ListItems
func TestItemHandler_ListItems_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.GET("/items", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ListItems)

	req := httptest.NewRequest("GET", "/items", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	items := response["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Items count = %d, want 2", len(items))
	}

	total := int(response["total"].(float64))
	if total != 2 {
		t.Errorf("Total = %d, want 2", total)
	}
}

func TestItemHandler_ListItems_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.GET("/items", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ListItems)

	// Test with starred filter
	req := httptest.NewRequest("GET", "/items?starred=true", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Test with read filter
	req = httptest.NewRequest("GET", "/items?read=false", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Test with feed_id filter
	req = httptest.NewRequest("GET", "/items?feed_id=feed-1", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestItemHandler_ListItems_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.GET("/items", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ListItems)

	req := httptest.NewRequest("GET", "/items?limit=10&cursor=abc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["has_more"] == nil {
		t.Error("Expected has_more field in response")
	}
}

// Test GetItem
func TestItemHandler_GetItem_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.GET("/items/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetItem)

	req := httptest.NewRequest("GET", "/items/item-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})
	if item["id"] != "item-1" {
		t.Errorf("Item id = %v, want 'item-1'", item["id"])
	}
	if item["title"] != "Test Item" {
		t.Errorf("Item title = %v, want 'Test Item'", item["title"])
	}
	if item["is_starred"] != true {
		t.Error("Expected item to be starred")
	}
	if item["is_read"] != false {
		t.Error("Expected item to be unread")
	}
}

func TestItemHandler_GetItem_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{
		getErr: service.ErrItemNotFound,
	}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.GET("/items/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetItem)

	req := httptest.NewRequest("GET", "/items/non-existent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test ToggleStar
func TestItemHandler_ToggleStar_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.PUT("/items/:id/star", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ToggleStar)

	body := map[string]bool{
		"starred": true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/item-1/star", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})
	if item["id"] != "item-1" {
		t.Errorf("Item id = %v, want 'item-1'", item["id"])
	}
	if item["is_starred"] != true {
		t.Error("Expected item to be starred")
	}
}

func TestItemHandler_ToggleStar_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.PUT("/items/:id/star", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ToggleStar)

	req := httptest.NewRequest("PUT", "/items/item-1/star", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// Test ToggleRead
func TestItemHandler_ToggleRead_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.PUT("/items/:id/read", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ToggleRead)

	body := map[string]bool{
		"read": true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/item-1/read", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})
	if item["id"] != "item-1" {
		t.Errorf("Item id = %v, want 'item-1'", item["id"])
	}
	if item["is_read"] != true {
		t.Error("Expected item to be read")
	}
}

func TestItemHandler_ToggleRead_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.PUT("/items/:id/read", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ToggleRead)

	req := httptest.NewRequest("PUT", "/items/item-1/read", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// Test MarkAllRead
func TestItemHandler_MarkAllRead_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.POST("/feeds/:id/read-all", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.MarkAllRead)

	req := httptest.NewRequest("POST", "/feeds/feed-1/read-all", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	count := int(response["count"].(float64))
	if count != 5 {
		t.Errorf("Count = %d, want 5", count)
	}
}

func TestItemHandler_MarkAllRead_FeedNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockItemService{
		markAllErr: service.ErrFeedNotFound,
	}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.POST("/feeds/:id/read-all", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.MarkAllRead)

	req := httptest.NewRequest("POST", "/feeds/non-existent/read-all", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
