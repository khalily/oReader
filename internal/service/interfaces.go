package service

import (
	"context"

	"oreader/internal/model"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByGitHubID(ctx context.Context, githubID string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

// FeedRepository defines the interface for feed data access
type FeedRepository interface {
	Create(ctx context.Context, feed *model.Feed) error
	GetByID(ctx context.Context, id string) (*model.Feed, error)
	GetByURL(ctx context.Context, url string) (*model.Feed, error)
	ListByUserID(ctx context.Context, userID string, opts ListOptions) ([]*model.Feed, int64, error)
	Update(ctx context.Context, feed *model.Feed) error
	Delete(ctx context.Context, id string) error
	ListAll(ctx context.Context) ([]*model.Feed, error)
}

// ItemRepository defines the interface for item data access
type ItemRepository interface {
	Create(ctx context.Context, item *model.Item) error
	CreateBatch(ctx context.Context, items []*model.Item) error
	GetByID(ctx context.Context, id string) (*model.Item, error)
	GetByGUID(ctx context.Context, feedID, guid string) (*model.Item, error)
	ListByFeedID(ctx context.Context, feedID string, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	ListStarred(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	ListUnread(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	CountByFeedID(ctx context.Context, feedID string) (int64, error)
}

// UserFeedRepository defines the interface for user-feed relationship data access
type UserFeedRepository interface {
	Create(ctx context.Context, userFeed *model.UserFeed) error
	GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error)
	Delete(ctx context.Context, userID, feedID string) error
	GetMaxPosition(ctx context.Context, userID string) (int, error)
}

// UserItemStateRepository defines the interface for user item state data access
type UserItemStateRepository interface {
	Create(ctx context.Context, state *model.UserItemState) error
	GetByUserAndItem(ctx context.Context, userID, itemID string) (*model.UserItemState, error)
	Update(ctx context.Context, state *model.UserItemState) error
	Upsert(ctx context.Context, state *model.UserItemState) error
	BulkMarkRead(ctx context.Context, userID string, itemIDs []string) error
	MarkAllRead(ctx context.Context, userID string, feedID string) error
}

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllByUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// ImportJobRepository defines the interface for import job data access
type ImportJobRepository interface {
	Create(ctx context.Context, job *model.ImportJob) error
	GetByID(ctx context.Context, id string) (*model.ImportJob, error)
	GetByUserID(ctx context.Context, userID string) ([]*model.ImportJob, error)
	Update(ctx context.Context, job *model.ImportJob) error
}

// OAuthStateRepository defines the interface for OAuth state data access
type OAuthStateRepository interface {
	Create(ctx context.Context, state *model.OAuthState) error
	GetByState(ctx context.Context, state string) (*model.OAuthState, error)
	Delete(ctx context.Context, state string) error
	DeleteExpired(ctx context.Context) error
}

// ListOptions defines pagination and filtering options
type ListOptions struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Cursor string `json:"cursor,omitempty"`
}

// ItemWithState represents an item with user-specific state
type ItemWithState struct {
	*model.Item
	IsStarred bool       `json:"is_starred"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *string    `json:"read_at,omitempty"`
}
