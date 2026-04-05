package repository

import (
	"context"

	"gorm.io/gorm"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// paperRepository implements service.PaperRepository
type paperRepository struct {
	db *gorm.DB
}

// NewPaperRepository creates a new paper repository
func NewPaperRepository(db *gorm.DB) service.PaperRepository {
	return &paperRepository{db: db}
}

// Create creates a new paper
func (r *paperRepository) Create(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Create(paper).Error
}

// GetByID retrieves a paper by ID
func (r *paperRepository) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&paper).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &paper, nil
}

// ListByUserID retrieves papers for a user with search, filtering, and pagination
func (r *paperRepository) ListByUserID(ctx context.Context, userID string, opts service.PaperListOptions) ([]*model.Paper, int64, error) {
	var papers []*model.Paper
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("user_id = ?", userID)

	// Search filter: title, authors, keywords
	if opts.Query != "" {
		likeQuery := "%" + opts.Query + "%"
		query = query.Where("title LIKE ? OR authors LIKE ? OR keywords LIKE ?", likeQuery, likeQuery, likeQuery)
	}

	// Year filter
	if opts.Year != "" {
		query = query.Where("published_year = ?", opts.Year)
	}

	// Status filter
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}

	// Tag filter: join with paper_tags
	if opts.Tag != "" {
		query = query.Joins("JOIN paper_tags ON paper_tags.paper_id = papers.id AND paper_tags.deleted_at IS NULL").
			Where("paper_tags.tag = ?", opts.Tag)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	sort := "created_at"
	switch opts.Sort {
	case "title":
		sort = "title"
	case "published_year":
		sort = "published_year"
	}

	order := "DESC"
	if opts.Order == "asc" {
		order = "ASC"
	}

	query = query.Order(sort + " " + order)

	// Pagination
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}

	if err := query.Find(&papers).Error; err != nil {
		return nil, 0, err
	}

	return papers, total, nil
}

// Update updates a paper
func (r *paperRepository) Update(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Save(paper).Error
}

// Delete deletes a paper and cascades to tags
func (r *paperRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete associated tags
		if err := tx.Where("paper_id = ?", id).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		// Delete the paper
		return tx.Delete(&model.Paper{}, "id = ?", id).Error
	})
}

// ListTags returns all distinct tags for a user's papers
func (r *paperRepository) ListTags(ctx context.Context, userID string) ([]string, error) {
	var tags []string
	err := r.db.WithContext(ctx).
		Model(&model.PaperTag{}).
		Distinct("tag").
		Joins("JOIN papers ON papers.id = paper_tags.paper_id AND papers.deleted_at IS NULL").
		Where("papers.user_id = ?", userID).
		Pluck("tag", &tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// UpdateCategory updates the category_id for a paper
func (r *paperRepository) UpdateCategory(ctx context.Context, paperID string, categoryID *string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", paperID).
		Update("category_id", categoryID)
	return result.Error
}

// paperTagRepository implements service.PaperTagRepository
type paperTagRepository struct {
	db *gorm.DB
}

// NewPaperTagRepository creates a new paper tag repository
func NewPaperTagRepository(db *gorm.DB) service.PaperTagRepository {
	return &paperTagRepository{db: db}
}

// SetTags replaces all tags for a paper (delete-then-create in transaction)
func (r *paperTagRepository) SetTags(ctx context.Context, paperID string, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing tags
		if err := tx.Where("paper_id = ?", paperID).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}

		// Create new tags
		for _, tag := range tags {
			paperTag := &model.PaperTag{
				PaperID: paperID,
				Tag:     tag,
			}
			if err := paperTag.GenerateID(); err != nil {
				return err
			}
			if err := tx.Create(paperTag).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetByPaperID retrieves all tags for a paper
func (r *paperTagRepository) GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error) {
	var tags []*model.PaperTag
	err := r.db.WithContext(ctx).Where("paper_id = ?", paperID).Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}
