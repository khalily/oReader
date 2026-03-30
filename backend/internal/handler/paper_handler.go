package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	apperrors "github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/service"
)

// PaperHandler handles paper-related HTTP requests
type PaperHandler struct {
	paperService   service.PaperService
	maxUploadSize  int64
}

// NewPaperHandler creates a new paper handler
func NewPaperHandler(paperService service.PaperService, maxUploadSize int64) *PaperHandler {
	return &PaperHandler{
		paperService:   paperService,
		maxUploadSize:  maxUploadSize,
	}
}

// UpdatePaperRequest represents the request to update a paper
type UpdatePaperRequest struct {
	Title         *string `json:"title,omitempty"`
	Abstract      *string `json:"abstract,omitempty"`
	PublishedYear *string `json:"published_year,omitempty"`
	DOI           *string `json:"doi,omitempty"`
}

// UpdateTagsRequest represents the request to update paper tags
type UpdateTagsRequest struct {
	Tags []string `json:"tags"`
}

// UploadPaper handles POST /api/v1/papers/upload
func (h *PaperHandler) UploadPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "No file provided", nil)
		return
	}

	// S1: Enforce file size limit
	if h.maxUploadSize > 0 && file.Size > h.maxUploadSize {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation,
			fmt.Sprintf("File too large (max %d MB)", h.maxUploadSize/1024/1024), nil)
		return
	}

	// S2: Validate file extension and content type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Only PDF files are accepted", nil)
		return
	}
	contentType := file.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/pdf" && contentType != "application/x-pdf" && contentType != "application/octet-stream" {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Only PDF files are accepted", nil)
		return
	}

	src, err := file.Open()
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read uploaded file", nil)
		return
	}
	defer func() { _ = src.Close() }()

	// Read with size limit to prevent memory exhaustion
	pdfContent, err := io.ReadAll(io.LimitReader(src, h.maxUploadSize+1))
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file content", nil)
		return
	}
	if int64(len(pdfContent)) > h.maxUploadSize {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "File too large", nil)
		return
	}

	// S2: Validate PDF magic bytes (%PDF-)
	if len(pdfContent) < 5 || string(pdfContent[:5]) != "%PDF-" {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid PDF file", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("filename", file.Filename).
		Int("size", len(pdfContent)).
		Msg("UploadPaper request received")

	paper, err := h.paperService.UploadPaper(c.Request.Context(), userID.(string), file.Filename, pdfContent)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Str("filename", file.Filename).
			Msg("UploadPaper failed")
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to upload paper", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("paper_id", paper.ID).
		Msg("UploadPaper completed successfully")

	c.JSON(http.StatusAccepted, paper)
}

// ListPapers handles GET /api/v1/papers
func (h *PaperHandler) ListPapers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Support page/per_page pagination (B1 fix)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "0"))
	if page > 0 && perPage > 0 {
		limit = perPage
		offset = (page - 1) * perPage
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	opts := service.PaperListOptions{
		Limit:  limit,
		Offset: offset,
		Query:  c.Query("q"),
		Year:   c.Query("year"),
		Tag:    c.Query("tag"),
		Status: c.Query("status"),
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	}

	papers, total, err := h.paperService.ListPapers(c.Request.Context(), userID.(string), opts)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Msg("ListPapers failed")
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve papers", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"papers": papers,
		"total":  total,
	})
}

// GetPaper handles GET /api/v1/papers/:id
func (h *PaperHandler) GetPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	paper, err := h.paperService.GetPaper(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve paper", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"paper": paper})
}

// GetPaperStatus handles GET /api/v1/papers/:id/status
func (h *PaperHandler) GetPaperStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	status, err := h.paperService.GetPaperStatus(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to get paper status", nil)
		return
	}

	c.JSON(http.StatusOK, status)
}

// UpdatePaper handles PUT /api/v1/papers/:id
func (h *PaperHandler) UpdatePaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	var req UpdatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	// Build updates map from non-nil fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Abstract != nil {
		updates["abstract"] = *req.Abstract
	}
	if req.PublishedYear != nil {
		updates["published_year"] = *req.PublishedYear
	}
	if req.DOI != nil {
		updates["doi"] = *req.DOI
	}

	if len(updates) == 0 {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "No fields to update", nil)
		return
	}

	paper, err := h.paperService.UpdatePaper(c.Request.Context(), userID.(string), paperID, updates)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to update paper", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"paper": paper})
}

// UpdateTags handles PUT /api/v1/papers/:id/tags
func (h *PaperHandler) UpdateTags(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	var req UpdateTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	err := h.paperService.UpdateTags(c.Request.Context(), userID.(string), paperID, req.Tags)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to update tags", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags": req.Tags,
	})
}

// RetryPaper handles POST /api/v1/papers/:id/retry
func (h *PaperHandler) RetryPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	logger.Info().
		Str("user_id", userID.(string)).
		Str("paper_id", paperID).
		Msg("RetryPaper request received")

	paper, err := h.paperService.RetryPaper(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retry paper conversion", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"paper": paper})
}

// DeletePaper handles DELETE /api/v1/papers/:id
func (h *PaperHandler) DeletePaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	logger.Info().
		Str("user_id", userID.(string)).
		Str("paper_id", paperID).
		Msg("DeletePaper request received")

	err := h.paperService.DeletePaper(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Str("paper_id", paperID).
			Msg("DeletePaper failed")
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to delete paper", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("paper_id", paperID).
		Msg("DeletePaper completed successfully")

	c.Status(http.StatusNoContent)
}

// ListTags handles GET /api/v1/papers/tags
func (h *PaperHandler) ListTags(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	tags, err := h.paperService.ListTags(c.Request.Context(), userID.(string))
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve tags", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
	})
}

// DownloadPaper handles GET /api/v1/papers/:id/download
func (h *PaperHandler) DownloadPaper(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("id")

	pdfPath, filename, err := h.paperService.DownloadPaper(c.Request.Context(), userID.(string), paperID)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to download paper", nil)
		return
	}

	// Set appropriate headers for file download (S3: sanitize filename for header safety)
	safeName := filepath.Base(filename)
	safeName = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '"' {
			return -1
		}
		return r
	}, safeName)
	encodedName := url.QueryEscape(safeName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", safeName, encodedName))
	c.File(pdfPath)
}
