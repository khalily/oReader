package model

// Category type constants
const (
	CategoryTypeFeed  = "feed"
	CategoryTypePaper = "paper"
)

// Category represents a user-defined category for feeds or papers
type Category struct {
	Base
	UserID   string `gorm:"type:varchar(36);not null;index:idx_user_cat_type" json:"user_id"`
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Type     string `gorm:"type:varchar(20);not null;index:idx_user_cat_type" json:"type"`
	Position int    `gorm:"default:0" json:"position"`
	User     *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
