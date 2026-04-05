package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// categoryRepository implements service.CategoryRepository
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *gorm.DB) service.CategoryRepository {
	return &categoryRepository{db: db}
}

// Create creates a new category
func (r *categoryRepository) Create(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

// GetByID retrieves a category by ID
func (r *categoryRepository) GetByID(ctx context.Context, id string) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, service.ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

// ListByUserID retrieves all categories for a user, optionally filtered by type
func (r *categoryRepository) ListByUserID(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
	var categories []*model.Category
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if categoryType != "" {
		query = query.Where("type = ?", categoryType)
	}
	err := query.Order("position ASC, created_at ASC").Find(&categories).Error
	return categories, err
}

// Update updates a category
func (r *categoryRepository) Update(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// Delete deletes a category by ID
func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Category{}, "id = ?", id).Error
}

// GetMaxPosition returns the maximum position value for categories of a given type for a user
func (r *categoryRepository) GetMaxPosition(ctx context.Context, userID string, categoryType string) (int, error) {
	var maxPos *int
	err := r.db.WithContext(ctx).
		Table("categories").
		Select("MAX(position)").
		Where("user_id = ? AND type = ?", userID, categoryType).
		Scan(&maxPos).Error
	if err != nil {
		return 0, err
	}
	if maxPos == nil {
		return 0, nil
	}
	return *maxPos, nil
}
