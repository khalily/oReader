package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/infra/grpc"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
	"github.com/khalily/oreader/internal/testutil"
)

// ─── E2E test for Paper upload -> convert -> retrieve flow ──────────
//
// This test requires:
//   - MySQL running (testutil.SetupTestDB)
//   - Converter gRPC service running on PAPER_GRPC_ADDR (default localhost:50051)
//
// Run with: go test ./internal/handler/ -run TestPaperE2E -tags=e2e -timeout 180s
// Or with the test environment: make docker-test

// gormPaperRepo is a test-only repository implementation for E2E tests.
// This avoids import cycles with the repository package.
type e2ePaperRepo struct {
	db *gorm.DB
}

func (r *e2ePaperRepo) Create(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Create(paper).Error
}
func (r *e2ePaperRepo) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&paper).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &paper, nil
}
func (r *e2ePaperRepo) ListByUserID(ctx context.Context, userID string, opts service.PaperListOptions) ([]*model.Paper, int64, error) {
	var papers []*model.Paper
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Paper{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	if err := query.Order("created_at DESC").Find(&papers).Error; err != nil {
		return nil, 0, err
	}
	return papers, total, nil
}
func (r *e2ePaperRepo) Update(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Save(paper).Error
}
func (r *e2ePaperRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("paper_id = ?", id).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Paper{}, "id = ?", id).Error
	})
}
func (r *e2ePaperRepo) ListTags(ctx context.Context, userID string) ([]string, error) {
	var tags []string
	err := r.db.WithContext(ctx).Model(&model.PaperTag{}).
		Joins("JOIN papers ON papers.id = paper_tags.paper_id").
		Where("papers.user_id = ?", userID).
		Distinct("tag").
		Pluck("tag", &tags).Error
	return tags, err
}

type e2ePaperTagRepo struct {
	db *gorm.DB
}

