package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"oreader/internal/model"
	"oreader/internal/service"
)

// mockImportService implements service.ImportService for testing
type mockImportService struct {
	parseErr    error
	startErr    error
	getJobErr   error
	job         *model.ImportJob
	parseResult []*service.FeedInfo
}

func (m *mockImportService) ParseOPML(ctx context.Context, content string) ([]*service.FeedInfo, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.parseResult, nil
}

func (m *mockImportService) StartImport(ctx context.Context, userID string, feeds []*service.FeedInfo) (*model.ImportJob, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	job := &model.ImportJob{
		Status:     model.ImportJobStatusPending,
		TotalFeeds: len(feeds),
	}
	job.GenerateID()
	return job, nil
}

func (m *mockImportService) GetJobStatus(ctx context.Context, jobID string) (*model.ImportJob, error) {
	if m.getJobErr != nil {
		return nil, m.getJobErr
	}
	return m.job, nil
}

func (m *mockImportService) ProcessImport(ctx context.Context, jobID string) error {
	return nil
}

// TestImportFeeds_NoFile tests import without a file
func TestImportFeeds_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	mockService := &mockImportService{}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ImportFeeds)

	// Create request without file
	req := httptest.NewRequest("POST", "/import", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestImportFeeds_FileTooLarge tests import with file exceeding size limit
func TestImportFeeds_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	mockService := &mockImportService{}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ImportFeeds)

	// Create a request with large file body
	body := []byte("x")
	for len(body) < 2*1024*1024 { // 2MB
		body = append(body, body...)
	}

	// Create multipart form
	var buf bytes.Buffer
	buf.WriteString("--boundary\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.opml\"\r\n")
	buf.WriteString("Content-Type: application/xml\r\n\r\n")
	buf.Write(body)
	buf.WriteString("\r\n--boundary--\r\n")

	req := httptest.NewRequest("POST", "/import", &buf)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 413 (Request Entity Too Large) or 400 (Bad Request)
	// depending on how the multipart parser handles the large file
	if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d or %d", w.Code, http.StatusRequestEntityTooLarge, http.StatusBadRequest)
	}
}

// TestImportFeeds_InvalidOPML tests import with invalid OPML content
func TestImportFeeds_InvalidOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	mockService := &mockImportService{
		parseErr: context.DeadlineExceeded, // Simulate parse error
	}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ImportFeeds)

	// Create multipart form with invalid OPML
	var buf bytes.Buffer
	buf.WriteString("--boundary\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.opml\"\r\n")
	buf.WriteString("Content-Type: application/xml\r\n\r\n")
	buf.WriteString("not valid opml content")
	buf.WriteString("\r\n--boundary--\r\n")

	req := httptest.NewRequest("POST", "/import", &buf)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestImportFeeds_EmptyOPML tests import with OPML containing no feeds
func TestImportFeeds_EmptyOPML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	mockService := &mockImportService{
		parseResult: []*service.FeedInfo{}, // Empty result
	}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ImportFeeds)

	// Create multipart form with empty OPML
	var buf bytes.Buffer
	buf.WriteString("--boundary\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.opml\"\r\n")
	buf.WriteString("Content-Type: application/xml\r\n\r\n")
	buf.WriteString("<?xml version=\"1.0\"?><opml></opml>")
	buf.WriteString("\r\n--boundary--\r\n")

	req := httptest.NewRequest("POST", "/import", &buf)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 200 with message about no feeds
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestGetImportStatus_UnauthorizedAccess tests accessing another user's job
func TestGetImportStatus_UnauthorizedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	otherUserID := "other-user-id"

	job := &model.ImportJob{
		Status:     model.ImportJobStatusCompleted,
		TotalFeeds: 5,
		Processed:  5,
		UserID:     otherUserID, // Belongs to different user
	}
	job.GenerateID()

	mockService := &mockImportService{
		job: job,
	}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.GET("/import/:job_id/status", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetImportStatus)

	req := httptest.NewRequest("GET", "/import/"+job.ID+"/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// TestGetImportStatus_NotFound tests accessing non-existent job
func TestGetImportStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"

	mockService := &mockImportService{
		job: nil, // Job not found
	}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.GET("/import/:job_id/status", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetImportStatus)

	req := httptest.NewRequest("GET", "/import/non-existent-job/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestExportFeeds_Success tests successful feed export
func TestExportFeeds_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	mockService := &mockImportService{}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.GET("/export", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ExportFeeds)

	req := httptest.NewRequest("GET", "/export", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Content-Type = %s, want application/xml", contentType)
	}

	// Verify disposition header
	disposition := w.Header().Get("Content-Disposition")
	if disposition == "" {
		t.Error("Content-Disposition header should be set")
	}
}

// TestExportFeeds_Unauthorized tests export without authentication
func TestExportFeeds_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockImportService{}
	mockFeedService := &mockFeedService{}
	handler := NewImportHandler(mockFeedService, mockService, nil)

	router := gin.New()
	router.GET("/export", handler.ExportFeeds) // No user_id middleware

	req := httptest.NewRequest("GET", "/export", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
