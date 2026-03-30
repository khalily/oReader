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

// FeedHandler handles feed-related HTTP requests
type FeedHandler struct {
	feedService service.FeedService
}

// NewFeedHandler creates a new feed handler
func NewFeedHandler(feedService service.FeedService) *FeedHandler {
	return &FeedHandler{
		feedService: feedService,
	}
}

// CreateFeedRequest represents the request to create a feed subscription
type CreateFeedRequest struct {
	FeedURL string `json:"feed_url" binding:"required"`
}

// CreateFeed handles POST /api/v1/feeds
func (h *FeedHandler) CreateFeed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	var req CreateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("feed_url", req.FeedURL).
		Msg("CreateFeed request received")

	result, err := h.feedService.Subscribe(c.Request.Context(), userID.(string), req.FeedURL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Str("feed_url", req.FeedURL).
			Msg("CreateFeed failed")
		if errors.Is(err, service.ErrInvalidFeedURL) || errors.Is(err, service.ErrFeedFetchFailed) || errors.Is(err, service.ErrFeedParseFailed) {
			apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrFeedAlreadySubscribed) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Already subscribed to this feed", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to subscribe to feed", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("feed_id", result.Feed.ID).
		Int("new_item_count", result.NewItemCount).
		Msg("CreateFeed completed successfully")

	c.JSON(http.StatusCreated, gin.H{
		"feed":           result.Feed,
		"new_item_count": result.NewItemCount,
	})
}

// ListFeeds handles GET /api/v1/feeds
func (h *FeedHandler) ListFeeds(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	logger.Debug().
		Str("user_id", userID.(string)).
		Msg("ListFeeds request received")

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	opts := service.ListOptions{
		Limit:  limit,
		Offset: offset,
	}

	feeds, total, err := h.feedService.GetUserFeeds(c.Request.Context(), userID.(string), opts)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Msg("ListFeeds failed")
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve feeds", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feeds": feeds,
		"total": total,
	})
}

// GetFeed handles GET /api/v1/feeds/:id
func (h *FeedHandler) GetFeed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("id")

	feed, itemCount, err := h.feedService.GetFeed(c.Request.Context(), userID.(string), feedID)
	if err != nil {
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve feed", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feed":       feed,
		"item_count": itemCount,
	})
}

// DeleteFeed handles DELETE /api/v1/feeds/:id
func (h *FeedHandler) DeleteFeed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("id")

	logger.Info().
		Str("user_id", userID.(string)).
		Str("feed_id", feedID).
		Msg("DeleteFeed request received")

	err := h.feedService.DeleteFeed(c.Request.Context(), userID.(string), feedID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID.(string)).
			Str("feed_id", feedID).
			Msg("DeleteFeed failed")
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		if errors.Is(err, service.ErrFeedAlreadyUnsubscribed) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Already unsubscribed from this feed", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to delete feed", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("feed_id", feedID).
		Msg("DeleteFeed completed successfully")

	c.Status(http.StatusNoContent)
}

// RefreshFeed handles POST /api/v1/feeds/:id/refresh
func (h *FeedHandler) RefreshFeed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("id")

	result, err := h.feedService.RefreshFeed(c.Request.Context(), userID.(string), feedID)
	if err != nil {
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		if errors.Is(err, service.ErrInvalidFeedURL) || errors.Is(err, service.ErrFeedFetchFailed) || errors.Is(err, service.ErrFeedParseFailed) {
			apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, err.Error(), nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to refresh feed", nil)
		return
	}

	c.JSON(http.StatusOK, result)
}
