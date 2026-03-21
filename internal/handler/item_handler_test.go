package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"oreader/internal/model"
	"oreader/internal/service"
)

// Mock item service for testing
type mockItemService struct {
	listErr       error
	getErr        error
	toggleStarErr error
	toggleReadErr error
	setStarErr    error
	setReadErr    error
	markAllErr    error

	// Track SetStar/SetRead calls for spec compliance verification
	SetStarCalls []struct {
		UserID  string
		ItemID  string
		Starred bool
	}
	SetReadCalls []struct {
		UserID string
		ItemID string
		Read   bool
	}
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
			{Item: item1, UserState: &service.UserItemStateResponse{ItemID: "item-1", IsStarred: false, IsRead: false}},
			{Item: item2, UserState: &service.UserItemStateResponse{ItemID: "item-2", IsStarred: true, IsRead: true}},
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
		Feed: &model.Feed{
			Base:  model.Base{ID: "feed-1"},
			Title: "Test Feed",
		},
	}
	return &service.ItemWithState{
		Item:      item,
		Feed:      &service.FeedResponse{ID: "feed-1", Title: "Test Feed"},
		UserState: &service.UserItemStateResponse{ItemID: itemID, IsStarred: true, IsRead: false},
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
		Item:      item,
		UserState: &service.UserItemStateResponse{ItemID: itemID, IsStarred: true, IsRead: false},
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
		Item:      item,
		UserState: &service.UserItemStateResponse{ItemID: itemID, IsStarred: false, IsRead: true},
	}, nil
}

func (m *mockItemService) MarkAllRead(ctx context.Context, userID, feedID string) (int, error) {
	if m.markAllErr != nil {
		return 0, m.markAllErr
	}
	return 5, nil
}

// SetStar sets the star status (spec-compliant: sets value, doesn't toggle)
func (m *mockItemService) SetStar(ctx context.Context, userID, itemID string, starred bool) (*service.ItemWithState, error) {
	m.SetStarCalls = append(m.SetStarCalls, struct {
		UserID  string
		ItemID  string
		Starred bool
	}{UserID: userID, ItemID: itemID, Starred: starred})

	if m.setStarErr != nil {
		return nil, m.setStarErr
	}
	item := &model.Item{
		Base:   model.Base{ID: itemID},
		FeedID: "feed-1",
		Title:  "Test Item",
		Link:   "https://example.com/item",
	}
	return &service.ItemWithState{
		Item:      item,
		UserState: &service.UserItemStateResponse{ItemID: itemID, IsStarred: starred, IsRead: false},
	}, nil
}

// SetRead sets the read status (spec-compliant: sets value, doesn't toggle)
func (m *mockItemService) SetRead(ctx context.Context, userID, itemID string, read bool) (*service.ItemWithState, error) {
	m.SetReadCalls = append(m.SetReadCalls, struct {
		UserID string
		ItemID string
		Read   bool
	}{UserID: userID, ItemID: itemID, Read: read})

	if m.setReadErr != nil {
		return nil, m.setReadErr
	}
	item := &model.Item{
		Base:   model.Base{ID: itemID},
		FeedID: "feed-1",
		Title:  "Test Item",
		Link:   "https://example.com/item",
	}
	result := &service.ItemWithState{
		Item:      item,
		UserState: &service.UserItemStateResponse{ItemID: itemID, IsStarred: false, IsRead: read},
	}
	if read {
		now := time.Now().Format(time.RFC3339)
		result.UserState.ReadAt = &now
	}
	return result, nil
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

	// Check nested user_state structure
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("Expected 'user_state' field in item response")
	}
	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("user_state should be an object, got: %T", userState)
	}
	if userStateMap["is_starred"] != true {
		t.Error("Expected user_state.is_starred to be true")
	}
	if userStateMap["is_read"] != false {
		t.Error("Expected user_state.is_read to be false")
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

	// Check nested user_state structure
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("Expected 'user_state' field in item response")
	}
	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("user_state should be an object, got: %T", userState)
	}
	if userStateMap["is_starred"] != true {
		t.Error("Expected user_state.is_starred to be true")
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

	// Check nested user_state structure
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("Expected 'user_state' field in item response")
	}
	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("user_state should be an object, got: %T", userState)
	}
	if userStateMap["is_read"] != true {
		t.Error("Expected user_state.is_read to be true")
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

// =============================================================================
// SPEC COMPLIANCE TESTS: SetStar (not toggle)
// These tests verify that the handler passes the request body value to the service,
// not ignoring it or toggling.
// =============================================================================

// TestItemHandler_SetStar_SetsToTrue verifies the handler passes starred=true to service
func TestItemHandler_SetStar_SetsToTrue(t *testing.T) {
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

	// Send request with starred=true
	body := map[string]bool{"starred": true}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/"+itemID+"/star", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify the service was called with starred=true (not toggled)
	if len(mockService.SetStarCalls) == 0 {
		t.Fatal("SetStar was not called")
	}

	call := mockService.SetStarCalls[len(mockService.SetStarCalls)-1]
	if call.Starred != true {
		t.Errorf("Service.SetStar called with starred=%v, want true", call.Starred)
	}
	if call.ItemID != itemID {
		t.Errorf("Service.SetStar called with itemID=%v, want %v", call.ItemID, itemID)
	}

	// Verify response with nested user_state structure
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("Expected 'user_state' field in item response")
	}
	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("user_state should be an object, got: %T", userState)
	}
	if userStateMap["is_starred"] != true {
		t.Error("Response should have user_state.is_starred=true")
	}
}

