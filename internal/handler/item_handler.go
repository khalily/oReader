package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apperrors "oreader/internal/infra/errors"
	"oreader/internal/service"
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

// ToggleStarRequest represents the request to toggle star status
type ToggleStarRequest struct {
	Starred bool `json:"starred" binding:"required"`
}

// ToggleReadRequest represents the request to toggle read status
type ToggleReadRequest struct {
	Read bool `json:"read" binding:"required"`
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
		if starredStr == "true" {
			val := true
			starred = &val
		} else if starredStr == "false" {
			val := false
			starred = &val
		}
	}

	var read *bool
	if readStr := c.Query("read"); readStr != "" {
		if readStr == "true" {
			val := true
			read = &val
		} else if readStr == "false" {
			val := false
			read = &val
		}
	}

	// Validate limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	opts := service.ListItemOptions{
		Limit:   limit,
		Cursor:  cursor,
		FeedID:  feedID,
		Starred: starred,
		Read:    read,
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
func (h *ItemHandler) ToggleStar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req ToggleStarRequest
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

// ToggleRead handles PUT /api/v1/items/:id/read
func (h *ItemHandler) ToggleRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	itemID := c.Param("id")

	var req ToggleReadRequest
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
