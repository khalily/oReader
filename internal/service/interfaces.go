package service

import (
	"context"
	"errors"

	"oreader/internal/model"
)

// Errors
var (
	// ErrFeedNotFound is returned when a feed is not found
	ErrFeedNotFound = errors.New("feed not found")
	// ErrInvalidFeedURL is returned when the feed URL is invalid
	ErrInvalidFeedURL = errors.New("invalid feed URL")
	// ErrFeedFetchFailed is returned when fetching the feed fails
	ErrFeedFetchFailed = errors.New("failed to fetch feed")
	// ErrFeedParseFailed is returned when parsing the feed fails
	ErrFeedParseFailed = errors.New("failed to parse feed")
	// ErrFeedAlreadySubscribed is returned when user is already subscribed to the feed
	ErrFeedAlreadySubscribed = errors.New("already subscribed to this feed")
	// ErrItemNotFound is returned when an item is not found
	ErrItemNotFound = errors.New("item not found")
)

// ItemWithState represents an item with user-specific state
type ItemWithState struct {
	*model.Item
	IsStarred bool    `json:"is_starred"`
	IsRead    bool    `json:"is_read"`
	ReadAt    *string `json:"read_at,omitempty"`
}

// ListOptions defines pagination and filtering options for feeds
type ListOptions struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Cursor string `json:"cursor,omitempty"`
}

// ListItemOptions defines options for listing items
type ListItemOptions struct {
	Limit   int    `json:"limit"`
	Cursor  string `json:"cursor,omitempty"`
	FeedID  string `json:"feed_id,omitempty"`
	Starred *bool  `json:"starred,omitempty"`
	Read    *bool  `json:"read,omitempty"`
}

// ItemListResult contains the result of listing items with pagination metadata
type ItemListResult struct {
	Items      []*ItemWithState `json:"items"`
	Total      int64            `json:"total"`
	HasMore    bool             `json:"has_more"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// FeedService defines the interface for feed business logic
type FeedService interface {
	// Subscribe subscribes a user to a feed
	Subscribe(ctx context.Context, userID, feedURL string) (*SubscribeResult, error)
	// GetUserFeeds retrieves all feeds for a user with item counts
	GetUserFeeds(ctx context.Context, userID string, opts ListOptions) ([]*FeedWithItemCount, int64, error)
	// GetFeed retrieves a specific feed for a user
	GetFeed(ctx context.Context, userID, feedID string) (*model.Feed, int, error)
	// DeleteFeed deletes a feed subscription for a user
	DeleteFeed(ctx context.Context, userID, feedID string) error
	// RefreshFeed manually refreshes a feed
	RefreshFeed(ctx context.Context, userID, feedID string) (*RefreshResult, error)
}

// ItemService defines the interface for item business logic
type ItemService interface {
	// ListItems retrieves items for a user with filtering and pagination
	ListItems(ctx context.Context, userID string, opts ListItemOptions) (*ItemListResult, error)
	// GetItem retrieves a single item by ID with user state
	GetItem(ctx context.Context, userID, itemID string) (*ItemWithState, error)
	// ToggleStar toggles the star status for an item
	ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error)
	// ToggleRead toggles the read status for an item
	ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error)
	// MarkAllRead marks all items in a feed as read for a user
	MarkAllRead(ctx context.Context, userID, feedID string) (int, error)
}

// SubscribeResult contains the result of subscribing to a feed
type SubscribeResult struct {
	Feed         *model.Feed
	NewItemCount int
}

// FeedWithItemCount contains a feed with its item count
type FeedWithItemCount struct {
	*model.Feed
	ItemCount int `json:"item_count"`
}

// RefreshResult contains the result of refreshing a feed
type RefreshResult struct {
	NewItemCount int `json:"new_item_count"`
}

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
