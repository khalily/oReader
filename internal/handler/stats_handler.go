package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "oreader/internal/infra/errors"
	"oreader/internal/service"
)

// StatsHandler handles stats-related HTTP requests
type StatsHandler struct {
	statsService service.StatsService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(statsService service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetStats handles GET /api/v1/stats
func (h *StatsHandler) GetStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	stats, err := h.statsService.GetUserStats(c.Request.Context(), userID.(string))
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve stats", nil)
		return
	}

	c.JSON(http.StatusOK, stats)
}