func (r *e2ePaperTagRepo) Create(ctx context.Context, tag *model.PaperTag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}
func (r *e2ePaperTagRepo) DeleteByPaperID(ctx context.Context, paperID string) error {
	return r.db.WithContext(ctx).Where("paper_id = ?", paperID).Delete(&model.PaperTag{}).Error
}
func (r *e2ePaperTagRepo) GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error) {
	var tags []*model.PaperTag
	err := r.db.WithContext(ctx).Where("paper_id = ?", paperID).Find(&tags).Error
	return tags, err
}
func (r *e2ePaperTagRepo) SetTags(ctx context.Context, paperID string, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("paper_id = ?", paperID).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		for _, tag := range tags {
			paperTag := &model.PaperTag{PaperID: paperID, Tag: tag}
			if err := paperTag.GenerateID(); err != nil {
				return err
			}
			if err := tx.Create(paperTag).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// setupE2EEnv sets up the full E2E environment: DB + gRPC client + service + handler + router.
// Returns the router (with auth middleware set to a test user) and a cleanup function.
func setupE2EEnv(t *testing.T) (*gin.Engine, func()) {
	t.Helper()

	// 1. Setup test database
	db := testutil.SetupTestDB(t)

	// 1b. AutoMigrate to ensure all tables exist
	if err := db.AutoMigrate(
		&model.User{},
		&model.Paper{},
		&model.PaperTag{},
		&model.PaperCollection{},
		&model.PaperCollectionItem{},
	); err != nil {
		t.Fatalf("AutoMigrate failed: %v (ensure oreader_test database exists)", err)
	}

	// 2. Create test user
	user := &model.User{Email: "e2e-paper@example.com", PasswordHash: "test_hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, db.Create(user).Error)
	testUserID := user.ID

	// 3. Connect to converter gRPC service
	grpcAddr := os.Getenv("PAPER_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}
	grpcClient, err := grpc.NewPaperClient(grpcAddr, 15*time.Minute)
	if err != nil {
		t.Skipf("Converter gRPC service not available at %s: %v", grpcAddr, err)
	}

	// 4. Create temp upload directory
	uploadDir := t.TempDir()

	// 5. Create service and handler
	paperRepo := &e2ePaperRepo{db: db}
	paperTagRepo := &e2ePaperTagRepo{db: db}
	paperSvc := service.NewPaperService(paperRepo, paperTagRepo, grpcClient, uploadDir, 15*time.Minute)
	paperHandler := NewPaperHandler(paperSvc, 52428800)

	// 6. Setup Gin router with auth middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})

	papers := router.Group("/api/v1/papers")
	{
		papers.POST("/upload", paperHandler.UploadPaper)
		papers.GET("", paperHandler.ListPapers)
		papers.GET("/tags", paperHandler.ListTags)
		papers.GET("/:id", paperHandler.GetPaper)
		papers.GET("/:id/status", paperHandler.GetPaperStatus)
		papers.PUT("/:id", paperHandler.UpdatePaper)
		papers.PUT("/:id/tags", paperHandler.UpdateTags)
		papers.POST("/:id/retry", paperHandler.RetryPaper)
		papers.DELETE("/:id", paperHandler.DeletePaper)
		papers.GET("/:id/download", paperHandler.DownloadPaper)
	}

	cleanup := func() {
		grpcClient.Close()
	}

	return router, cleanup
}

// ─── E2E Test: Full Upload -> Convert -> Retrieve Pipeline ──────────

func TestPaperE2E_UploadConvertRetrieve(t *testing.T) {
	router, cleanup := setupE2EEnv(t)
	defer cleanup()

	// Locate the test PDF
	pdfPath := filepath.Join("..", "..", "..", "uploads", "papers",
		"019d2fae-69a3-73f4-90ea-ce15ee1db999", "2026-03-28", "rdma.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		// Try absolute path as fallback
		pdfPath = "/data00/home/wangyang.backend/work/oReader/uploads/papers/019d2fae-69a3-73f4-90ea-ce15ee1db999/2026-03-28/rdma.pdf"
		if _, err2 := os.Stat(pdfPath); os.IsNotExist(err2) {
			t.Skipf("Test PDF not found, skipping E2E test")
		}
	}

	pdfContent, err := os.ReadFile(pdfPath)
	require.NoError(t, err)
	require.True(t, len(pdfContent) > 100, "PDF content should be non-trivial")

	// ── Step 1: Upload the PDF ────────────────────────────────────
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "rdma.pdf")
	require.NoError(t, err)
	_, err = part.Write(pdfContent)
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/papers/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code, "Upload should return 202 Accepted")

	var uploadResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &uploadResp))

	paperID, ok := uploadResp["id"].(string)
	require.True(t, ok && paperID != "", "Response should contain a valid paper ID")
	assert.Equal(t, "pending", uploadResp["status"])
	assert.Equal(t, "rdma.pdf", uploadResp["original_filename"])

	t.Logf("Paper uploaded: id=%s status=%s", paperID, uploadResp["status"])

	// ── Step 2: Poll status until completed ────────────────────────
	var finalStatus string
	var finalProgress float64
	deadline := time.Now().Add(15 * time.Minute)

	for time.Now().Before(deadline) {
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/papers/%s/status", paperID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var statusResp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &statusResp))

		finalStatus = statusResp["status"].(string)
		finalProgress, _ = statusResp["progress"].(float64)

		t.Logf("Paper status: status=%s progress=%.0f%%", finalStatus, finalProgress)

		if finalStatus == "completed" {
			break
		}
		if finalStatus == "failed" {
			errMsg, _ := statusResp["error"].(string)
			t.Fatalf("Paper conversion failed: %s", errMsg)
		}

		time.Sleep(3 * time.Second)
	}

	require.Equal(t, "completed", finalStatus, "Paper conversion should complete within timeout")
	assert.Equal(t, float64(100), finalProgress, "Completed paper should have 100%% progress")

	// ── Step 3: Retrieve the paper and verify markdown content ─────
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/papers/%s", paperID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var getResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &getResp))

	paper, ok := getResp["paper"].(map[string]interface{})
	require.True(t, ok, "Response should contain paper object")
	assert.Equal(t, "completed", paper["status"])

	markdownContent, _ := paper["markdown_content"].(string)
	require.NotEmpty(t, markdownContent, "Completed paper should have markdown content")
	require.Greater(t, len(markdownContent), 500, "Markdown content should be substantial")

	t.Logf("Paper retrieved: title=%q markdown_length=%d", paper["title"], len(markdownContent))

	// ── Step 4: Verify markdown quality ────────────────────────────

	// 4a. LaTeX formulas
	hasInlineMath := bytes.ContainsAny([]byte(markdownContent), "$")
	hasDisplayMath := bytes.Contains([]byte(markdownContent), []byte("$$"))
	assert.True(t, hasInlineMath || hasDisplayMath,
		"Markdown should contain LaTeX formulas ($ or $$)")

	// 4b. Headings
	hasHeadings := bytes.Contains([]byte(markdownContent), []byte("# "))
	assert.True(t, hasHeadings, "Markdown should contain heading structure")

	// 4c. No mojibake
	mojibakePatterns := []string{
		"\u00e2\u20ac\u201c", // â€" (en-dash)
		"\u00e2\u20ac\u201d", // â€" (em-dash)
		"\u00e2\u20ac\u0153", // â€œ (left double quote)
		"\u00e2\u20ac\u00a6", // â€¦ (ellipsis)
	}
	for _, broken := range mojibakePatterns {
		assert.NotContains(t, markdownContent, broken,
			"Markdown should not contain UTF-8 mojibake")
	}

	// ── Step 5: Verify LLM metadata extraction ──────────────────────
	// These assertions verify the LLM service is actually working.
	// If LLM is unavailable the test MUST fail (not silently pass).
	title, _ := paper["title"].(string)
	require.NotEmpty(t, title, "LLM must extract a non-empty title from the paper")

	originalFilename, _ := paper["original_filename"].(string)
	require.NotEqual(t, title, originalFilename,
		"LLM-extracted title must differ from original filename (got %q == %q)",
		title, originalFilename)

	authors := paper["authors"]
	require.NotNil(t, authors, "LLM must extract authors from the paper")
	authorsStr, _ := authors.(string)
	require.NotEmpty(t, authorsStr, "LLM-extracted authors must not be empty")

	keywords := paper["keywords"]
	require.NotNil(t, keywords, "LLM must extract keywords from the paper")
	keywordsStr, _ := keywords.(string)
	require.NotEmpty(t, keywordsStr, "LLM-extracted keywords must not be empty")

	// ── Step 6: List papers should include our paper ───────────────
	req = httptest.NewRequest("GET", "/api/v1/papers?limit=20", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))

	papers, ok := listResp["papers"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(papers), 1)

	total, _ := listResp["total"].(float64)
	assert.GreaterOrEqual(t, total, float64(1))

	// ── Step 7: Update paper metadata ──────────────────────────────
	body := `{"title": "E2E Test: Updated RDMA Paper Title"}`
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/papers/%s", paperID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var updateResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updateResp))
	updatedPaper, _ := updateResp["paper"].(map[string]interface{})
	assert.Equal(t, "E2E Test: Updated RDMA Paper Title", updatedPaper["title"])

	// ── Step 8: Download the original PDF ──────────────────────────
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/papers/%s/download", paperID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/pdf")
	downloadedContent, err := io.ReadAll(w.Body)
	require.NoError(t, err)
	assert.Equal(t, len(pdfContent), len(downloadedContent), "Downloaded PDF size should match original")

	// ── Step 9: Delete the paper ───────────────────────────────────
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/papers/%s", paperID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it is gone
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/papers/%s", paperID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestPaperE2E_ErrorCases tests API error handling with real backend.
func TestPaperE2E_ErrorCases(t *testing.T) {
	router, cleanup := setupE2EEnv(t)
	defer cleanup()

	t.Run("upload without file returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/papers/upload", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("upload non-PDF returns 400", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("not a pdf"))
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/papers/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get nonexistent paper returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/papers/nonexistent-id-12345", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete nonexistent paper returns 404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/papers/nonexistent-id-12345", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
