package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"oreader/internal/model"
	"oreader/internal/service"
)

// mockPaperService for handler testing
type mockPaperService struct {
	mock.Mock
}

func (m *mockPaperService) UploadPaper(ctx context.Context, userID string, filename string, pdfContent []byte) (*model.Paper, error) {
	args := m.Called(ctx, userID, filename, pdfContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}

func (m *mockPaperService) GetPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}

func (m *mockPaperService) ListPapers(ctx context.Context, userID string, opts service.PaperListOptions) ([]*model.Paper, int64, error) {
	args := m.Called(ctx, userID, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.Paper), args.Get(1).(int64), args.Error(2)
}

func (m *mockPaperService) UpdatePaper(ctx context.Context, userID, paperID string, updates map[string]interface{}) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}

func (m *mockPaperService) DeletePaper(ctx context.Context, userID, paperID string) error {
	args := m.Called(ctx, userID, paperID)
	return args.Error(0)
}

func (m *mockPaperService) GetPaperStatus(ctx context.Context, userID, paperID string) (*service.PaperStatusResponse, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PaperStatusResponse), args.Error(1)
}

func (m *mockPaperService) RetryPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	args := m.Called(ctx, userID, paperID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Paper), args.Error(1)
}

func (m *mockPaperService) ListTags(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockPaperService) UpdateTags(ctx context.Context, userID, paperID string, tags []string) error {
	args := m.Called(ctx, userID, paperID, tags)
	return args.Error(0)
}

func (m *mockPaperService) DownloadPaper(ctx context.Context, userID, paperID string) (string, string, error) {
	args := m.Called(ctx, userID, paperID)
	return args.String(0), args.String(1), args.Error(2)
}

func setupPaperRouter(svc *mockPaperService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	})

	h := NewPaperHandler(svc, 52428800) // 50MB max upload size
	papers := router.Group("/api/v1/papers")
	{
		papers.POST("/upload", h.UploadPaper)
		papers.GET("", h.ListPapers)
		papers.GET("/tags", h.ListTags)
		papers.GET("/:id", h.GetPaper)
		papers.GET("/:id/status", h.GetPaperStatus)
		papers.PUT("/:id", h.UpdatePaper)
		papers.PUT("/:id/tags", h.UpdateTags)
		papers.POST("/:id/retry", h.RetryPaper)
		papers.DELETE("/:id", h.DeletePaper)
		papers.GET("/:id/download", h.DownloadPaper)
	}

	return router
}

func TestPaperHandler_UploadPaper(t *testing.T) {
	svc := new(mockPaperService)
	paper := &model.Paper{Base: model.Base{ID: "paper-1"}, Status: model.PaperStatusPending, OriginalFilename: "test.pdf"}
	svc.On("UploadPaper", mock.Anything, "test-user-id", "test.pdf", mock.Anything).Return(paper, nil)

	router := setupPaperRouter(svc)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("%PDF-1.4 fake-pdf-content"))
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/papers/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "paper-1", resp["id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestPaperHandler_UploadPaper_NoFile(t *testing.T) {
	svc := new(mockPaperService)
	router := setupPaperRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/papers/upload", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPaperHandler_ListPapers(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("ListPapers", mock.Anything, "test-user-id", mock.Anything).
		Return([]*model.Paper{{Base: model.Base{ID: "p1"}, Title: "Paper 1"}}, int64(1), nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(1), resp["total"])
}

func TestPaperHandler_GetPaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("GetPaper", mock.Anything, "test-user-id", "paper-1").
		Return(&model.Paper{Base: model.Base{ID: "paper-1"}, Title: "My Paper"}, nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/paper-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaperHandler_DeletePaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("DeletePaper", mock.Anything, "test-user-id", "paper-1").Return(nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/v1/papers/paper-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestPaperHandler_DownloadPaper(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("fake-pdf"), 0644))

	svc := new(mockPaperService)
	svc.On("DownloadPaper", mock.Anything, "test-user-id", "paper-1").
		Return(pdfPath, "test.pdf", nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/paper-1/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}

func TestPaperHandler_GetPaperStatus(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("GetPaperStatus", mock.Anything, "test-user-id", "paper-1").
		Return(&service.PaperStatusResponse{
			ID:       "paper-1",
			Status:   model.PaperStatusCompleted,
			Progress: 100,
		}, nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/paper-1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "paper-1", resp["id"])
	assert.Equal(t, float64(100), resp["progress"])
}

func TestPaperHandler_RetryPaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("RetryPaper", mock.Anything, "test-user-id", "paper-1").
		Return(&model.Paper{Base: model.Base{ID: "paper-1"}, Status: model.PaperStatusPending}, nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("POST", "/api/v1/papers/paper-1/retry", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaperHandler_UpdatePaper(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("UpdatePaper", mock.Anything, "test-user-id", "paper-1", mock.Anything).
		Return(&model.Paper{Base: model.Base{ID: "paper-1"}, Title: "Updated Title"}, nil)

	router := setupPaperRouter(svc)

	body := `{"title": "Updated Title"}`
	req := httptest.NewRequest("PUT", "/api/v1/papers/paper-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaperHandler_UpdateTags(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("UpdateTags", mock.Anything, "test-user-id", "paper-1", mock.Anything).Return(nil)

	router := setupPaperRouter(svc)

	body := `{"tags": ["machine-learning", "nlp"]}`
	req := httptest.NewRequest("PUT", "/api/v1/papers/paper-1/tags", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPaperHandler_ListTags(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("ListTags", mock.Anything, "test-user-id").
		Return([]string{"machine-learning", "nlp"}, nil)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/tags", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	tags := resp["tags"].([]interface{})
	assert.Len(t, tags, 2)
}

func TestPaperHandler_GetPaper_NotFound(t *testing.T) {
	svc := new(mockPaperService)
	svc.On("GetPaper", mock.Anything, "test-user-id", "nonexistent").
		Return(nil, service.ErrPaperNotFound)

	router := setupPaperRouter(svc)

	req := httptest.NewRequest("GET", "/api/v1/papers/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
