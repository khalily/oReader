package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// pendingOAuthRepository implements service.PendingOAuthRepository
type pendingOAuthRepository struct {
	db *gorm.DB
}

// NewPendingOAuthRepository creates a new PendingOAuth repository
func NewPendingOAuthRepository(db *gorm.DB) service.PendingOAuthRepository {
	return &pendingOAuthRepository{db: db}
}

// Create creates a new PendingOAuth record
func (r *pendingOAuthRepository) Create(ctx context.Context, pending *model.PendingOAuth) error {
	return r.db.WithContext(ctx).Create(pending).Error
}

// GetByToken retrieves a PendingOAuth by token
func (r *pendingOAuthRepository) GetByToken(ctx context.Context, token string) (*model.PendingOAuth, error) {
	var pending model.PendingOAuth
	err := r.db.WithContext(ctx).First(&pending, "token = ?", token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pending, nil
}

// Delete deletes a PendingOAuth by token
func (r *pendingOAuthRepository) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Delete(&model.PendingOAuth{}, "token = ?", token).Error
}
