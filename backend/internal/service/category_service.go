package service

import (
	"context"
	"strings"

	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/model"
)

type categoryService struct {
	catRepo      CategoryRepository
	userFeedRepo UserFeedRepository
	paperRepo    PaperRepository
}

// NewCategoryService creates a new category service
func NewCategoryService(
	catRepo CategoryRepository,
	userFeedRepo UserFeedRepository,
	paperRepo PaperRepository,
) CategoryService {
	return &categoryService{
		catRepo:      catRepo,
		userFeedRepo: userFeedRepo,
		paperRepo:    paperRepo,
	}
}

// CreateCategory creates a new category for a user
func (s *categoryService) CreateCategory(ctx context.Context, userID string, name string, categoryType string) (*model.Category, error) {
	logger.Debug().
		Str("user_id", userID).
		Str("name", name).
		Str("type", categoryType).
		Msg("Creating category")

	maxPos, err := s.catRepo.GetMaxPosition(ctx, userID, categoryType)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get max category position")
		return nil, err
	}

	category := &model.Category{
		UserID:   userID,
		Name:     name,
		Type:     categoryType,
		Position: maxPos + 1,
	}
	if err := category.GenerateID(); err != nil {
		logger.Error().Err(err).Msg("Failed to generate category ID")
		return nil, err
	}

	if err := s.catRepo.Create(ctx, category); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate") {
			logger.Warn().
				Str("user_id", userID).
				Str("name", name).
				Str("type", categoryType).
				Msg("Duplicate category name")
			return nil, ErrCategoryDuplicate
		}
		logger.Error().Err(err).Msg("Failed to create category")
		return nil, err
	}

	logger.Info().
		Str("category_id", category.ID).
		Str("user_id", userID).
		Str("name", name).
		Str("type", categoryType).
		Msg("Category created")

	return category, nil
}

// ListCategories lists categories for a user, optionally filtered by type
func (s *categoryService) ListCategories(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
	logger.Debug().
		Str("user_id", userID).
		Str("type", categoryType).
		Msg("Listing categories")

	categories, err := s.catRepo.ListByUserID(ctx, userID, categoryType)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list categories")
		return nil, err
	}

	return categories, nil
}

// RenameCategory renames an existing category
func (s *categoryService) RenameCategory(ctx context.Context, userID, categoryID, newName string) (*model.Category, error) {
	logger.Debug().
		Str("user_id", userID).
		Str("category_id", categoryID).
		Str("new_name", newName).
		Msg("Renaming category")

	category, err := s.catRepo.GetByID(ctx, userID, categoryID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("category_id", categoryID).
			Msg("Category not found")
		return nil, err
	}

	category.Name = newName

	if err := s.catRepo.Update(ctx, category); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate") {
			logger.Warn().
				Str("user_id", userID).
				Str("new_name", newName).
				Msg("Duplicate category name on rename")
			return nil, ErrCategoryDuplicate
		}
		logger.Error().Err(err).Msg("Failed to update category")
		return nil, err
	}

	logger.Info().
		Str("category_id", categoryID).
		Str("new_name", newName).
		Msg("Category renamed")

	return category, nil
}

// DeleteCategory deletes a category. Associated feeds/papers have their category_id set to NULL
// via DB constraint (ON DELETE SET NULL).
func (s *categoryService) DeleteCategory(ctx context.Context, userID, categoryID string) error {
	logger.Debug().
		Str("user_id", userID).
		Str("category_id", categoryID).
		Msg("Deleting category")

	// Verify category exists and belongs to user
	if _, err := s.catRepo.GetByID(ctx, userID, categoryID); err != nil {
		logger.Warn().
			Err(err).
			Str("category_id", categoryID).
			Msg("Category not found for deletion")
		return err
	}

	if err := s.catRepo.Delete(ctx, userID, categoryID); err != nil {
		logger.Error().Err(err).Msg("Failed to delete category")
		return err
	}

	logger.Info().
		Str("category_id", categoryID).
		Str("user_id", userID).
		Msg("Category deleted")

	return nil
}

// MoveFeedToCategory assigns a feed to a category, or removes it from a category if categoryID is empty.
func (s *categoryService) MoveFeedToCategory(ctx context.Context, userID, feedID, categoryID string) error {
	logger.Debug().
		Str("user_id", userID).
		Str("feed_id", feedID).
		Str("category_id", categoryID).
		Msg("Moving feed to category")

	// Validate user owns the feed
	if _, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID); err != nil {
		return err
	}

	// Empty categoryID means uncategorize (remove from category)
	if categoryID == "" {
		if err := s.userFeedRepo.UpdateCategory(ctx, userID, feedID, nil); err != nil {
			logger.Error().Err(err).Msg("Failed to remove feed from category")
			return err
		}

		logger.Info().
			Str("user_id", userID).
			Str("feed_id", feedID).
			Msg("Feed removed from category")
		return nil
	}

	// Validate category exists and is a feed category
	category, err := s.catRepo.GetByID(ctx, userID, categoryID)
	if err != nil {
		return err
	}
	if category.Type != model.CategoryTypeFeed {
		logger.Warn().
			Str("category_id", categoryID).
			Str("type", category.Type).
			Msg("Category is not a feed category")
		return ErrCategoryNotFound
	}

	catID := categoryID
	if err := s.userFeedRepo.UpdateCategory(ctx, userID, feedID, &catID); err != nil {
		logger.Error().Err(err).Msg("Failed to update feed category")
		return err
	}

	logger.Info().
		Str("user_id", userID).
		Str("feed_id", feedID).
		Str("category_id", categoryID).
		Msg("Feed moved to category")

	return nil
}

// MovePaperToCategory assigns a paper to a category, or removes it from a category if categoryID is empty.
func (s *categoryService) MovePaperToCategory(ctx context.Context, userID, paperID, categoryID string) error {
	logger.Debug().
		Str("user_id", userID).
		Str("paper_id", paperID).
		Str("category_id", categoryID).
		Msg("Moving paper to category")

	// Validate paper exists and belongs to user
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get paper")
		return err
	}
	if paper == nil || paper.UserID != userID {
		logger.Warn().
			Str("paper_id", paperID).
			Str("user_id", userID).
			Msg("Paper not found or does not belong to user")
		return ErrPaperNotFound
	}

	// Empty categoryID means uncategorize (remove from category)
	if categoryID == "" {
		if err := s.paperRepo.UpdateCategory(ctx, paperID, nil); err != nil {
			logger.Error().Err(err).Msg("Failed to remove paper from category")
			return err
		}

		logger.Info().
			Str("user_id", userID).
			Str("paper_id", paperID).
			Msg("Paper removed from category")
		return nil
	}

	// Validate category exists and is a paper category
	category, err := s.catRepo.GetByID(ctx, userID, categoryID)
	if err != nil {
		return err
	}
	if category.Type != model.CategoryTypePaper {
		logger.Warn().
			Str("category_id", categoryID).
			Str("type", category.Type).
			Msg("Category is not a paper category")
		return ErrCategoryNotFound
	}

	catID := categoryID
	if err := s.paperRepo.UpdateCategory(ctx, paperID, &catID); err != nil {
		logger.Error().Err(err).Msg("Failed to update paper category")
		return err
	}

	logger.Info().
		Str("user_id", userID).
		Str("paper_id", paperID).
		Str("category_id", categoryID).
		Msg("Paper moved to category")

	return nil
}
