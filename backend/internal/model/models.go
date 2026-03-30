package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base contains common fields for all models
type Base struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// GenerateID generates a UUID v7 for the model
func (b *Base) GenerateID() error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	b.ID = id.String()
	return nil
}

// User represents a user in the system
type User struct {
	Base
	Email        string `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	PasswordHash string `gorm:"type:varchar(255)" json:"-"` // Nullable for OAuth users
	Nickname     string `gorm:"type:varchar(100)" json:"nickname"`
	AvatarURL    string `gorm:"type:varchar(500)" json:"avatar_url"`
	AuthProvider string `gorm:"type:varchar(50);default:'email'" json:"auth_provider"`
	GitHubID     string `gorm:"type:varchar(100)" json:"github_id,omitempty"`
	GitHubLogin  string `gorm:"type:varchar(100)" json:"github_login,omitempty"`
}

// Feed represents an RSS feed (shared among users)
type Feed struct {
	Base
	FeedURL            string    `gorm:"uniqueIndex;type:varchar(500);not null" json:"feed_url"`
	Title              string    `gorm:"type:varchar(255);not null" json:"title"`
	Description        string    `gorm:"type:text" json:"description"`
	ImageURL           string    `gorm:"type:varchar(500)" json:"image_url"`
	LastFetchedAt      *time.Time `json:"last_fetched_at"`
	LastFetchStatus    string    `gorm:"type:varchar(20)" json:"last_fetch_status"`
	ConsecutiveFailures int      `gorm:"default:0" json:"consecutive_failures"`
}

// UserFeed represents the subscription relationship between user and feed
type UserFeed struct {
	Base
	UserID    string `gorm:"type:varchar(36);not null;index:idx_user_feed" json:"user_id"`
	FeedID    string `gorm:"type:varchar(36);not null;index:idx_user_feed" json:"feed_id"`
	Position  int    `gorm:"default:0" json:"position"`
	User      *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Feed      *Feed  `gorm:"foreignKey:FeedID" json:"feed,omitempty"`
}

// Item represents an RSS feed item (shared among users)
type Item struct {
	Base
	FeedID      string     `gorm:"type:varchar(36);not null;index:idx_feed_guid,unique" json:"feed_id"`
	GUID        string     `gorm:"type:varchar(500);not null;index:idx_feed_guid,unique" json:"guid"`
	Title       string     `gorm:"type:varchar(500);not null" json:"title"`
	Link        string     `gorm:"type:varchar(500);not null" json:"link"`
	Description string     `gorm:"type:text" json:"description"`
	Content     string     `gorm:"type:longtext" json:"content"`
	PubDate     *time.Time `json:"pub_date"`
	Creator     string     `gorm:"type:varchar(255)" json:"creator"`
	Feed        *Feed      `gorm:"foreignKey:FeedID" json:"feed,omitempty"`
}

// UserItemState represents per-user item state (starred, read)
type UserItemState struct {
	Base
	UserID    string     `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_item" json:"user_id"`
	ItemID    string     `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_item" json:"item_id"`
	IsStarred bool       `gorm:"default:false" json:"is_starred"`
	IsRead    bool       `gorm:"default:false" json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Item      *Item      `gorm:"foreignKey:ItemID" json:"item,omitempty"`
}

// RefreshToken represents a refresh token for authentication
type RefreshToken struct {
	Base
	UserID    string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	Revoked   bool       `gorm:"default:false" json:"revoked"`
	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// IsExpired returns true if the token has expired
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid returns true if the token is valid (not expired and not revoked)
func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && !t.IsExpired()
}

// Import job statuses
const (
	ImportJobStatusPending    = "pending"
	ImportJobStatusProcessing = "processing"
	ImportJobStatusCompleted  = "completed"
	ImportJobStatusFailed     = "failed"
)

// ImportJob represents an OPML import job
type ImportJob struct {
	Base
	UserID     string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Status     string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	TotalFeeds int        `gorm:"default:0" json:"total_feeds"`
	Processed  int        `gorm:"default:0" json:"processed"`
	Failed     int        `gorm:"default:0" json:"failed"`
	Error      string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	User       *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// OAuthState represents an OAuth state for CSRF protection
type OAuthState struct {
	Base
	State     string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"state"`
	Provider  string    `gorm:"type:varchar(50);not null" json:"provider"`
	UserID    string    `gorm:"type:varchar(36)" json:"user_id,omitempty"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}

// IsExpired returns true if the OAuth state has expired
func (s *OAuthState) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// PendingOAuth 存储待确认的 OAuth 数据（5分钟过期）
type PendingOAuth struct {
	Base
	Token       string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"token"`
	GitHubID    string    `gorm:"type:varchar(100);not null" json:"github_id"`
	GitHubLogin string    `gorm:"type:varchar(100);not null" json:"github_login"`
	Nickname    string    `gorm:"type:varchar(100)" json:"nickname"`
	AvatarURL   string    `gorm:"type:varchar(500)" json:"avatar_url"`
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
}

// IsExpired 判断是否过期
func (p *PendingOAuth) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
