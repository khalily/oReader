package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"oreader/internal/model"
)

// Errors
var (
	// ErrFeedNotFound is returned when a feed is not found
	ErrFeedNotFound = errors.New("feed not found")
	// ErrFeedAlreadyUnsubscribed is returned when trying to delete an already unsubscribed feed
	ErrFeedAlreadyUnsubscribed = errors.New("already unsubscribed from this feed")
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
	Feed      *FeedResponse `json:"feed"`
	UserState *UserItemStateResponse `json:"user_state"`
}

// FeedResponse represents feed information in API responses
type FeedResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	FeedURL     string  `json:"feed_url"`
	Description string  `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

// UserItemStateResponse represents user-specific item state in API responses
type UserItemStateResponse struct {
	ItemID    string  `json:"item_id"`
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
	Limit          int    `json:"limit"`
	Cursor         string `json:"cursor,omitempty"`
	FeedID         string `json:"feed_id,omitempty"`
	Starred        *bool  `json:"starred,omitempty"`
	Read           *bool  `json:"read,omitempty"`
	PublishedToday *bool  `json:"published_today,omitempty"`
}

// UserStats represents statistics for a user's articles
type UserStats struct {
	Total   int64 `json:"total"`
	Unread  int64 `json:"unread"`
	Starred int64 `json:"starred"`
	Today   int64 `json:"today"`
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
	// Deprecated: Use SetStar instead for spec-compliant behavior
	ToggleStar(ctx context.Context, userID, itemID string) (*ItemWithState, error)
	// ToggleRead toggles the read status for an item
	// Deprecated: Use SetRead instead for spec-compliant behavior
	ToggleRead(ctx context.Context, userID, itemID string) (*ItemWithState, error)
	// SetStar sets the star status for an item to the specified value (spec-compliant: sets, doesn't toggle)
	SetStar(ctx context.Context, userID, itemID string, starred bool) (*ItemWithState, error)
	// SetRead sets the read status for an item to the specified value (spec-compliant: sets, doesn't toggle)
	SetRead(ctx context.Context, userID, itemID string, read bool) (*ItemWithState, error)
	// MarkAllRead marks all items in a feed as read for a user
	MarkAllRead(ctx context.Context, userID, feedID string) (int, error)
}

// RefreshWorkerService defines the interface for background feed refresh
type RefreshWorkerService interface {
	// RefreshAllFeeds refreshes all feeds concurrently
	RefreshAllFeeds(ctx context.Context) (*RefreshAllResult, error)
	// RefreshSingleFeed refreshes a single feed
	RefreshSingleFeed(ctx context.Context, feed *model.Feed) (SingleFeedResult, error)
}

// ImportService defines the interface for OPML import operations
type ImportService interface {
	// ParseOPML parses an OPML file and returns feed information
	ParseOPML(ctx context.Context, content string) ([]*FeedInfo, error)
	// StartImport starts an async import job for the user
	StartImport(ctx context.Context, userID string, feeds []*FeedInfo) (*model.ImportJob, error)
	// GetJobStatus retrieves the status of an import job
	GetJobStatus(ctx context.Context, jobID string) (*model.ImportJob, error)
	// ProcessImport processes an import job (called by worker)
	ProcessImport(ctx context.Context, jobID string) error
}

// FeedInfo contains information about a feed from OPML
type FeedInfo struct {
	Title   string
	FeedURL string
	SiteURL string
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

// RefreshAllResult contains the result of refreshing all feeds
type RefreshAllResult struct {
	TotalFeeds    int               `json:"total_feeds"`
	SuccessCount  int               `json:"success_count"`
	FailureCount  int               `json:"failure_count"`
	TotalItems    int               `json:"total_items"`
	NewItems      int               `json:"new_items"`
	Duration      time.Duration     `json:"duration"`
	FailedFeedIDs []string          `json:"failed_feed_ids,omitempty"`
	Results       []SingleFeedResult `json:"results,omitempty"`
}

// String returns a string representation of the refresh result
func (r *RefreshAllResult) String() string {
	duration := r.Duration.String()
	if r.Duration.Seconds() >= 1 {
		duration = fmt.Sprintf("%.0fs", r.Duration.Seconds())
	} else if r.Duration.Milliseconds() >= 1 {
		duration = fmt.Sprintf("%dms", r.Duration.Milliseconds())
	}
	return fmt.Sprintf("%d feeds, %d success, %d failures, %d total items, %d new items, %s duration",
		r.TotalFeeds, r.SuccessCount, r.FailureCount, r.TotalItems, r.NewItems, duration)
}

// SingleFeedResult contains the result of refreshing a single feed
type SingleFeedResult struct {
	FeedID    string        `json:"feed_id"`
	Success   bool          `json:"success"`
	NewItems  int           `json:"new_items"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
}

