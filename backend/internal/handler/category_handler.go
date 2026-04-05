package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/service"
)

// CategoryHandler handles category-related HTTP requests
type CategoryHandler struct {
	categoryService service.CategoryService
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// CreateCategoryRequest represents the request to create a category
type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,max=100"`
	Type string `json:"type" binding:"required,oneof=feed paper"`
}

// RenameCategoryRequest represents the request to rename a category
type RenameCategoryRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

// MoveToCategoryRequest represents the request to move a feed/paper to a category
type MoveToCategoryRequest struct {
	CategoryID string `json:"category_id"`
}

// CreateCategory handles POST /api/v1/categories
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("name", req.Name).
		Str("type", req.Type).
		Msg("CreateCategory request received")

	category, err := h.categoryService.CreateCategory(c.Request.Context(), userID.(string), req.Name, req.Type)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID.(string)).Msg("CreateCategory failed")
		if errors.Is(err, service.ErrCategoryDuplicate) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Category with this name already exists", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to create category", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("category_id", category.ID).
		Msg("CreateCategory completed successfully")

	c.JSON(http.StatusCreated, gin.H{
		"category": category,
	})
}

// ListCategories handles GET /api/v1/categories?type=feed
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryType := c.Query("type")

	logger.Debug().
		Str("user_id", userID.(string)).
		Str("type", categoryType).
		Msg("ListCategories request received")

	categories, err := h.categoryService.ListCategories(c.Request.Context(), userID.(string), categoryType)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID.(string)).Msg("ListCategories failed")
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to retrieve categories", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
	})
}

// RenameCategory handles PUT /api/v1/categories/:id/rename
func (h *CategoryHandler) RenameCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryID := c.Param("id")

	var req RenameCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("category_id", categoryID).
		Str("new_name", req.Name).
		Msg("RenameCategory request received")

	category, err := h.categoryService.RenameCategory(c.Request.Context(), userID.(string), categoryID, req.Name)
	if err != nil {
		logger.Error().Err(err).Str("category_id", categoryID).Msg("RenameCategory failed")
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryDuplicate) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Category with this name already exists", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to rename category", nil)
		return
	}

	logger.Info().
		Str("category_id", categoryID).
		Str("new_name", req.Name).
		Msg("RenameCategory completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"category": category,
	})
}

// DeleteCategory handles DELETE /api/v1/categories/:id
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryID := c.Param("id")

	logger.Info().
		Str("user_id", userID.(string)).
		Str("category_id", categoryID).
		Msg("DeleteCategory request received")

	err := h.categoryService.DeleteCategory(c.Request.Context(), userID.(string), categoryID)
	if err != nil {
		logger.Error().Err(err).Str("category_id", categoryID).Msg("DeleteCategory failed")
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to delete category", nil)
		return
	}

	logger.Info().
		Str("category_id", categoryID).
		Msg("DeleteCategory completed successfully")

	c.Status(http.StatusNoContent)
}

// MoveFeedToCategory handles PUT /api/v1/categories/feeds/:feedId
func (h *CategoryHandler) MoveFeedToCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("feedId")

	var req MoveToCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("feed_id", feedID).
		Str("category_id", req.CategoryID).
		Msg("MoveFeedToCategory request received")

	err := h.categoryService.MoveFeedToCategory(c.Request.Context(), userID.(string), feedID, req.CategoryID)
	if err != nil {
		logger.Error().Err(err).Str("feed_id", feedID).Msg("MoveFeedToCategory failed")
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to move feed to category", nil)
		return
	}

	logger.Info().
		Str("feed_id", feedID).
		Str("category_id", req.CategoryID).
		Msg("MoveFeedToCategory completed successfully")

	c.Status(http.StatusNoContent)
}

// MovePaperToCategory handles PUT /api/v1/categories/papers/:paperId
func (h *CategoryHandler) MovePaperToCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("paperId")

	var req MoveToCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("paper_id", paperID).
		Str("category_id", req.CategoryID).
		Msg("MovePaperToCategory request received")

	err := h.categoryService.MovePaperToCategory(c.Request.Context(), userID.(string), paperID, req.CategoryID)
	if err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("MovePaperToCategory failed")
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to move paper to category", nil)
		return
	}

	logger.Info().
		Str("paper_id", paperID).
		Str("category_id", req.CategoryID).
		Msg("MovePaperToCategory completed successfully")

	c.Status(http.StatusNoContent)
}
