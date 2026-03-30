package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/service"
)

// ItemHandler handles item-related HTTP requests
type ItemHandler struct {
	itemService service.ItemService
}

// NewItemHandler creates a new item handler
func NewItemHandler(itemService service.ItemService) *ItemHandler {
	return &ItemHandler{
		itemService: itemService,
	}
}

// SetStarRequest represents the request to set star status
type SetStarRequest struct {
	Starred bool `json:"starred"`
}

// SetReadRequest represents the request to set read status
type SetReadRequest struct {
	Read bool `json:"read"`
}

// ListItems handles GET /api/v1/items
func (h *ItemHandler) ListItems(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cursor := c.Query("cursor")
	feedID := c.Query("feed_id")

	var starred *bool
	if starredStr := c.Query("starred"); starredStr != "" {
		switch starredStr {
		case "true":
			val := true
			starred = &val
		case "false":
			val := false
			starred = &val
		}
	}

	var read *bool
	if readStr := c.Query("read"); readStr != "" {
		switch readStr {
		case "true":
			val := true
			read = &val
		case "false":
			val := false
			read = &val
		}
	}

	var publishedToday *bool
	if publishedTodayStr := c.Query("published_today"); publishedTodayStr != "" {
		switch publishedTodayStr {
		case "true":
			val := true
			publishedToday = &val
		case "false":
			val := false
			publishedToday = &val
		}
	}

	// Validate limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	opts := service.ListItemOptions{
		Limit:          limit,
		Cursor:         cursor,
		FeedID:         feedID,
		Starred:        starred,
		Read:           read,
		PublishedToday: publishedToday,
	}

	result, err := h.itemService.ListItems(c.Request.Context(), userID.(string), opts)
	if err != nil {
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve items", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      result.Items,
		"total":      result.Total,
		"has_more":   result.HasMore,
		"next_cursor": result.NextCursor,
	})
}

// GetItem handles GET /api/v1/items/:id
func (h *ItemHandler) GetItem(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	item, err := h.itemService.GetItem(c.Request.Context(), userID.(string), itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Item not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve item", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

// ToggleStar handles PUT /api/v1/items/:id/star
// Deprecated: Use SetStar instead for spec-compliant behavior
func (h *ItemHandler) ToggleStar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req SetStarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	// Note: The service toggles regardless of the request body
	// But we validate the body to ensure it's well-formed
	item, err := h.itemService.ToggleStar(c.Request.Context(), userID.(string), itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Item not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to toggle star status", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

// SetStar handles PUT /api/v1/items/:id/star (spec-compliant: sets value from request body)
func (h *ItemHandler) SetStar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req SetStarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("item_id", itemID).
		Bool("starred", req.Starred).
		Msg("SetStar request received")

	// Set star status to the value from request body (spec-compliant)
	item, err := h.itemService.SetStar(c.Request.Context(), userID.(string), itemID, req.Starred)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Str("item_id", itemID).
			Msg("SetStar failed")
		if errors.Is(err, service.ErrItemNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Item not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to set star status", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

// ToggleRead handles PUT /api/v1/items/:id/read
// Deprecated: Use SetRead instead for spec-compliant behavior
func (h *ItemHandler) ToggleRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req SetReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	// Note: The service toggles regardless of the request body
	// But we validate the body to ensure it's well-formed
	item, err := h.itemService.ToggleRead(c.Request.Context(), userID.(string), itemID)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Item not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to toggle read status", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

// SetRead handles PUT /api/v1/items/:id/read (spec-compliant: sets value from request body)
func (h *ItemHandler) SetRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req SetReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	// Set read status to the value from request body (spec-compliant)
	item, err := h.itemService.SetRead(c.Request.Context(), userID.(string), itemID, req.Read)
	if err != nil {
		if errors.Is(err, service.ErrItemNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Item not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to set read status", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
	})
}

// MarkAllRead handles POST /api/v1/feeds/:id/read-all
func (h *ItemHandler) MarkAllRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("id")

	count, err := h.itemService.MarkAllRead(c.Request.Context(), userID.(string), feedID)
	if err != nil {
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to mark items as read", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}
