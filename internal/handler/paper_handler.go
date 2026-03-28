package handler

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "oreader/internal/infra/errors"
	"oreader/internal/infra/logger"
	"oreader/internal/service"
)

// PaperHandler handles paper-related HTTP requests
type PaperHandler struct {
	paperService service.PaperService
}

// NewPaperHandler creates a new paper handler
func NewPaperHandler(paperService service.PaperService) *PaperHandler {
	return &PaperHandler{
		paperService: paperService,
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

	src, err := file.Open()
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read uploaded file", nil)
		return
	}
	defer src.Close()

	pdfContent, err := io.ReadAll(src)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file content", nil)
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

	c.JSON(http.StatusOK, paper)
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

	c.JSON(http.StatusOK, paper)
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

	c.JSON(http.StatusOK, paper)
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

	// Set appropriate headers for file download
	c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(filename)+"\"")
	c.File(pdfPath)
}
