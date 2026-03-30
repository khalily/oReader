package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/opml"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// ImportHandler handles import/export HTTP requests
type ImportHandler struct {
	feedService   service.FeedService
	importService service.ImportService
	feedRepo      service.FeedRepository
}

// NewImportHandler creates a new import/export handler
func NewImportHandler(
	feedService service.FeedService,
	importService service.ImportService,
	feedRepo service.FeedRepository,
) *ImportHandler {
	return &ImportHandler{
		feedService:   feedService,
		importService: importService,
		feedRepo:      feedRepo,
	}
}

// ExportFeeds handles GET /api/v1/feeds/export
func (h *ImportHandler) ExportFeeds(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	// Get all user feeds
	feeds, _, err := h.feedService.GetUserFeeds(c.Request.Context(), userID.(string), service.ListOptions{Limit: 10000})
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve feeds", nil)
		return
	}

	// Convert to []*model.Feed
	modelFeeds := make([]*model.Feed, len(feeds))
	for i, f := range feeds {
		modelFeeds[i] = f.Feed
	}

	// Generate OPML
	output, err := opml.Export(modelFeeds)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to generate OPML", nil)
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/xml")
	c.Header("Content-Disposition", `attachment; filename="oreader-subscriptions.xml"`)

	// Send the OPML content
	c.Data(http.StatusOK, "application/xml", output)
}

// ImportFeeds handles POST /api/v1/feeds/import
func (h *ImportHandler) ImportFeeds(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	// Read the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "No file uploaded", nil)
		return
	}

	// Check file size (max 1MB)
	const maxFileSize = 1 * 1024 * 1024
	if file.Size > maxFileSize {
		apperrors.SendError(c, http.StatusRequestEntityTooLarge, apperrors.ErrValidation, "File too large (max 1MB)", nil)
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file", nil)
		return
	}
	defer src.Close()

	// Read file content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(src, maxFileSize)); err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to read file", nil)
		return
	}
	content := buf.String()

	// Parse OPML
	feeds, err := h.importService.ParseOPML(c.Request.Context(), content)
	if err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid OPML format", nil)
		return
	}

	// Check if we have any feeds to import
	if len(feeds) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No feeds found in OPML file",
			"job":     nil,
		})
		return
	}

	// Start async import job
	job, err := h.importService.StartImport(c.Request.Context(), userID.(string), feeds)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to start import", nil)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Import job started",
		"job_id":      job.ID,
		"total_feeds": job.TotalFeeds,
		"status":      job.Status,
	})
}

// GetImportStatus handles GET /api/v1/feeds/import/:job_id/status
func (h *ImportHandler) GetImportStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	jobID := c.Param("job_id")

	// Get job status
	job, err := h.importService.GetJobStatus(c.Request.Context(), jobID)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to get job status", nil)
		return
	}
	if job == nil {
		apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Import job not found", nil)
		return
	}

	// Verify the job belongs to the user
	if job.UserID != userID.(string) {
		apperrors.SendError(c, http.StatusForbidden, apperrors.ErrForbidden, "Access denied", nil)
		return
	}

	// Calculate progress percentage
	progress := 0
	if job.TotalFeeds > 0 {
		progress = (job.Processed * 100) / job.TotalFeeds
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":     job.ID,
		"status":     job.Status,
		"total":      job.TotalFeeds,
		"processed":  job.Processed,
		"failed":     job.Failed,
		"progress":   progress,
		"started_at": job.StartedAt,
		"ended_at":   job.EndedAt,
		"error":      job.Error,
	})
}