// String returns a string representation of the single feed result
func (r *SingleFeedResult) String() string {
	duration := r.Duration.String()
	if r.Duration.Seconds() >= 1 {
		duration = fmt.Sprintf("%.0fs", r.Duration.Seconds())
	} else if r.Duration.Milliseconds() >= 1 {
		duration = fmt.Sprintf("%dms", r.Duration.Milliseconds())
	}
	if r.Success {
		return fmt.Sprintf("%s: success (%d items, %s)", r.FeedID, r.NewItems, duration)
	}
	return fmt.Sprintf("%s: failed (%s, %s)", r.FeedID, r.Error, duration)
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
	Update(ctx context.Context, item *model.Item) error
	ListByFeedID(ctx context.Context, feedID string, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	ListStarred(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	ListUnread(ctx context.Context, userID string, opts ListOptions) ([]*ItemWithState, int64, error)
	CountByFeedID(ctx context.Context, feedID string) (int64, error)
}

// UserFeedRepository defines the interface for user-feed relationship data access
type UserFeedRepository interface {
	Create(ctx context.Context, userFeed *model.UserFeed) error
	GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error)
	GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error)
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

// PendingOAuthRepository 定义待确认 OAuth 数据访问接口
type PendingOAuthRepository interface {
	Create(ctx context.Context, pending *model.PendingOAuth) error
	GetByToken(ctx context.Context, token string) (*model.PendingOAuth, error)
	Delete(ctx context.Context, token string) error
}

// StatsRepository defines the interface for user statistics data access
type StatsRepository interface {
	// GetUserStats returns article statistics for a user
	GetUserStats(ctx context.Context, userID string) (*UserStats, error)
}

// StatsService defines the interface for statistics business logic
type StatsService interface {
	// GetUserStats returns article statistics for a user
	GetUserStats(ctx context.Context, userID string) (*UserStats, error)
}

// PaperRepository defines the interface for paper data access
type PaperRepository interface {
	Create(ctx context.Context, paper *model.Paper) error
	GetByID(ctx context.Context, id string) (*model.Paper, error)
	ListByUserID(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error)
	Update(ctx context.Context, paper *model.Paper) error
	Delete(ctx context.Context, id string) error
	ListTags(ctx context.Context, userID string) ([]string, error)
}

// PaperTagRepository defines the interface for paper tag data access
type PaperTagRepository interface {
	SetTags(ctx context.Context, paperID string, tags []string) error
	GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error)
}

// PaperCollectionRepository defines the interface for paper collection data access
type PaperCollectionRepository interface {
	Create(ctx context.Context, collection *model.PaperCollection) error
	GetByID(ctx context.Context, id string) (*model.PaperCollection, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.PaperCollection, error)
	Update(ctx context.Context, collection *model.PaperCollection) error
	Delete(ctx context.Context, id string) error
	AddPaper(ctx context.Context, collectionID, paperID string) error
	RemovePaper(ctx context.Context, collectionID, paperID string) error
	ListPapers(ctx context.Context, collectionID string) ([]*model.Paper, error)
}

// PaperListOptions defines pagination and filtering for papers
type PaperListOptions struct {
	Limit  int
	Offset int
	Query  string // search title, authors, keywords
	Year   string
	Tag    string
	Status string
	Sort   string // "created_at", "title", "published_year"
	Order  string // "asc", "desc"
}
