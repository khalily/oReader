package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"oreader/internal/model"
	"oreader/internal/service"
)

// oauthStateRepository implements service.OAuthStateRepository
type oauthStateRepository struct {
	db *gorm.DB
}

// NewOAuthStateRepository creates a new OAuth state repository
func NewOAuthStateRepository(db *gorm.DB) service.OAuthStateRepository {
	return &oauthStateRepository{db: db}
}

// Create creates a new OAuth state
func (r *oauthStateRepository) Create(ctx context.Context, state *model.OAuthState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

// GetByState retrieves an OAuth state by its value
func (r *oauthStateRepository) GetByState(ctx context.Context, state string) (*model.OAuthState, error) {
	var oauthState model.OAuthState
	err := r.db.WithContext(ctx).First(&oauthState, "state = ?", state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &oauthState, nil
}

// Delete deletes an OAuth state by its value
func (r *oauthStateRepository) Delete(ctx context.Context, state string) error {
	return r.db.WithContext(ctx).Delete(&model.OAuthState{}, "state = ?", state).Error
}

// DeleteExpired deletes all expired OAuth states
func (r *oauthStateRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&model.OAuthState{}).Error
}
