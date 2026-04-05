package model

// Category type constants
const (
	CategoryTypeFeed  = "feed"
	CategoryTypePaper = "paper"
)

// Category represents a user-defined category for feeds or papers
type Category struct {
	Base
	UserID   string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_cat_name" json:"user_id"`
	Name     string `gorm:"type:varchar(100);not null;uniqueIndex:idx_user_cat_name" json:"name"`
	Type     string `gorm:"type:varchar(20);not null;uniqueIndex:idx_user_cat_name" json:"type"`
	Position int    `gorm:"default:0" json:"position"`
	User     *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
