package model

// Paper statuses
const (
	PaperStatusPending    = "pending"
	PaperStatusProcessing = "processing"
	PaperStatusCompleted  = "completed"
	PaperStatusFailed     = "failed"
)

// Paper represents an uploaded academic paper
type Paper struct {
	Base
	UserID           string `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Title            string `gorm:"type:varchar(500)" json:"title"`
	Authors          string `gorm:"type:text" json:"authors"`
	Abstract         string `gorm:"type:text" json:"abstract"`
	Keywords         string `gorm:"type:text" json:"keywords"`
	PublishedYear    string `gorm:"type:varchar(10)" json:"published_year"`
	DOI              string `gorm:"type:varchar(200)" json:"doi"`
	PDFPath          string `gorm:"type:varchar(500)" json:"-"`
	PDFSize          int64  `json:"pdf_size"`
	MarkdownContent  string `gorm:"type:longtext" json:"markdown_content"`
	CoverImage       string `gorm:"type:varchar(500)" json:"cover_image"`
	OriginalFilename string `gorm:"type:varchar(255)" json:"original_filename"`
	Status           string `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Error            string `gorm:"type:text" json:"error,omitempty"`
	User             *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PaperTag represents a tag for a paper
type PaperTag struct {
	Base
	PaperID string `gorm:"type:varchar(36);not null;index" json:"paper_id"`
	Tag     string `gorm:"type:varchar(100);not null;index" json:"tag"`
	Paper   *Paper `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}

// PaperCollection represents a user-defined paper collection
type PaperCollection struct {
	Base
	UserID      string `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	User        *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PaperCollectionItem represents a paper in a collection
type PaperCollectionItem struct {
	Base
	CollectionID string           `gorm:"type:varchar(36);not null;uniqueIndex:idx_collection_paper" json:"collection_id"`
	PaperID      string           `gorm:"type:varchar(36);not null;uniqueIndex:idx_collection_paper" json:"paper_id"`
	Collection   *PaperCollection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	Paper        *Paper           `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
}