// TestItemHandler_SetStar_SetsToFalse verifies the handler passes starred=false to service
func TestItemHandler_SetStar_SetsToFalse(t *testing.T) {
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

	// Send request with starred=false
	body := map[string]bool{"starred": false}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/"+itemID+"/star", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify the service was called with starred=false (not toggled)
	if len(mockService.SetStarCalls) == 0 {
		t.Fatal("SetStar was not called")
	}

	call := mockService.SetStarCalls[len(mockService.SetStarCalls)-1]
	if call.Starred != false {
		t.Errorf("Service.SetStar called with starred=%v, want false", call.Starred)
	}

	// Verify response with nested user_state structure
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("Expected 'user_state' field in item response")
	}
	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("user_state should be an object, got: %T", userState)
	}
	if userStateMap["is_starred"] != false {
		t.Error("Response should have user_state.is_starred=false")
	}
}

// =============================================================================
// TDD RED PHASE: Response Structure Tests
// These tests verify the EXPECTED nested response structure.
// They will FAIL with the current implementation.
// =============================================================================

// TestItemHandler_ListItems_NestedUserStateStructure verifies ListItems response
// has nested user_state object instead of flat is_starred/is_read fields
func TestItemHandler_ListItems_NestedUserStateStructure(t *testing.T) {
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
	if len(items) == 0 {
		t.Fatal("Expected at least one item")
	}

	item := items[0].(map[string]interface{})

	// TDD: These assertions will FAIL with current implementation
	// Current: is_starred at top level
	// Expected: user_state object containing is_starred

	// Check that is_starred is NOT at top level
	if _, exists := item["is_starred"]; exists {
		t.Error("FAIL: 'is_starred' should NOT be at item top level, should be in user_state")
	}

	// Check that user_state exists
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("FAIL: 'user_state' field must exist in item response")
	}

	// user_state should be an object or null
	if userState != nil {
		userStateMap, ok := userState.(map[string]interface{})
		if !ok {
			t.Errorf("'user_state' should be an object, got: %T", userState)
		} else {
			if _, exists := userStateMap["is_starred"]; !exists {
				t.Error("'user_state' should contain 'is_starred'")
			}
			if _, exists := userStateMap["is_read"]; !exists {
				t.Error("'user_state' should contain 'is_read'")
			}
		}
	}
}

// TestItemHandler_GetItem_NestedFeedObject verifies GetItem response
// has nested feed object instead of feed_title string
func TestItemHandler_GetItem_NestedFeedObject(t *testing.T) {
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

	// TDD: These assertions will FAIL with current implementation
	// Current: feed_title string at top level
	// Expected: feed object with id and title

	// Check that feed_title is NOT at top level
	if _, exists := item["feed_title"]; exists {
		t.Error("FAIL: 'feed_title' should NOT be at item top level, should use 'feed' object")
	}

	// Check that feed object exists
	feed, exists := item["feed"]
	if !exists {
		t.Fatal("FAIL: 'feed' field must exist in item response")
	}

	feedMap, ok := feed.(map[string]interface{})
	if !ok {
		t.Fatalf("'feed' should be an object, got: %T", feed)
	}

	if _, exists := feedMap["id"]; !exists {
		t.Error("'feed' should contain 'id'")
	}
	if _, exists := feedMap["title"]; !exists {
		t.Error("'feed' should contain 'title'")
	}
}

// TestItemHandler_SetStar_NestedUserStateResponse verifies SetStar response
// has nested user_state object with updated is_starred value
func TestItemHandler_SetStar_NestedUserStateResponse(t *testing.T) {
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

	body := map[string]bool{"starred": true}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/"+itemID+"/star", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})

	// TDD: Check for nested user_state structure
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("FAIL: 'user_state' field must exist in item response")
	}

	if userState == nil {
		t.Fatal("FAIL: 'user_state' should not be null after SetStar")
	}

	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("'user_state' should be an object, got: %T", userState)
	}

	// Verify is_starred is true inside user_state
	if userStateMap["is_starred"] != true {
		t.Errorf("user_state.is_starred should be true, got: %v", userStateMap["is_starred"])
	}
}

// TestItemHandler_SetRead_NestedUserStateResponse verifies SetRead response
// has nested user_state object with updated is_read value
func TestItemHandler_SetRead_NestedUserStateResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	itemID := "item-1"

	mockService := &mockItemService{}
	handler := NewItemHandler(mockService)

	router := gin.New()
	router.PUT("/items/:id/read", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.SetRead)

	body := map[string]bool{"read": true}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/items/"+itemID+"/read", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	item := response["item"].(map[string]interface{})

	// TDD: Check for nested user_state structure
	userState, exists := item["user_state"]
	if !exists {
		t.Fatal("FAIL: 'user_state' field must exist in item response")
	}

	if userState == nil {
		t.Fatal("FAIL: 'user_state' should not be null after SetRead")
	}

	userStateMap, ok := userState.(map[string]interface{})
	if !ok {
		t.Fatalf("'user_state' should be an object, got: %T", userState)
	}

	// Verify is_read is true inside user_state
	if userStateMap["is_read"] != true {
		t.Errorf("user_state.is_read should be true, got: %v", userStateMap["is_read"])
	}

	// Verify read_at is set
	if _, exists := userStateMap["read_at"]; !exists {
		t.Error("user_state should contain 'read_at' after marking as read")
	}
}

