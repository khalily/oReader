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

// --- mock category service ---

type mockCategoryService struct {
	createErr    error
	listErr      error
	renameErr    error
	deleteErr    error
	moveFeedErr  error
	movePaperErr error
	category     *model.Category
	categories   []*model.Category
}

func (m *mockCategoryService) CreateCategory(_ context.Context, _ string, name string, categoryType string) (*model.Category, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.category != nil {
		return m.category, nil
	}
	cat := &model.Category{
		UserID:   "test-user-id",
		Name:     name,
		Type:     categoryType,
		Position: 1,
	}
	cat.GenerateID()
	return cat, nil
}

func (m *mockCategoryService) ListCategories(_ context.Context, _ string, _ string) ([]*model.Category, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.categories != nil {
		return m.categories, nil
	}
	cat := &model.Category{
		UserID:   "test-user-id",
		Name:     "Tech",
		Type:     model.CategoryTypeFeed,
		Position: 1,
	}
	cat.GenerateID()
	return []*model.Category{cat}, nil
}

func (m *mockCategoryService) RenameCategory(_ context.Context, _ string, _ string, newName string) (*model.Category, error) {
	if m.renameErr != nil {
		return nil, m.renameErr
	}
	cat := &model.Category{
		UserID:   "test-user-id",
		Name:     newName,
		Type:     model.CategoryTypeFeed,
		Position: 1,
	}
	cat.GenerateID()
	return cat, nil
}

func (m *mockCategoryService) DeleteCategory(_ context.Context, _ string, _ string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *mockCategoryService) MoveFeedToCategory(_ context.Context, _ string, _ string, _ string) error {
	if m.moveFeedErr != nil {
		return m.moveFeedErr
	}
	return nil
}

func (m *mockCategoryService) MovePaperToCategory(_ context.Context, _ string, _ string, _ string) error {
	if m.movePaperErr != nil {
		return m.movePaperErr
	}
	return nil
}

// --- helper ---

func setupCategoryRouter(h *CategoryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	}

	categories := router.Group("/api/v1/categories")
	categories.Use(authMiddleware)
	{
		categories.GET("", h.ListCategories)
		categories.POST("", h.CreateCategory)
		categories.PUT("/:id/rename", h.RenameCategory)
		categories.DELETE("/:id", h.DeleteCategory)
		categories.PUT("/feeds/:feedId", h.MoveFeedToCategory)
		categories.PUT("/papers/:paperId", h.MovePaperToCategory)
	}

	return router
}

// --- tests ---

func TestCreateCategory(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"name": "Tech",
		"type": "feed",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateCategory: status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	cat, ok := response["category"].(map[string]interface{})
	if !ok {
		t.Fatalf("CreateCategory: response missing 'category' key, got %+v", response)
	}
	if cat["name"] != "Tech" {
		t.Errorf("CreateCategory: name = %v, want 'Tech'", cat["name"])
	}
}

func TestCreateCategory_Duplicate(t *testing.T) {
	mockSvc := &mockCategoryService{createErr: service.ErrCategoryDuplicate}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"name": "Tech",
		"type": "feed",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("CreateCategory duplicate: status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestCreateCategory_Validation(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	// missing name
	body := map[string]string{
		"type": "feed",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateCategory validation: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListCategories(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	req := httptest.NewRequest("GET", "/api/v1/categories?type=feed", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListCategories: status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	cats, ok := response["categories"].([]interface{})
	if !ok {
		t.Fatalf("ListCategories: response missing 'categories' key, got %+v", response)
	}
	if len(cats) != 1 {
		t.Errorf("ListCategories: count = %d, want 1", len(cats))
	}
}

func TestRenameCategory(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"name": "New Name",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/cat-1/rename", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RenameCategory: status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	cat, ok := response["category"].(map[string]interface{})
	if !ok {
		t.Fatalf("RenameCategory: response missing 'category' key, got %+v", response)
	}
	if cat["name"] != "New Name" {
		t.Errorf("RenameCategory: name = %v, want 'New Name'", cat["name"])
	}
}

func TestRenameCategory_NotFound(t *testing.T) {
	mockSvc := &mockCategoryService{renameErr: service.ErrCategoryNotFound}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"name": "New Name",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/nonexistent/rename", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("RenameCategory not found: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteCategory(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	req := httptest.NewRequest("DELETE", "/api/v1/categories/cat-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DeleteCategory: status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	mockSvc := &mockCategoryService{deleteErr: service.ErrCategoryNotFound}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	req := httptest.NewRequest("DELETE", "/api/v1/categories/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DeleteCategory not found: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMoveFeedToCategory(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"category_id": "cat-1",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/feeds/feed-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("MoveFeedToCategory: status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestMoveFeedToCategory_NotFound(t *testing.T) {
	mockSvc := &mockCategoryService{moveFeedErr: service.ErrCategoryNotFound}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"category_id": "nonexistent",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/feeds/feed-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("MoveFeedToCategory not found: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMovePaperToCategory(t *testing.T) {
	mockSvc := &mockCategoryService{}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"category_id": "cat-1",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/papers/paper-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("MovePaperToCategory: status = %d, want %d; body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestMovePaperToCategory_NotFound(t *testing.T) {
	mockSvc := &mockCategoryService{movePaperErr: service.ErrCategoryNotFound}
	h := NewCategoryHandler(mockSvc)
	router := setupCategoryRouter(h)

	body := map[string]string{
		"category_id": "nonexistent",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/v1/categories/papers/paper-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("MovePaperToCategory not found: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
