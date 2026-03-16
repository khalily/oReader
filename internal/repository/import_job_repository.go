package repository

import (
	"context"

	"gorm.io/gorm"

	"oreader/internal/model"
	"oreader/internal/service"
)

// importJobRepository implements service.ImportJobRepository
type importJobRepository struct {
	db *gorm.DB
}

// NewImportJobRepository creates a new import job repository
func NewImportJobRepository(db *gorm.DB) service.ImportJobRepository {
	return &importJobRepository{db: db}
}

// Create creates a new import job
func (r *importJobRepository) Create(ctx context.Context, job *model.ImportJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

// GetByID retrieves an import job by its ID
func (r *importJobRepository) GetByID(ctx context.Context, id string) (*model.ImportJob, error) {
	var job model.ImportJob
	err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetByUserID retrieves all import jobs for a user
func (r *importJobRepository) GetByUserID(ctx context.Context, userID string) ([]*model.ImportJob, error) {
	var jobs []*model.ImportJob
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// Update updates an import job
func (r *importJobRepository) Update(ctx context.Context, job *model.ImportJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}
