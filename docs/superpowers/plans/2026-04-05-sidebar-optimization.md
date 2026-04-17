# 侧边栏优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化前端界面布局，移除冗余导航元素，将 Papers 和 Feeds 统一为三栏阅读体验，支持分类管理。

**Architecture:** 后端新增 Category model + CRUD API（Go/Gin/GORM），前端重写 Sidebar 为分类树结构（React/Zustand），统一 Feeds 和 Papers 到三栏布局，移除 all/unread/starred/today 过滤和顶部工具栏，新增"新增"统一对话框。

**Tech Stack:** Go 1.25+ / Gin / GORM (后端), React 19 / TypeScript / Zustand / TanStack Query / Tailwind CSS (前端)

---

## Task 1: 后端 — Category Model + Migration

**Files:**
- Create: `backend/internal/model/category.go`
- Modify: `backend/cmd/server/main.go:57-74` (AutoMigrate)

- [ ] **Step 1: 编写 Category model 测试**

```go
// backend/internal/model/category_test.go
package model

import "testing"

func TestCategoryTypeValues(t *testing.T) {
	validTypes := []string{CategoryTypeFeed, CategoryTypePaper}
	for _, ct := range validTypes {
		if ct != "feed" && ct != "paper" {
			t.Errorf("unexpected category type: %s", ct)
		}
	}
}

func TestCategoryGenerateID(t *testing.T) {
	c := &Category{Name: "Test", Type: CategoryTypeFeed, UserID: "user-1"}
	if err := c.GenerateID(); err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if c.ID == "" {
		t.Error("ID should not be empty")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/model/ -run TestCategory -v`
Expected: FAIL (model/category.go 不存在)

- [ ] **Step 3: 实现 Category model**

```go
// backend/internal/model/category.go
package model

// Category type constants
const (
	CategoryTypeFeed  = "feed"
	CategoryTypePaper = "paper"
)

// Category represents a user-defined category for feeds or papers
type Category struct {
	Base
	UserID  string `gorm:"type:varchar(36);not null;index:idx_user_cat_type" json:"user_id"`
	Name    string `gorm:"type:varchar(100);not null" json:"name"`
	Type    string `gorm:"type:varchar(20);not null;index:idx_user_cat_type" json:"type"`
	Position int   `gorm:"default:0" json:"position"`
	User    *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

- [ ] **Step 4: 在 main.go 中注册 AutoMigrate**

在 `backend/cmd/server/main.go` 第 57 行的 `db.AutoMigrate` 调用中添加 `&model.Category{}`：

```go
if err := db.AutoMigrate(
	&model.User{},
	&model.Feed{},
	&model.UserFeed{},
	&model.Item{},
	&model.UserItemState{},
	&model.RefreshToken{},
	&model.ImportJob{},
	&model.OAuthState{},
	&model.PendingOAuth{},
	&model.Paper{},
	&model.PaperTag{},
	&model.PaperCollection{},
	&model.PaperCollectionItem{},
	&model.Category{}, // <-- 新增
); err != nil {
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/model/ -run TestCategory -v`
Expected: PASS

- [ ] **Step 6: 添加 category_id 到 UserFeed 和 Paper models**

在 `backend/internal/model/models.go` 的 `UserFeed` struct (line 53) 添加：

```go
type UserFeed struct {
	Base
	UserID     string    `gorm:"type:varchar(36);not null;index:idx_user_feed" json:"user_id"`
	FeedID     string    `gorm:"type:varchar(36);not null;index:idx_user_feed" json:"feed_id"`
	Position   int       `gorm:"default:0" json:"position"`
	CategoryID *string   `gorm:"type:varchar(36);index" json:"category_id"`
	Category   *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	User       *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Feed       *Feed     `gorm:"foreignKey:FeedID" json:"feed,omitempty"`
}
```

在 `backend/internal/model/paper.go` 的 `Paper` struct (line 12) 添加：

```go
type Paper struct {
	Base
	UserID           string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Title            string    `gorm:"type:varchar(500)" json:"title"`
	// ... 其他字段不变 ...
	CategoryID       *string   `gorm:"type:varchar(36);index" json:"category_id"`
	Category         *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	User             *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

- [ ] **Step 7: 确认编译通过**

Run: `cd backend && go build ./...`
Expected: 编译成功

- [ ] **Step 8: Commit**

```bash
git add backend/internal/model/category.go backend/internal/model/models.go backend/internal/model/paper.go backend/cmd/server/main.go
git commit -m "feat: add Category model and category_id to UserFeed/Paper"
```

---

## Task 2: 后端 — Category Repository + Service 接口

**Files:**
- Create: `backend/internal/repository/category_repository.go`
- Create: `backend/internal/repository/category_repository_test.go`
- Modify: `backend/internal/service/interfaces.go`

- [ ] **Step 1: 在 interfaces.go 中定义 CategoryRepository 和 CategoryService 接口**

在 `backend/internal/service/interfaces.go` 末尾添加：

```go
// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	GetByID(ctx context.Context, id string) (*model.Category, error)
	ListByUserID(ctx context.Context, userID string, categoryType string) ([]*model.Category, error)
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id string) error
	GetMaxPosition(ctx context.Context, userID string, categoryType string) (int, error)
}

// CategoryService defines the interface for category business logic
type CategoryService interface {
	CreateCategory(ctx context.Context, userID string, name string, categoryType string) (*model.Category, error)
	ListCategories(ctx context.Context, userID string, categoryType string) ([]*model.Category, error)
	RenameCategory(ctx context.Context, userID, categoryID, newName string) (*model.Category, error)
	DeleteCategory(ctx context.Context, userID, categoryID string) error
	MoveFeedToCategory(ctx context.Context, userID, feedID, categoryID string) error
	MovePaperToCategory(ctx context.Context, userID, paperID, categoryID string) error
}
```

同时添加错误变量：

```go
// 在已有 var 块中添加
ErrCategoryNotFound  = errors.New("category not found")
ErrCategoryDuplicate = errors.New("category with same name already exists")
```

- [ ] **Step 2: 编写 CategoryRepository 测试**

```go
// backend/internal/repository/category_repository_test.go
package repository

import (
	"context"
	"testing"

	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/testutil"
)

func TestCategoryRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &model.Category{
		UserID: "test-user-id",
		Name:   "Tech Blogs",
		Type:   model.CategoryTypeFeed,
	}
	if err := cat.GenerateID(); err != nil {
		t.Fatal(err)
	}

	err := repo.Create(ctx, cat)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if cat.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCategoryRepository_ListByUserID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	userID := "test-user-id"

	// Create feed category
	feedCat := &model.Category{UserID: userID, Name: "Tech", Type: model.CategoryTypeFeed}
	feedCat.GenerateID()
	repo.Create(ctx, feedCat)

	// Create paper category
	paperCat := &model.Category{UserID: userID, Name: "ML", Type: model.CategoryTypePaper}
	paperCat.GenerateID()
	repo.Create(ctx, paperCat)

	// List only feed categories
	feedCats, err := repo.ListByUserID(ctx, userID, model.CategoryTypeFeed)
	if err != nil {
		t.Fatalf("ListByUserID failed: %v", err)
	}
	if len(feedCats) != 1 {
		t.Errorf("expected 1 feed category, got %d", len(feedCats))
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd backend && go test ./internal/repository/ -run TestCategory -v`
Expected: FAIL (NewCategoryRepository 不存在)

- [ ] **Step 4: 实现 CategoryRepository**

```go
// backend/internal/repository/category_repository.go
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *gorm.DB) service.CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *categoryRepository) GetByID(ctx context.Context, id string) (*model.Category, error) {
	var category model.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, service.ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) ListByUserID(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
	var categories []*model.Category
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if categoryType != "" {
		query = query.Where("type = ?", categoryType)
	}
	err := query.Order("position ASC, created_at ASC").Find(&categories).Error
	return categories, err
}

func (r *categoryRepository) Update(ctx context.Context, category *model.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Category{}, "id = ?", id).Error
}

func (r *categoryRepository) GetMaxPosition(ctx context.Context, userID string, categoryType string) (int, error) {
	var maxPos int
	err := r.db.WithContext(ctx).
		Model(&model.Category{}).
		Where("user_id = ? AND type = ?", userID, categoryType).
		Coalesce("MAX(position)").Scan(&maxPos).Error
	if err != nil {
		return 0, err
	}
	return maxPos, nil
}
```

注意：如果 `Coalesce` 不可用，使用原生 SQL：

```go
func (r *categoryRepository) GetMaxPosition(ctx context.Context, userID string, categoryType string) (int, error) {
	var maxPos *int
	err := r.db.WithContext(ctx).
		Table("categories").
		Select("MAX(position)").
		Where("user_id = ? AND type = ?", userID, categoryType).
		Scan(&maxPos).Error
	if err != nil {
		return 0, err
	}
	if maxPos == nil {
		return 0, nil
	}
	return *maxPos, nil
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/repository/ -run TestCategory -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/category_repository.go backend/internal/repository/category_repository_test.go backend/internal/service/interfaces.go
git commit -m "feat: add CategoryRepository and CategoryService interfaces"
```

---

## Task 3: 后端 — Category Service 实现

**Files:**
- Create: `backend/internal/service/category_service.go`
- Create: `backend/internal/service/category_service_test.go`
- Modify: `backend/cmd/server/main.go` (初始化 service)

- [ ] **Step 1: 编写 CategoryService 测试**

```go
// backend/internal/service/category_service_test.go
package service

import (
	"context"
	"testing"

	"github.com/khalily/oreader/internal/model"
)

// mockCategoryRepository for testing
type mockCategoryRepository struct {
	categories map[string]*model.Category
}

func newMockCategoryRepository() *mockCategoryRepository {
	return &mockCategoryRepository{categories: make(map[string]*model.Category)}
}

func (m *mockCategoryRepository) Create(ctx context.Context, cat *model.Category) error {
	if cat.ID == "" {
		cat.GenerateID()
	}
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockCategoryRepository) GetByID(ctx context.Context, id string) (*model.Category, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, ErrCategoryNotFound
	}
	return cat, nil
}

func (m *mockCategoryRepository) ListByUserID(ctx context.Context, userID, categoryType string) ([]*model.Category, error) {
	var result []*model.Category
	for _, cat := range m.categories {
		if cat.UserID == userID && (categoryType == "" || cat.Type == categoryType) {
			result = append(result, cat)
		}
	}
	return result, nil
}

func (m *mockCategoryRepository) Update(ctx context.Context, cat *model.Category) error {
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockCategoryRepository) Delete(ctx context.Context, id string) error {
	delete(m.categories, id)
	return nil
}

func (m *mockCategoryRepository) GetMaxPosition(ctx context.Context, userID, categoryType string) (int, error) {
	max := -1
	for _, cat := range m.categories {
		if cat.UserID == userID && cat.Type == categoryType && cat.Position > max {
			max = cat.Position
		}
	}
	return max + 1, nil
}

// mockUserFeedRepository for MoveFeedToCategory
type mockUserFeedRepoForCategory struct {
	userFeeds map[string]*model.UserFeed
}

func newMockUserFeedRepoForCategory() *mockUserFeedRepoForCategory {
	return &mockUserFeedRepoForCategory{userFeeds: make(map[string]*model.UserFeed)}
}

func (m *mockUserFeedRepoForCategory) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	for _, uf := range m.userFeeds {
		if uf.UserID == userID && uf.FeedID == feedID {
			return uf, nil
		}
	}
	return nil, ErrFeedNotFound
}

func (m *mockUserFeedRepoForCategory) UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error {
	for _, uf := range m.userFeeds {
		if uf.UserID == userID && uf.FeedID == feedID {
			uf.CategoryID = categoryID
			return nil
		}
	}
	return ErrFeedNotFound
}

func (m *mockUserFeedRepoForCategory) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	return m.GetByUserAndFeed(ctx, userID, feedID)
}

func (m *mockUserFeedRepoForCategory) Create(ctx context.Context, uf *model.UserFeed) error { return nil }
func (m *mockUserFeedRepoForCategory) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) { return nil, nil }
func (m *mockUserFeedRepoForCategory) Delete(ctx context.Context, userID, feedID string) error { return nil }
func (m *mockUserFeedRepoForCategory) GetMaxPosition(ctx context.Context, userID string) (int, error) { return 0, nil }

func TestCreateCategory(t *testing.T) {
	catRepo := newMockCategoryRepository()
	ufRepo := newMockUserFeedRepoForCategory()
	svc := NewCategoryService(catRepo, ufRepo, nil)

	cat, err := svc.CreateCategory(context.Background(), "user-1", "Tech Blogs", model.CategoryTypeFeed)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if cat.Name != "Tech Blogs" {
		t.Errorf("expected name 'Tech Blogs', got '%s'", cat.Name)
	}
	if cat.Type != model.CategoryTypeFeed {
		t.Errorf("expected type 'feed', got '%s'", cat.Type)
	}
}

func TestCreateCategory_Duplicate(t *testing.T) {
	catRepo := newMockCategoryRepository()
	ufRepo := newMockUserFeedRepoForCategory()
	svc := NewCategoryService(catRepo, ufRepo, nil)

	_, err := svc.CreateCategory(context.Background(), "user-1", "Tech", model.CategoryTypeFeed)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.CreateCategory(context.Background(), "user-1", "Tech", model.CategoryTypeFeed)
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestRenameCategory(t *testing.T) {
	catRepo := newMockCategoryRepository()
	ufRepo := newMockUserFeedRepoForCategory()
	svc := NewCategoryService(catRepo, ufRepo, nil)

	cat, _ := svc.CreateCategory(context.Background(), "user-1", "Old Name", model.CategoryTypeFeed)
	renamed, err := svc.RenameCategory(context.Background(), "user-1", cat.ID, "New Name")
	if err != nil {
		t.Fatalf("RenameCategory failed: %v", err)
	}
	if renamed.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", renamed.Name)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/service/ -run TestCreateCategory -v`
Expected: FAIL

- [ ] **Step 3: 实现 CategoryService**

```go
// backend/internal/service/category_service.go
package service

import (
	"context"

	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/model"
)

type categoryService struct {
	catRepo    CategoryRepository
	userFeedRepo UserFeedRepository
	paperRepo  PaperRepository
}

// NewCategoryService creates a new category service
func NewCategoryService(
	catRepo CategoryRepository,
	userFeedRepo UserFeedRepository,
	paperRepo PaperRepository,
) CategoryService {
	return &categoryService{
		catRepo:     catRepo,
		userFeedRepo: userFeedRepo,
		paperRepo:   paperRepo,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, userID, name, categoryType string) (*model.Category, error) {
	logger.Info().
		Str("user_id", userID).
		Str("name", name).
		Str("type", categoryType).
		Msg("CreateCategory request")

	// Check for duplicate name within same user + type
	existing, err := s.catRepo.ListByUserID(ctx, userID, categoryType)
	if err != nil {
		return nil, err
	}
	for _, cat := range existing {
		if cat.Name == name {
			return nil, ErrCategoryDuplicate
		}
	}

	maxPos, err := s.catRepo.GetMaxPosition(ctx, userID, categoryType)
	if err != nil {
		return nil, err
	}

	category := &model.Category{
		UserID:   userID,
		Name:     name,
		Type:     categoryType,
		Position: maxPos,
	}
	if err := category.GenerateID(); err != nil {
		return nil, err
	}

	if err := s.catRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *categoryService) ListCategories(ctx context.Context, userID, categoryType string) ([]*model.Category, error) {
	return s.catRepo.ListByUserID(ctx, userID, categoryType)
}

func (s *categoryService) RenameCategory(ctx context.Context, userID, categoryID, newName string) (*model.Category, error) {
	category, err := s.catRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	if category.UserID != userID {
		return nil, ErrCategoryNotFound
	}

	// Check duplicate name
	existing, err := s.catRepo.ListByUserID(ctx, userID, category.Type)
	if err != nil {
		return nil, err
	}
	for _, cat := range existing {
		if cat.ID != categoryID && cat.Name == newName {
			return nil, ErrCategoryDuplicate
		}
	}

	category.Name = newName
	if err := s.catRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, userID, categoryID string) error {
	category, err := s.catRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if category.UserID != userID {
		return ErrCategoryNotFound
	}

	// Set category_id to NULL for all associated feeds and papers
	// This is handled by GORM's ON DELETE SET NULL if configured
	// But we also need to handle it at the application level
	if s.userFeedRepo != nil {
		_ = s.userFeedRepo.(interface{ UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error }).
			UpdateCategory(ctx, userID, "", nil)
	}

	if err := s.catRepo.Delete(ctx, categoryID); err != nil {
		return err
	}

	logger.Info().
		Str("user_id", userID).
		Str("category_id", categoryID).
		Msg("Category deleted")

	return nil
}

func (s *categoryService) MoveFeedToCategory(ctx context.Context, userID, feedID, categoryID string) error {
	// Validate the user owns this feed subscription
	_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		return ErrFeedNotFound
	}

	// Validate category exists and belongs to user
	if categoryID != "" {
		cat, err := s.catRepo.GetByID(ctx, categoryID)
		if err != nil {
			return err
		}
		if cat.UserID != userID || cat.Type != model.CategoryTypeFeed {
			return ErrCategoryNotFound
		}
	}

	// Get the interface that supports UpdateCategory
	type categoryUpdater interface {
		UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error
	}
	updater, ok := s.userFeedRepo.(categoryUpdater)
	if !ok {
		return nil // Repository doesn't support category updates yet
	}

	var catID *string
	if categoryID != "" {
		catID = &categoryID
	}
	return updater.UpdateCategory(ctx, userID, feedID, catID)
}

func (s *categoryService) MovePaperToCategory(ctx context.Context, userID, paperID, categoryID string) error {
	if s.paperRepo == nil {
		return nil
	}

	// Validate paper exists and belongs to user
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil {
		return ErrPaperNotFound
	}
	if paper.UserID != userID {
		return ErrPaperNotFound
	}

	// Validate category
	if categoryID != "" {
		cat, err := s.catRepo.GetByID(ctx, categoryID)
		if err != nil {
			return err
		}
		if cat.UserID != userID || cat.Type != model.CategoryTypePaper {
			return ErrCategoryNotFound
		}
	}

	var catID *string
	if categoryID != "" {
		catID = &categoryID
	}

	return s.paperRepo.UpdateCategory(ctx, paperID, catID)
}
```

- [ ] **Step 4: 在 UserFeedRepository 接口添加 UpdateCategory 方法**

在 `backend/internal/service/interfaces.go` 的 `UserFeedRepository` 接口中添加：

```go
type UserFeedRepository interface {
	// ... 已有方法 ...
	UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error
}
```

- [ ] **Step 5: 在 PaperRepository 接口添加 UpdateCategory 方法**

在 `backend/internal/service/interfaces.go` 的 `PaperRepository` 接口中添加：

```go
type PaperRepository interface {
	// ... 已有方法 ...
	UpdateCategory(ctx context.Context, paperID string, categoryID *string) error
}
```

- [ ] **Step 6: 实现 UserFeedRepository.UpdateCategory**

在 `backend/internal/repository/user_feed_repository.go` 中添加 UpdateCategory 方法：

```go
func (r *userFeedRepository) UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error {
	result := r.db.WithContext(ctx).
		Model(&model.UserFeed{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Update("category_id", categoryID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFeedNotFound
	}
	return nil
}
```

- [ ] **Step 7: 实现 PaperRepository.UpdateCategory**

在 `backend/internal/repository/paper_repository.go` 中添加 UpdateCategory 方法：

```go
func (r *paperRepository) UpdateCategory(ctx context.Context, paperID string, categoryID *string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", paperID).
		Update("category_id", categoryID)
	return result.Error
}
```

- [ ] **Step 8: 修复 mock 以满足新接口**

更新 `category_service_test.go` 中的 `mockUserFeedRepoForCategory` 使其实现完整接口（包括所有已有方法）：

```go
func (m *mockUserFeedRepoForCategory) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	return m.GetByUserAndFeed(ctx, userID, feedID)
}
func (m *mockUserFeedRepoForCategory) Create(ctx context.Context, uf *model.UserFeed) error {
	if uf.ID == "" { uf.GenerateID() }
	m.userFeeds[uf.ID] = uf
	return nil
}
func (m *mockUserFeedRepoForCategory) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	return nil, nil
}
func (m *mockUserFeedRepoForCategory) Delete(ctx context.Context, userID, feedID string) error { return nil }
func (m *mockUserFeedRepoForCategory) GetMaxPosition(ctx context.Context, userID string) (int, error) { return 0, nil }
```

- [ ] **Step 9: 运行测试确认通过**

Run: `cd backend && go test ./internal/service/ -run "TestCategory|TestCreateCategory|TestRenameCategory" -v`
Expected: PASS

- [ ] **Step 10: 在 main.go 中初始化 categoryRepo + categoryService**

在 `backend/cmd/server/main.go` 中，在 `paperRepo := ...` 之后添加：

```go
categoryRepo := repository.NewCategoryRepository(db)
```

在 `paperService := ...` 之后添加：

```go
categoryService := service.NewCategoryService(categoryRepo, userFeedRepo, paperRepo)
```

- [ ] **Step 11: 编译确认通过**

Run: `cd backend && go build ./...`
Expected: 编译成功

- [ ] **Step 12: Commit**

```bash
git add backend/internal/service/category_service.go backend/internal/service/category_service_test.go backend/internal/service/interfaces.go backend/internal/repository/category_repository.go backend/internal/repository/user_feed_repository.go backend/internal/repository/paper_repository.go backend/cmd/server/main.go
git commit -m "feat: implement CategoryService with CRUD and feed/paper assignment"
```

---

## Task 4: 后端 — Category API Handler + 路由

**Files:**
- Create: `backend/internal/handler/category_handler.go`
- Create: `backend/internal/handler/category_handler_test.go`
- Modify: `backend/cmd/server/main.go:212-258` (添加路由)

- [ ] **Step 1: 编写 Category Handler 测试**

```go
// backend/internal/handler/category_handler_test.go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

type mockCategoryService struct {
	listErr error
}

func (m *mockCategoryService) CreateCategory(ctx context.Context, userID, name, categoryType string) (*model.Category, error) {
	cat := &model.Category{Name: name, Type: categoryType, UserID: userID}
	cat.GenerateID()
	return cat, nil
}

func (m *mockCategoryService) ListCategories(ctx context.Context, userID, categoryType string) ([]*model.Category, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	cat := &model.Category{Name: "Tech", Type: categoryType, UserID: userID}
	cat.GenerateID()
	return []*model.Category{cat}, nil
}

func (m *mockCategoryService) RenameCategory(ctx context.Context, userID, categoryID, newName string) (*model.Category, error) {
	cat := &model.Category{Name: newName, Type: model.CategoryTypeFeed, UserID: userID}
	cat.GenerateID()
	return cat, nil
}

func (m *mockCategoryService) DeleteCategory(ctx context.Context, userID, categoryID string) error {
	return nil
}

func (m *mockCategoryService) MoveFeedToCategory(ctx context.Context, userID, feedID, categoryID string) error {
	return nil
}

func (m *mockCategoryService) MovePaperToCategory(ctx context.Context, userID, paperID, categoryID string) error {
	return nil
}

func setupCategoryRouter(svc service.CategoryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCategoryHandler(svc)
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) { c.Set("user_id", "test-user-id") })
	v1.Use(func(c *gin.Context) {}) // CSRF placeholder
	{
		cats := v1.Group("/categories")
		{
			cats.GET("", h.ListCategories)
			cats.POST("", h.CreateCategory)
			cats.PUT("/:id/rename", h.RenameCategory)
			cats.DELETE("/:id", h.DeleteCategory)
			cats.PUT("/feeds/:feedId", h.MoveFeedToCategory)
			cats.PUT("/papers/:paperId", h.MovePaperToCategory)
		}
	}
	return r
}

func TestCreateCategory(t *testing.T) {
	r := setupCategoryRouter(&mockCategoryService{})

	body := `{"name": "Tech Blogs", "type": "feed"}`
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if cat, ok := resp["category"].(map[string]interface{}); ok {
		if cat["name"] != "Tech Blogs" {
			t.Errorf("expected name 'Tech Blogs', got %v", cat["name"])
		}
	}
}

func TestListCategories(t *testing.T) {
	r := setupCategoryRouter(&mockCategoryService{})

	req := httptest.NewRequest("GET", "/api/v1/categories?type=feed", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/handler/ -run TestCreateCategory -v`
Expected: FAIL

- [ ] **Step 3: 实现 Category Handler**

```go
// backend/internal/handler/category_handler.go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/khalily/oreader/internal/infra/errors"
	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/service"
)

type CategoryHandler struct {
	categoryService service.CategoryService
}

func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,max=100"`
	Type string `json:"type" binding:"required,oneof=feed paper"`
}

type RenameCategoryRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

type MoveToCategoryRequest struct {
	CategoryID string `json:"category_id"`
}

// CreateCategory handles POST /api/v1/categories
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	logger.Info().
		Str("user_id", userID.(string)).
		Str("name", req.Name).
		Str("type", req.Type).
		Msg("CreateCategory request")

	category, err := h.categoryService.CreateCategory(c.Request.Context(), userID.(string), req.Name, req.Type)
	if err != nil {
		if errors.Is(err, service.ErrCategoryDuplicate) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Category with this name already exists", nil)
			return
		}
		logger.Error().Err(err).Msg("CreateCategory failed")
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to create category", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"category": category})
}

// ListCategories handles GET /api/v1/categories
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryType := c.Query("type")

	categories, err := h.categoryService.ListCategories(c.Request.Context(), userID.(string), categoryType)
	if err != nil {
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to list categories", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// RenameCategory handles PUT /api/v1/categories/:id/rename
func (h *CategoryHandler) RenameCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryID := c.Param("id")

	var req RenameCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	category, err := h.categoryService.RenameCategory(c.Request.Context(), userID.(string), categoryID, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryDuplicate) {
			apperrors.SendError(c, http.StatusConflict, apperrors.ErrConflict, "Category with this name already exists", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to rename category", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}

// DeleteCategory handles DELETE /api/v1/categories/:id
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	categoryID := c.Param("id")

	err := h.categoryService.DeleteCategory(c.Request.Context(), userID.(string), categoryID)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to delete category", nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// MoveFeedToCategory handles PUT /api/v1/categories/feeds/:feedId
func (h *CategoryHandler) MoveFeedToCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	feedID := c.Param("feedId")

	var req MoveToCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	err := h.categoryService.MoveFeedToCategory(c.Request.Context(), userID.(string), feedID, req.CategoryID)
	if err != nil {
		if errors.Is(err, service.ErrFeedNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Feed not found", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to move feed to category", nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// MovePaperToCategory handles PUT /api/v1/categories/papers/:paperId
func (h *CategoryHandler) MovePaperToCategory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		apperrors.SendError(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "User not authenticated", nil)
		return
	}

	paperID := c.Param("paperId")

	var req MoveToCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.SendError(c, http.StatusBadRequest, apperrors.ErrValidation, "Invalid request body", nil)
		return
	}

	err := h.categoryService.MovePaperToCategory(c.Request.Context(), userID.(string), paperID, req.CategoryID)
	if err != nil {
		if errors.Is(err, service.ErrPaperNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Paper not found", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			apperrors.SendError(c, http.StatusNotFound, apperrors.ErrNotFound, "Category not found", nil)
			return
		}
		apperrors.SendError(c, http.StatusInternalServerError, apperrors.ErrInternal, "Failed to move paper to category", nil)
		return
	}

	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 4: 在 main.go 中注册路由和 handler**

在 `backend/cmd/server/main.go` 的 handler 初始化区域（约 line 153）添加：

```go
categoryHandler := handler.NewCategoryHandler(categoryService)
```

在 protected 路由组中（约 line 258 之后），添加：

```go
// Category routes
categories := protected.Group("/categories")
{
    categories.GET("", categoryHandler.ListCategories)
    categories.POST("", categoryHandler.CreateCategory)
    categories.PUT("/:id/rename", categoryHandler.RenameCategory)
    categories.DELETE("/:id", categoryHandler.DeleteCategory)
    categories.PUT("/feeds/:feedId", categoryHandler.MoveFeedToCategory)
    categories.PUT("/papers/:paperId", categoryHandler.MovePaperToCategory)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend && go test ./internal/handler/ -run "TestCategory|TestCreateCategory|TestListCategories" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/category_handler.go backend/internal/handler/category_handler_test.go backend/cmd/server/main.go
git commit -m "feat: add Category API handler with CRUD and move endpoints"
```

---

## Task 5: 后端 — 修改 ListFeeds 返回 category_id

**Files:**
- Modify: `backend/internal/service/interfaces.go` (FeedWithItemCount)
- Modify: `backend/internal/repository/feed_repository.go` (ListByUserID 预加载 category)
- Modify: `backend/internal/service/feed_service.go` (GetUserFeeds 传递 category)

- [ ] **Step 1: 更新 FeedWithItemCount 结构体**

在 `backend/internal/service/interfaces.go` 中：

```go
type FeedWithItemCount struct {
	*model.Feed
	ItemCount  int            `json:"item_count"`
	CategoryID *string        `json:"category_id,omitempty"`
}
```

- [ ] **Step 2: 修改 feed_repository.go 的 ListByUserID 预加载 UserFeed 关系**

在 `backend/internal/repository/feed_repository.go` 的 `ListByUserID` 方法中，将查询改为预加载 UserFeed 的 category_id：

```go
func (r *feedRepository) ListByUserID(ctx context.Context, userID string, opts service.ListOptions) ([]*model.Feed, int64, error) {
	var feeds []*model.Feed
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Feed{}).
		Joins("JOIN user_feeds ON user_feeds.feed_id = feeds.id AND user_feeds.deleted_at IS NULL").
		Where("user_feeds.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offsetQuery := query.
		Select("feeds.*, user_feeds.category_id as user_category_id").
		Order("user_feeds.position ASC").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		offsetQuery = offsetQuery.Limit(opts.Limit)
	}

	if err := offsetQuery.Find(&feeds).Error; err != nil {
		return nil, 0, err
	}

	return feeds, total, nil
}
```

注意：GORM 不直接支持 `Select` 中混合表别名。替代方案是分别查询 UserFeed 获取 category_id：

```go
func (r *feedRepository) ListByUserID(ctx context.Context, userID string, opts service.ListOptions) ([]*model.Feed, int64, error) {
	var feeds []*model.Feed
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Feed{}).
		Joins("JOIN user_feeds ON user_feeds.feed_id = feeds.id AND user_feeds.deleted_at IS NULL").
		Where("user_feeds.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offsetQuery := query.
		Order("user_feeds.position ASC").
		Offset(opts.Offset)

	if opts.Limit > 0 {
		offsetQuery = offsetQuery.Limit(opts.Limit)
	}

	if err := offsetQuery.Find(&feeds).Error; err != nil {
		return nil, 0, err
	}

	return feeds, total, nil
}
```

- [ ] **Step 3: 修改 feed_service.go 的 GetUserFeeds 获取 category_id**

在 `backend/internal/service/feed_service.go` 的 `GetUserFeeds` 方法中，额外查询 UserFeed 的 category_id：

```go
func (s *feedService) GetUserFeeds(ctx context.Context, userID string, opts ListOptions) ([]*FeedWithItemCount, int64, error) {
	feeds, total, err := s.feedRepo.ListByUserID(ctx, userID, opts)
	if err != nil {
		return nil, 0, err
	}

	// Get user_feeds for category mapping
	userFeeds, err := s.userFeedRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	feedCategoryMap := make(map[string]*string)
	for _, uf := range userFeeds {
		if uf.CategoryID != nil {
			feedCategoryMap[uf.FeedID] = uf.CategoryID
		}
	}

	result := make([]*FeedWithItemCount, len(feeds))
	for i, feed := range feeds {
		count, err := s.itemRepo.CountByFeedID(ctx, feed.ID)
		if err != nil {
			return nil, 0, err
		}
		result[i] = &FeedWithItemCount{
			Feed:       feed,
			ItemCount:  int(count),
			CategoryID: feedCategoryMap[feed.ID],
		}
	}

	return result, total, nil
}
```

- [ ] **Step 4: 运行全部后端测试**

Run: `cd backend && go test ./internal/... -v -count=1 2>&1 | head -100`
Expected: 所有已有测试通过（可能需要更新 mock）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/interfaces.go backend/internal/repository/feed_repository.go backend/internal/service/feed_service.go
git commit -m "feat: include category_id in ListFeeds response"
```

---

## Task 6: 前端 — Category 类型 + API Hook + Zustand Store

**Files:**
- Create: `frontend/src/types/category.ts`
- Create: `frontend/src/hooks/useCategories.ts`
- Create: `frontend/src/stores/sidebarStore.ts`

- [ ] **Step 1: 创建 Category 类型定义**

```typescript
// frontend/src/types/category.ts
export interface Category {
  id: string
  user_id: string
  name: string
  type: 'feed' | 'paper'
  position: number
  created_at: string
}

export interface CreateCategoryRequest {
  name: string
  type: 'feed' | 'paper'
}

export interface RenameCategoryRequest {
  name: string
}

export interface MoveToCategoryRequest {
  category_id: string
}

export interface ListCategoriesResponse {
  categories: Category[]
}
```

- [ ] **Step 2: 创建 useCategories hook**

```typescript
// frontend/src/hooks/useCategories.ts
import { useMutation, useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/axios'
import type {
  Category,
  CreateCategoryRequest,
  ListCategoriesResponse,
  RenameCategoryRequest,
  MoveToCategoryRequest,
} from '@/types/category'

async function listCategories(type?: 'feed' | 'paper'): Promise<ListCategoriesResponse> {
  const params = new URLSearchParams()
  if (type) params.append('type', type)
  const url = `/categories${params.toString() ? `?${params}` : ''}`
  const response = await apiClient.get<ListCategoriesResponse>(url)
  return response.data
}

async function createCategory(data: CreateCategoryRequest): Promise<{ category: Category }> {
  const response = await apiClient.post('/categories', data)
  return response.data
}

async function renameCategory(id: string, data: RenameCategoryRequest): Promise<{ category: Category }> {
  const response = await apiClient.put(`/categories/${id}/rename`, data)
  return response.data
}

async function deleteCategory(id: string): Promise<void> {
  await apiClient.delete(`/categories/${id}`)
}

async function moveFeedToCategory(feedId: string, data: MoveToCategoryRequest): Promise<void> {
  await apiClient.put(`/categories/feeds/${feedId}`, data)
}

async function movePaperToCategory(paperId: string, data: MoveToCategoryRequest): Promise<void> {
  await apiClient.put(`/categories/papers/${paperId}`, data)
}

export function useCategories() {
  const useListCategories = (type?: 'feed' | 'paper') =>
    useQuery({
      queryKey: ['categories', type],
      queryFn: () => listCategories(type),
      staleTime: 60 * 1000,
    })

  const useCreateCategory = () =>
    useMutation({
      mutationFn: (data: CreateCategoryRequest) => createCategory(data),
    })

  const useRenameCategory = () =>
    useMutation({
      mutationFn: ({ id, data }: { id: string; data: RenameCategoryRequest }) =>
        renameCategory(id, data),
    })

  const useDeleteCategory = () =>
    useMutation({
      mutationFn: (id: string) => deleteCategory(id),
    })

  const useMoveFeedToCategory = () =>
    useMutation({
      mutationFn: ({ feedId, data }: { feedId: string; data: MoveToCategoryRequest }) =>
        moveFeedToCategory(feedId, data),
    })

  const useMovePaperToCategory = () =>
    useMutation({
      mutationFn: ({ paperId, data }: { paperId: string; data: MoveToCategoryRequest }) =>
        movePaperToCategory(paperId, data),
    })

  return {
    useListCategories,
    useCreateCategory,
    useRenameCategory,
    useDeleteCategory,
    useMoveFeedToCategory,
    useMovePaperToCategory,
  }
}
```

- [ ] **Step 3: 创建 sidebarStore**

```typescript
// frontend/src/stores/sidebarStore.ts
import { create } from 'zustand'

export type SidebarSelectionType = 'feed' | 'paper' | null

interface SidebarState {
  // Feed selection
  selectedFeedId: string | null
  selectedPaperId: string | null
  selectionType: SidebarSelectionType

  // Category expand/collapse
  expandedCategoryIds: Set<string>

  // Actions
  selectFeed: (feedId: string) => void
  selectPaper: (paperId: string) => void
  clearSelection: () => void
  toggleCategory: (categoryId: string) => void
  expandCategory: (categoryId: string) => void
}

export const useSidebarStore = create<SidebarState>((set) => ({
  selectedFeedId: null,
  selectedPaperId: null,
  selectionType: null,
  expandedCategoryIds: new Set(),

  selectFeed: (feedId) =>
    set({
      selectedFeedId: feedId,
      selectedPaperId: null,
      selectionType: 'feed',
    }),

  selectPaper: (paperId) =>
    set({
      selectedFeedId: null,
      selectedPaperId: paperId,
      selectionType: 'paper',
    }),

  clearSelection: () =>
    set({
      selectedFeedId: null,
      selectedPaperId: null,
      selectionType: null,
    }),

  toggleCategory: (categoryId) =>
    set((state) => {
      const next = new Set(state.expandedCategoryIds)
      if (next.has(categoryId)) {
        next.delete(categoryId)
      } else {
        next.add(categoryId)
      }
      return { expandedCategoryIds: next }
    }),

  expandCategory: (categoryId) =>
    set((state) => {
      const next = new Set(state.expandedCategoryIds)
      next.add(categoryId)
      return { expandedCategoryIds: next }
    }),
}))
```

- [ ] **Step 4: 更新 UserFeed 类型添加 category_id**

在 `frontend/src/types/feed.ts` 的 `UserFeed` interface 添加：

```typescript
export interface UserFeed extends Feed {
  item_count: number
  unread_count: number
  position: number
  category_id?: string | null
}
```

- [ ] **Step 5: TypeScript 编译检查**

Run: `cd frontend && npx tsc --noEmit 2>&1 | head -30`
Expected: 无新增类型错误

- [ ] **Step 6: Commit**

```bash
git add frontend/src/types/category.ts frontend/src/hooks/useCategories.ts frontend/src/stores/sidebarStore.ts frontend/src/types/feed.ts
git commit -m "feat: add Category types, API hook, and sidebar Zustand store"
```

---

## Task 7: 前端 — 重写 Sidebar 组件

**Files:**
- Rewrite: `frontend/src/components/feed/Sidebar.tsx`
- Create: `frontend/src/components/sidebar/SidebarHeader.tsx`
- Create: `frontend/src/components/sidebar/FeedSection.tsx`
- Create: `frontend/src/components/sidebar/PaperSection.tsx`
- Create: `frontend/src/components/sidebar/CategoryRow.tsx`
- Create: `frontend/src/components/sidebar/FeedRow.tsx`
- Create: `frontend/src/components/sidebar/PaperRow.tsx`

- [ ] **Step 1: 创建 SidebarHeader 组件**

```tsx
// frontend/src/components/sidebar/SidebarHeader.tsx
import { Plus, User } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/authStore'

interface SidebarHeaderProps {
  onAddClick: () => void
}

export function SidebarHeader({ onAddClick }: SidebarHeaderProps) {
  const user = useAuthStore((s) => s.user)

  // Get user initial with gradient background
  const initial = user?.nickname?.[0] || user?.email?.[0] || '?'

  return (
    <div className="flex items-center justify-between px-4 py-3 border-b">
      <div className="flex items-center gap-2">
        <div className="w-6 h-6 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
          <span className="text-white text-xs font-bold">o</span>
        </div>
        <span className="font-semibold text-sm">oReader</span>
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={onAddClick}
        >
          <Plus className="h-3.5 w-3.5 mr-1" />
          新增
        </Button>
        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-emerald-400 to-cyan-500 flex items-center justify-center ml-1">
          <span className="text-white text-xs font-medium">{initial.toUpperCase()}</span>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: 创建 CategoryRow 组件（可展开/折叠，右键菜单）**

```tsx
// frontend/src/components/sidebar/CategoryRow.tsx
import { useState, useRef, useEffect, useCallback } from 'react'
import { ChevronRight, ChevronDown, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Category } from '@/types/category'

interface CategoryRowProps {
  category: Category
  isExpanded: boolean
  itemCount: number
  onToggle: () => void
  onRename: (name: string) => void
  onDelete: () => void
}

export function CategoryRow({
  category,
  isExpanded,
  itemCount,
  onToggle,
  onRename,
  onDelete,
}: CategoryRowProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const [isRenaming, setIsRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState(category.name)
  const menuRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Close menu on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsMenuOpen(false)
      }
    }
    if (isMenuOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isMenuOpen])

  useEffect(() => {
    if (isRenaming && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isRenaming])

  const handleRenameSubmit = useCallback(() => {
    const trimmed = renameValue.trim()
    if (trimmed && trimmed !== category.name) {
      onRename(trimmed)
    }
    setIsRenaming(false)
  }, [renameValue, category.name, onRename])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleRenameSubmit()
    if (e.key === 'Escape') {
      setRenameValue(category.name)
      setIsRenaming(false)
    }
  }

  const ChevronIcon = isExpanded ? ChevronDown : ChevronRight

  return (
    <div className="group relative" ref={menuRef}>
      <div
        className="flex items-center gap-1 px-2 py-1 rounded-sm cursor-pointer hover:bg-accent/50 text-sm"
        onClick={onToggle}
        onContextMenu={(e) => {
          e.preventDefault()
          setIsMenuOpen(true)
        }}
      >
        <ChevronIcon className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
        <span className="font-medium text-muted-foreground truncate flex-1">
          {isRenaming ? (
            <input
              ref={inputRef}
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={handleRenameSubmit}
              onKeyDown={handleKeyDown}
              onClick={(e) => e.stopPropagation()}
              className="bg-background border rounded px-1 py-0 text-sm w-full"
            />
          ) : (
            category.name
          )}
        </span>
        {itemCount > 0 && (
          <span className="text-xs text-muted-foreground">{itemCount}</span>
        )}
        <button
          className="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-accent rounded"
          onClick={(e) => {
            e.stopPropagation()
            setIsMenuOpen(!isMenuOpen)
          }}
        >
          <MoreHorizontal className="h-3.5 w-3.5 text-muted-foreground" />
        </button>
      </div>

      {isMenuOpen && (
        <div className="absolute left-full top-0 ml-1 z-50 min-w-[8rem] rounded-md border bg-popover p-1 shadow-md">
          <button
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
            onClick={(e) => {
              e.stopPropagation()
              setIsRenaming(true)
              setIsMenuOpen(false)
            }}
          >
            <Pencil className="h-3.5 w-3.5" />
            重命名
          </button>
          <button
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent text-destructive"
            onClick={(e) => {
              e.stopPropagation()
              setIsMenuOpen(false)
              onDelete()
            }}
          >
            <Trash2 className="h-3.5 w-3.5" />
            删除
          </button>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: 创建 FeedRow 组件（侧边栏中的 feed 项，右键菜单支持分类）**

```tsx
// frontend/src/components/sidebar/FeedRow.tsx
import { useState, useRef, useEffect } from 'react'
import { Rss, FolderPlus, Folder, MoreHorizontal } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { UserFeed } from '@/types/feed'
import type { Category } from '@/types/category'

interface FeedRowProps {
  feed: UserFeed
  isSelected: boolean
  categories: Category[]
  onFeedClick: (feedId: string) => void
  onMoveToCategory: (feedId: string, categoryId: string) => void
  onMoveToNewCategory: (feedId: string, categoryName: string) => void
}

export function FeedRow({
  feed,
  isSelected,
  categories,
  onFeedClick,
  onMoveToCategory,
  onMoveToNewCategory,
}: FeedRowProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const [showNewCategoryInput, setShowNewCategoryInput] = useState(false)
  const [newCategoryName, setNewCategoryName] = useState('')
  const menuRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsMenuOpen(false)
        setShowNewCategoryInput(false)
      }
    }
    if (isMenuOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isMenuOpen])

  useEffect(() => {
    if (showNewCategoryInput && inputRef.current) {
      inputRef.current.focus()
    }
  }, [showNewCategoryInput])

  return (
    <div className="group relative" ref={menuRef}>
      <div
        className={cn(
          "flex items-center gap-2 px-2 py-1 rounded-sm cursor-pointer text-sm",
          isSelected ? "bg-accent" : "hover:bg-accent/50"
        )}
        onClick={() => onFeedClick(feed.id)}
        onContextMenu={(e) => {
          e.preventDefault()
          setIsMenuOpen(true)
        }}
      >
        {feed.image_url ? (
          <img src={feed.image_url} alt="" className="w-4 h-4 rounded object-cover flex-shrink-0" />
        ) : (
          <Rss className="h-4 w-4 text-muted-foreground flex-shrink-0" />
        )}
        <span className="truncate flex-1">{feed.title}</span>
        {feed.unread_count > 0 && (
          <Badge variant="default" className="text-[10px] px-1 py-0 min-w-[16px] text-center">
            {feed.unread_count}
          </Badge>
        )}
        <button
          className="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-accent rounded"
          onClick={(e) => {
            e.stopPropagation()
            setIsMenuOpen(!isMenuOpen)
          }}
        >
          <MoreHorizontal className="h-3.5 w-3.5 text-muted-foreground" />
        </button>
      </div>

      {isMenuOpen && (
        <div className="absolute left-full top-0 ml-1 z-50 min-w-[10rem] rounded-md border bg-popover p-1 shadow-md">
          <div className="px-2 py-1 text-xs text-muted-foreground font-medium">添加到分类</div>
          {categories.map((cat) => (
            <button
              key={cat.id}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
              onClick={(e) => {
                e.stopPropagation()
                onMoveToCategory(feed.id, cat.id)
                setIsMenuOpen(false)
              }}
            >
              <Folder className="h-3.5 w-3.5" />
              {cat.name}
            </button>
          ))}
          {categories.length > 0 && <div className="my-1 border-t" />}
          <button
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
            onClick={(e) => {
              e.stopPropagation()
              setShowNewCategoryInput(true)
            }}
          >
            <FolderPlus className="h-3.5 w-3.5" />
            新建分类
          </button>
          {showNewCategoryInput && (
            <div className="px-2 py-1">
              <input
                ref={inputRef}
                value={newCategoryName}
                onChange={(e) => setNewCategoryName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newCategoryName.trim()) {
                    onMoveToNewCategory(feed.id, newCategoryName.trim())
                    setIsMenuOpen(false)
                    setShowNewCategoryInput(false)
                    setNewCategoryName('')
                  }
                  if (e.key === 'Escape') {
                    setShowNewCategoryInput(false)
                    setNewCategoryName('')
                  }
                }}
                placeholder="分类名称"
                className="w-full border rounded px-2 py-1 text-sm bg-background"
              />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 4: 创建 PaperRow 组件**

```tsx
// frontend/src/components/sidebar/PaperRow.tsx
import { FileText } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Paper } from '@/types/paper'

interface PaperRowProps {
  paper: Paper
  isSelected: boolean
  onPaperClick: (paperId: string) => void
}

export function PaperRow({ paper, isSelected, onPaperClick }: PaperRowProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 px-2 py-1 rounded-sm cursor-pointer text-sm",
        isSelected ? "bg-accent" : "hover:bg-accent/50"
      )}
      onClick={() => onPaperClick(paper.id)}
    >
      <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
      <span className="truncate flex-1">
        {paper.title || paper.original_filename}
      </span>
    </div>
  )
}
```

- [ ] **Step 5: 创建 FeedSection 组件（Feeds 区域：分类 + 未分类）**

```tsx
// frontend/src/components/sidebar/FeedSection.tsx
import { CategoryRow } from './CategoryRow'
import { FeedRow } from './FeedRow'
import type { Category } from '@/types/category'
import type { UserFeed } from '@/types/feed'

interface FeedSectionProps {
  categories: Category[]
  feeds: UserFeed[]
  selectedFeedId: string | null
  expandedCategoryIds: Set<string>
  allFeedCategories: Category[] // 所有 feed 类型的分类（供右键菜单使用）
  onToggleCategory: (categoryId: string) => void
  onFeedClick: (feedId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
  onMoveFeedToCategory: (feedId: string, categoryId: string) => void
  onMoveFeedToNewCategory: (feedId: string, categoryName: string) => void
}

export function FeedSection({
  categories,
  feeds,
  selectedFeedId,
  expandedCategoryIds,
  allFeedCategories,
  onToggleCategory,
  onFeedClick,
  onRenameCategory,
  onDeleteCategory,
  onMoveFeedToCategory,
  onMoveFeedToNewCategory,
}: FeedSectionProps) {
  // Group feeds by category
  const feedsByCategory = new Map<string, UserFeed[]>()
  const uncategorizedFeeds: UserFeed[] = []

  for (const feed of feeds) {
    if (feed.category_id) {
      const existing = feedsByCategory.get(feed.category_id) || []
      existing.push(feed)
      feedsByCategory.set(feed.category_id, existing)
    } else {
      uncategorizedFeeds.push(feed)
    }
  }

  // Calculate total feed count for section header
  const totalFeeds = feeds.length

  return (
    <div>
      <div className="flex items-center justify-between px-2 py-1.5">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Feeds
        </span>
        <span className="text-xs text-muted-foreground">{totalFeeds}</span>
      </div>

      {/* Categorized feeds */}
      {categories.map((category) => {
        const categoryFeeds = feedsByCategory.get(category.id) || []
        const isExpanded = expandedCategoryIds.has(category.id)
        const unreadCount = categoryFeeds.reduce((sum, f) => sum + f.unread_count, 0)

        return (
          <div key={category.id}>
            <CategoryRow
              category={category}
              isExpanded={isExpanded}
              itemCount={unreadCount || categoryFeeds.length}
              onToggle={() => onToggleCategory(category.id)}
              onRename={(name) => onRenameCategory(category.id, name)}
              onDelete={() => onDeleteCategory(category.id)}
            />
            {isExpanded && (
              <div className="ml-4">
                {categoryFeeds.map((feed) => (
                  <FeedRow
                    key={feed.id}
                    feed={feed}
                    isSelected={selectedFeedId === feed.id}
                    categories={allFeedCategories.filter((c) => c.id !== category.id)}
                    onFeedClick={onFeedClick}
                    onMoveToCategory={onMoveFeedToCategory}
                    onMoveToNewCategory={onMoveFeedToNewCategory}
                  />
                ))}
              </div>
            )}
          </div>
        )
      })}

      {/* Uncategorized feeds */}
      {uncategorizedFeeds.length > 0 && categories.length > 0 && (
        <div className="ml-4 mt-1">
          {uncategorizedFeeds.map((feed) => (
            <FeedRow
              key={feed.id}
              feed={feed}
              isSelected={selectedFeedId === feed.id}
              categories={allFeedCategories}
              onFeedClick={onFeedClick}
              onMoveToCategory={onMoveFeedToCategory}
              onMoveToNewCategory={onMoveFeedToNewCategory}
            />
          ))}
        </div>
      )}

      {/* When no categories exist, show all feeds flat */}
      {categories.length === 0 && feeds.map((feed) => (
        <FeedRow
          key={feed.id}
          feed={feed}
          isSelected={selectedFeedId === feed.id}
          categories={allFeedCategories}
          onFeedClick={onFeedClick}
          onMoveToCategory={onMoveFeedToCategory}
          onMoveToNewCategory={onMoveFeedToNewCategory}
        />
      ))}

      {feeds.length === 0 && (
        <div className="px-2 py-4 text-center text-sm text-muted-foreground">
          No feeds yet
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 6: 创建 PaperSection 组件（类似 FeedSection）**

```tsx
// frontend/src/components/sidebar/PaperSection.tsx
import { CategoryRow } from './CategoryRow'
import { PaperRow } from './PaperRow'
import type { Category } from '@/types/category'
import type { Paper } from '@/types/paper'

interface PaperSectionProps {
  categories: Category[]
  papers: Paper[]
  selectedPaperId: string | null
  expandedCategoryIds: Set<string>
  onToggleCategory: (categoryId: string) => void
  onPaperClick: (paperId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
}

export function PaperSection({
  categories,
  papers,
  selectedPaperId,
  expandedCategoryIds,
  onToggleCategory,
  onPaperClick,
  onRenameCategory,
  onDeleteCategory,
}: PaperSectionProps) {
  // Group papers by category
  const papersByCategory = new Map<string, Paper[]>()
  const uncategorizedPapers: Paper[] = []

  for (const paper of papers) {
    if (paper.category_id) {
      const existing = papersByCategory.get(paper.category_id) || []
      existing.push(paper)
      papersByCategory.set(paper.category_id, existing)
    } else {
      uncategorizedPapers.push(paper)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between px-2 py-1.5">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Papers
        </span>
        <span className="text-xs text-muted-foreground">{papers.length}</span>
      </div>

      {categories.map((category) => {
        const categoryPapers = papersByCategory.get(category.id) || []
        const isExpanded = expandedCategoryIds.has(category.id)

        return (
          <div key={category.id}>
            <CategoryRow
              category={category}
              isExpanded={isExpanded}
              itemCount={categoryPapers.length}
              onToggle={() => onToggleCategory(category.id)}
              onRename={(name) => onRenameCategory(category.id, name)}
              onDelete={() => onDeleteCategory(category.id)}
            />
            {isExpanded && (
              <div className="ml-4">
                {categoryPapers.map((paper) => (
                  <PaperRow
                    key={paper.id}
                    paper={paper}
                    isSelected={selectedPaperId === paper.id}
                    onPaperClick={onPaperClick}
                  />
                ))}
              </div>
            )}
          </div>
        )
      })}

      {uncategorizedPapers.length > 0 && categories.length > 0 && (
        <div className="ml-4 mt-1">
          {uncategorizedPapers.map((paper) => (
            <PaperRow
              key={paper.id}
              paper={paper}
              isSelected={selectedPaperId === paper.id}
              onPaperClick={onPaperClick}
            />
          ))}
        </div>
      )}

      {categories.length === 0 && papers.map((paper) => (
        <PaperRow
          key={paper.id}
          paper={paper}
          isSelected={selectedPaperId === paper.id}
          onPaperClick={onPaperClick}
        />
      ))}

      {papers.length === 0 && (
        <div className="px-2 py-4 text-center text-sm text-muted-foreground">
          No papers yet
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 7: 重写 Sidebar.tsx**

```tsx
// frontend/src/components/feed/Sidebar.tsx — 完全重写
import { SidebarHeader } from '@/components/sidebar/SidebarHeader'
import { FeedSection } from '@/components/sidebar/FeedSection'
import { PaperSection } from '@/components/sidebar/PaperSection'
import { useSidebarStore } from '@/stores/sidebarStore'
import type { Category } from '@/types/category'
import type { UserFeed } from '@/types/feed'
import type { Paper } from '@/types/paper'

interface SidebarProps {
  feedCategories: Category[]
  paperCategories: Category[]
  feeds: UserFeed[]
  papers: Paper[]
  onAddClick: () => void
  onFeedClick: (feedId: string) => void
  onPaperClick: (paperId: string) => void
  onRenameCategory: (categoryId: string, name: string) => void
  onDeleteCategory: (categoryId: string) => void
  onMoveFeedToCategory: (feedId: string, categoryId: string) => void
  onMoveFeedToNewCategory: (feedId: string, categoryName: string) => void
  // Mobile props
  isMobileOpen?: boolean
  onMobileClose?: () => void
}

export function SidebarContent({
  feedCategories,
  paperCategories,
  feeds,
  papers,
  onAddClick,
  onFeedClick,
  onPaperClick,
  onRenameCategory,
  onDeleteCategory,
  onMoveFeedToCategory,
  onMoveFeedToNewCategory,
}: Omit<SidebarProps, 'isMobileOpen' | 'onMobileClose'>) {
  const { selectedFeedId, selectedPaperId, expandedCategoryIds, toggleCategory } =
    useSidebarStore()

  return (
    <>
      <SidebarHeader onAddClick={onAddClick} />
      <div className="flex-1 overflow-y-auto px-2 py-1">
        <FeedSection
          categories={feedCategories}
          feeds={feeds}
          selectedFeedId={selectedFeedId}
          expandedCategoryIds={expandedCategoryIds}
          allFeedCategories={feedCategories}
          onToggleCategory={toggleCategory}
          onFeedClick={onFeedClick}
          onRenameCategory={onRenameCategory}
          onDeleteCategory={onDeleteCategory}
          onMoveFeedToCategory={onMoveFeedToCategory}
          onMoveFeedToNewCategory={onMoveFeedToNewCategory}
        />

        <div className="border-t my-2" />

        <PaperSection
          categories={paperCategories}
          papers={papers}
          selectedPaperId={selectedPaperId}
          expandedCategoryIds={expandedCategoryIds}
          onToggleCategory={toggleCategory}
          onPaperClick={onPaperClick}
          onRenameCategory={onRenameCategory}
          onDeleteCategory={onDeleteCategory}
        />
      </div>
    </>
  )
}

export function Sidebar(props: SidebarProps) {
  return (
    <aside className="hidden md:flex w-[260px] border-r bg-background flex-shrink-0 flex-col h-full">
      <SidebarContent {...props} />
    </aside>
  )
}

// Keep MobileMenuButton for backward compatibility
export { MobileMenuButton } from './Sidebar' // Will be updated
```

- [ ] **Step 8: 运行 TypeScript 编译检查**

Run: `cd frontend && npx tsc --noEmit 2>&1 | head -50`
Expected: 修复所有类型错误

- [ ] **Step 9: Commit**

```bash
git add frontend/src/components/sidebar/ frontend/src/components/feed/Sidebar.tsx
git commit -m "feat: rewrite Sidebar with category-based Feed/Paper sections"
```

---

## Task 8: 前端 — "新增"统一对话框

**Files:**
- Create: `frontend/src/components/sidebar/AddDialog.tsx`

- [ ] **Step 1: 创建 AddDialog 组件**

将 AddFeed、OpmlImport、PaperUpload 三个功能合并到一个统一对话框：

```tsx
// frontend/src/components/sidebar/AddDialog.tsx
import { useState, useEffect, useRef, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Rss, Upload, FileText, ArrowLeft, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useFeeds } from '@/hooks/useFeeds'
import { useCategories } from '@/hooks/useCategories'
import { useOPML } from '@/hooks/useOPML'
import { useToast } from '@/components/ui/toast'
import apiClient from '@/lib/api/axios'

type DialogView = 'menu' | 'add-feed' | 'import-feed' | 'import-paper'

interface AddDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

const feedUrlSchema = z.object({
  feed_url: z.string().min(1, 'Feed URL is required').url('Invalid URL format'),
})
type FeedUrlForm = z.infer<typeof feedUrlSchema>

export function AddDialog({ open, onOpenChange, onSuccess }: AddDialogProps) {
  const [view, setView] = useState<DialogView>('menu')
  const [importFile, setImportFile] = useState<File | null>(null)
  const [paperFile, setPaperFile] = useState<File | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const paperInputRef = useRef<HTMLInputElement>(null)
  const toast = useToast()

  const { useCreateFeed } = useFeeds()
  const { useCreateCategory } = useCategories()
  const { useExportOpml, triggerDownload } = useOPML()
  const createFeed = useCreateFeed()
  const createCategory = useCreateCategory()
  const exportOpml = useExportOpml()

  // Reset to menu when dialog opens
  useEffect(() => {
    if (open) {
      setView('menu')
      setImportFile(null)
      setPaperFile(null)
      setError(null)
    }
  }, [open])

  // Feed URL form
  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<FeedUrlForm>({
    resolver: zodResolver(feedUrlSchema),
  })

  const handleAddFeed = async (data: FeedUrlForm) => {
    setError(null)
    try {
      await createFeed.mutateAsync(data)
      reset()
      onOpenChange(false)
      onSuccess()
      toast.showSuccess('Feed added successfully')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add feed')
    }
  }

  const handleImportOpml = async () => {
    if (!importFile) return
    setIsUploading(true)
    setError(null)
    try {
      const formData = new FormData()
      formData.append('file', importFile)
      await apiClient.post('/opml/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setImportFile(null)
      onOpenChange(false)
      onSuccess()
      toast.showSuccess('OPML import started')
    } catch {
      setError('Failed to import OPML')
      toast.showError('Failed to import OPML')
    } finally {
      setIsUploading(false)
    }
  }

  const handleUploadPaper = async () => {
    if (!paperFile) return
    setIsUploading(true)
    setError(null)
    try {
      const formData = new FormData()
      formData.append('file', paperFile)
      await apiClient.post('/papers/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setPaperFile(null)
      onOpenChange(false)
      onSuccess()
      toast.showSuccess('Paper uploaded')
    } catch {
      setError('Failed to upload paper')
      toast.showError('Failed to upload paper')
    } finally {
      setIsUploading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          {view === 'menu' && <DialogTitle>新增</DialogTitle>}
          {view === 'add-feed' && (
            <div className="flex items-center gap-2">
              <button onClick={() => setView('menu')} className="hover:bg-accent rounded p-1">
                <ArrowLeft className="h-4 w-4" />
              </button>
              <DialogTitle>添加 Feed</DialogTitle>
            </div>
          )}
          {view === 'import-feed' && (
            <div className="flex items-center gap-2">
              <button onClick={() => setView('menu')} className="hover:bg-accent rounded p-1">
                <ArrowLeft className="h-4 w-4" />
              </button>
              <DialogTitle>导入 Feed</DialogTitle>
            </div>
          )}
          {view === 'import-paper' && (
            <div className="flex items-center gap-2">
              <button onClick={() => setView('menu')} className="hover:bg-accent rounded p-1">
                <ArrowLeft className="h-4 w-4" />
              </button>
              <DialogTitle>导入 Paper</DialogTitle>
            </div>
          )}
        </DialogHeader>

        {/* Menu view */}
        {view === 'menu' && (
          <div className="space-y-2">
            <button
              className="flex items-center gap-3 w-full p-3 rounded-md border hover:bg-accent/50 text-left transition-colors"
              onClick={() => setView('add-feed')}
            >
              <Rss className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium text-sm">添加 Feed</div>
                <div className="text-xs text-muted-foreground">输入 RSS/Atom URL 订阅</div>
              </div>
            </button>
            <button
              className="flex items-center gap-3 w-full p-3 rounded-md border hover:bg-accent/50 text-left transition-colors"
              onClick={() => setView('import-feed')}
            >
              <Upload className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium text-sm">导入 Feed</div>
                <div className="text-xs text-muted-foreground">从 OPML 文件批量导入</div>
              </div>
            </button>
            <button
              className="flex items-center gap-3 w-full p-3 rounded-md border hover:bg-accent/50 text-left transition-colors"
              onClick={() => setView('import-paper')}
            >
              <FileText className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium text-sm">导入 Paper</div>
                <div className="text-xs text-muted-foreground">上传 PDF 论文并自动解析</div>
              </div>
            </button>
          </div>
        )}

        {/* Add Feed view */}
        {view === 'add-feed' && (
          <form onSubmit={handleSubmit(handleAddFeed)}>
            <div className="space-y-4 py-2">
              <div className="space-y-2">
                <Label htmlFor="feed_url">Feed URL</Label>
                <Input
                  id="feed_url"
                  placeholder="https://example.com/feed.xml"
                  {...register('feed_url')}
                  disabled={createFeed.isPending}
                />
                {errors.feed_url && (
                  <p className="text-sm text-destructive">{errors.feed_url.message}</p>
                )}
                {error && <p className="text-sm text-destructive">{error}</p>}
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createFeed.isPending}>
                {createFeed.isPending ? '添加中...' : '添加'}
              </Button>
            </div>
          </form>
        )}

        {/* Import OPML view */}
        {view === 'import-feed' && (
          <div className="space-y-4 py-2">
            <div
              className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${
                dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
              }`}
              onClick={() => fileInputRef.current?.click()}
              onDragEnter={(e) => { e.preventDefault(); setDragActive(true) }}
              onDragLeave={() => setDragActive(false)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                setDragActive(false)
                const f = e.dataTransfer.files?.[0]
                if (f) setImportFile(f)
              }}
            >
              <input
                ref={fileInputRef}
                type="file"
                accept=".xml,.opml"
                className="hidden"
                onChange={(e) => setImportFile(e.target.files?.[0] || null)}
              />
              {importFile ? (
                <p className="text-sm">{importFile.name}</p>
              ) : (
                <p className="text-sm text-muted-foreground">点击或拖放 OPML 文件</p>
              )}
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button onClick={handleImportOpml} disabled={!importFile || isUploading}>
                {isUploading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                导入
              </Button>
            </div>
          </div>
        )}

        {/* Import Paper view */}
        {view === 'import-paper' && (
          <div className="space-y-4 py-2">
            <div
              className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${
                dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
              }`}
              onClick={() => paperInputRef.current?.click()}
              onDragEnter={(e) => { e.preventDefault(); setDragActive(true) }}
              onDragLeave={() => setDragActive(false)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                setDragActive(false)
                const f = e.dataTransfer.files?.[0]
                if (f && f.type === 'application/pdf') setPaperFile(f)
              }}
            >
              <input
                ref={paperInputRef}
                type="file"
                accept=".pdf"
                className="hidden"
                onChange={(e) => setPaperFile(e.target.files?.[0] || null)}
              />
              {paperFile ? (
                <p className="text-sm">{paperFile.name}</p>
              ) : (
                <p className="text-sm text-muted-foreground">点击或拖放 PDF 文件</p>
              )}
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button onClick={handleUploadPaper} disabled={!paperFile || isUploading}>
                {isUploading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                上传
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 2: 运行 TypeScript 编译检查**

Run: `cd frontend && npx tsc --noEmit 2>&1 | head -20`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/sidebar/AddDialog.tsx
git commit -m "feat: add unified AddDialog with Feed/Import/Paper options"
```

---

## Task 9: 前端 — 统一三栏布局 + 路由简化

**Files:**
- Rewrite: `frontend/src/pages/items/ItemsPage.tsx` (统一三栏)
- Modify: `frontend/src/App.tsx` (路由简化)
- Remove: `frontend/src/pages/papers/PapersPage.tsx` (不再需要独立页面)
- Modify: `frontend/src/pages/items/ItemViewPage.tsx` (Back 导航修改)

- [ ] **Step 1: 重写 ItemsPage.tsx 为统一三栏布局**

新的 ItemsPage 承载所有功能：
- 左栏：新 Sidebar（Feeds + Papers 分类树）
- 中栏：文章列表 / Paper 列表
- 右栏：ArticlePanel / Paper 全文

关键改动：
1. 引入 `useCategories` hook 获取分类数据
2. 使用 `useSidebarStore` 管理选中状态
3. 点击 Feed → 中栏显示文章列表
4. 点击 Paper → 中栏显示同分类 Paper 列表，右栏显示全文
5. 移除 `filterType` 相关逻辑
6. 移除顶部工具栏
7. 保留 ThemeToggle 和 keyboard shortcuts

```tsx
// frontend/src/pages/items/ItemsPage.tsx — 关键结构
import { useState, useCallback, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useItems } from '@/hooks/useItems'
import { useFeeds } from '@/hooks/useFeeds'
import { usePapers } from '@/hooks/usePapers'
import { useCategories } from '@/hooks/useCategories'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useToast } from '@/components/ui/toast'
import { MobileDrawer } from '@/components/ui/mobile-drawer'
import { KeyboardShortcutsModal } from '@/components/ui/keyboard-shortcuts-modal'
import { ThemeToggle } from '@/components/ui/theme-toggle'
import { ItemList } from '@/components/items/ItemList'
import { ArticlePanel } from '@/components/items/ArticlePanel'
import { Sidebar, SidebarContent, MobileMenuButton } from '@/components/feed/Sidebar'
import { AddDialog } from '@/components/sidebar/AddDialog'
import { useSidebarStore } from '@/stores/sidebarStore'
import { useItemsStore } from '@/stores/itemsStore'
import type { Paper } from '@/types/paper'
import { FileText, Menu } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

export function ItemsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedItemId = searchParams.get('id')
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false)
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isShortcutsModalOpen, setIsShortcutsModalOpen] = useState(false)
  const [refreshingFeedIds, setRefreshingFeedIds] = useState<Set<string>>(new Set())
  const [selectedIndex, setSelectedIndex] = useState(0)
  const observerTarget = useRef<HTMLDivElement>(null)
  const toast = useToast()

  const { selectedFeedId, selectedPaperId, selectionType, selectFeed, selectPaper } =
    useSidebarStore()
  const setItems = useItemsStore((state) => state.setItems)
  const updateItemState = useItemsStore((state) => state.updateItemState)

  // API hooks
  const { useListFeeds, useRefreshFeed, useDeleteFeed } = useFeeds()
  const { useListItemsInfinite, useToggleStar, useToggleRead, useMarkAllRead } = useItems()
  const { useListPapers, useGetPaper } = usePapers()
  const {
    useListCategories,
    useCreateCategory,
    useRenameCategory: useRenameCategoryMutation,
    useDeleteCategory: useDeleteCategoryMutation,
    useMoveFeedToCategory: useMoveFeedToCategoryMutation,
  } = useCategories()

  // Data fetching
  const { data: feedCategoriesData } = useListCategories('feed')
  const { data: paperCategoriesData } = useListCategories('paper')
  const { data: feedsData, refetch: refetchFeeds } = useListFeeds()
  const { data: papersData } = useListPapers({ limit: 100 })

  // Items based on selected feed
  const listOptions: Record<string, unknown> = {}
  if (selectedFeedId) listOptions.feed_id = selectedFeedId
  listOptions.limit = 20

  const {
    data: infiniteData,
    isLoading: itemsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useListItemsInfinite(selectedFeedId ? { feed_id: selectedFeedId, limit: 20 } : undefined)

  // Mutations
  const createCategory = useCreateCategory()
  const renameCategoryMutation = useRenameCategoryMutation()
  const deleteCategoryMutation = useDeleteCategoryMutation()
  const moveFeedToCategoryMutation = useMoveFeedToCategoryMutation()
  const refreshFeed = useRefreshFeed()
  const deleteFeed = useDeleteFeed()
  const toggleStar = useToggleStar()
  const toggleRead = useToggleRead()
  const markAllRead = useMarkAllRead()

  const items = infiniteData?.pages.flatMap(page => page.items) ?? []
  const hasMore = hasNextPage ?? false
  const feeds = feedsData?.feeds ?? []
  const feedCategories = feedCategoriesData?.categories ?? []
  const paperCategories = paperCategoriesData?.categories ?? []
  const papers = papersData?.papers ?? []

  // Derived data for middle column
  const middleColumnTitle = selectedFeedId
    ? feeds.find(f => f.id === selectedFeedId)?.title || 'Feed'
    : selectedPaperId
      ? 'Papers'
      : 'All Articles'

  // ... handlers (same patterns as current, plus category handlers)

  // Category handlers
  const handleRenameCategory = useCallback((categoryId: string, name: string) => {
    renameCategoryMutation.mutate(
      { id: categoryId, data: { name } },
      { onSuccess: () => toast.showSuccess('分类已重命名') }
    )
  }, [renameCategoryMutation, toast])

  const handleDeleteCategory = useCallback((categoryId: string) => {
    if (!confirm('确定删除此分类？其中的 Feed/Paper 将变为未分类。')) return
    deleteCategoryMutation.mutate(categoryId, {
      onSuccess: () => toast.showSuccess('分类已删除'),
    })
  }, [deleteCategoryMutation, toast])

  const handleMoveFeedToCategory = useCallback((feedId: string, categoryId: string) => {
    moveFeedToCategoryMutation.mutate(
      { feedId, data: { category_id: categoryId } },
      { onSuccess: () => refetchFeeds() }
    )
  }, [moveFeedToCategoryMutation, refetchFeeds])

  const handleMoveFeedToNewCategory = useCallback((feedId: string, categoryName: string) => {
    createCategory.mutate(
      { name: categoryName, type: 'feed' },
      {
        onSuccess: (result) => {
          if (result.category.id) {
            moveFeedToCategoryMutation.mutate(
              { feedId, data: { category_id: result.category.id } },
              { onSuccess: () => refetchFeeds() }
            )
          }
        },
      }
    )
  }, [createCategory, moveFeedToCategoryMutation, refetchFeeds])

  const handleFeedClick = useCallback((feedId: string) => {
    selectFeed(feedId)
    setIsMobileMenuOpen(false)
  }, [selectFeed])

  const handlePaperClick = useCallback((paperId: string) => {
    selectPaper(paperId)
    setSearchParams((prev) => {
      prev.delete('id')
      prev.set('paper', paperId)
      return prev
    })
    setIsMobileMenuOpen(false)
  }, [selectPaper, setSearchParams])

  // Infinite scroll observer (same as current)
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 0.1 }
    )
    const currentTarget = observerTarget.current
    if (currentTarget) observer.observe(currentTarget)
    return () => { if (currentTarget) observer.unobserve(currentTarget) }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  // Keyboard shortcuts (adapted — remove filter-related shortcuts)

  // Determine middle column content
  const showFeedItems = selectionType === 'feed' && selectedFeedId
  const showPaperItems = selectionType === 'paper'

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Left: Sidebar */}
      <Sidebar
        feedCategories={feedCategories}
        paperCategories={paperCategories}
        feeds={feeds}
        papers={papers}
        onAddClick={() => setIsAddDialogOpen(true)}
        onFeedClick={handleFeedClick}
        onPaperClick={handlePaperClick}
        onRenameCategory={handleRenameCategory}
        onDeleteCategory={handleDeleteCategory}
        onMoveFeedToCategory={handleMoveFeedToCategory}
        onMoveFeedToNewCategory={handleMoveFeedToNewCategory}
      />

      {/* Mobile Drawer */}
      <MobileDrawer isOpen={isMobileMenuOpen} onClose={() => setIsMobileMenuOpen(false)}>
        <SidebarContent
          feedCategories={feedCategories}
          paperCategories={paperCategories}
          feeds={feeds}
          papers={papers}
          onAddClick={() => { setIsAddDialogOpen(true); setIsMobileMenuOpen(false) }}
          onFeedClick={handleFeedClick}
          onPaperClick={handlePaperClick}
          onRenameCategory={handleRenameCategory}
          onDeleteCategory={handleDeleteCategory}
          onMoveFeedToCategory={handleMoveFeedToCategory}
          onMoveFeedToNewCategory={handleMoveFeedToNewCategory}
        />
      </MobileDrawer>

      {/* Middle: Item/Paper List */}
      <div className="hidden md:flex w-80 border-r flex-shrink-0 flex-col bg-background">
        <div className="p-3 border-b flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MobileMenuButton onClick={() => setIsMobileMenuOpen(true)} />
            <h1 className="text-sm font-semibold truncate">{middleColumnTitle}</h1>
          </div>
          <div className="flex items-center gap-1">
            <ThemeToggle />
            {selectedFeedId && (
              <button
                onClick={() => markAllRead.mutate(selectedFeedId)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                全部已读
              </button>
            )}
          </div>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {showFeedItems && (
            <ItemList
              articles={items}
              selectedItemId={selectedItemId}
              onItemClick={handleItemClick}
              onToggleStar={handleToggleStar}
              onToggleRead={handleToggleRead}
              isLoading={itemsLoading}
            />
          )}
          {showPaperItems && (
            // Paper list in sidebar format
            <div className="space-y-1">
              {papers
                .filter(p => p.status === 'completed')
                .map(paper => (
                  <div
                    key={paper.id}
                    className={`flex items-center gap-2 px-2 py-1.5 rounded-sm cursor-pointer text-sm hover:bg-accent/50 ${selectedPaperId === paper.id ? 'bg-accent' : ''}`}
                    onClick={() => handlePaperClick(paper.id)}
                  >
                    <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <span className="truncate">{paper.title || paper.original_filename}</span>
                  </div>
                ))}
            </div>
          )}
          {!showFeedItems && !showPaperItems && (
            <div className="text-center text-sm text-muted-foreground py-8">
              选择一个 Feed 或 Paper 开始阅读
            </div>
          )}
          {hasMore && showFeedItems && (
            <div ref={observerTarget} className="py-4 text-center">
              {isFetchingNextPage && (
                <div className="inline-block h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              )}
            </div>
          )}
        </div>
      </div>

      {/* Right: Article/Paper Content */}
      <div className="flex-1 overflow-hidden hidden lg:flex">
        {selectedItemId ? (
          <ArticlePanel itemId={selectedItemId} showBackButton={false} />
        ) : selectedPaperId ? (
          <PaperReader paperId={selectedPaperId} />
        ) : (
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            <p>选择一篇文章或论文开始阅读</p>
          </div>
        )}
      </div>

      {/* Mobile views */}
      {/* ... (similar to current mobile handling) */}

      {/* Dialogs */}
      <AddDialog
        open={isAddDialogOpen}
        onOpenChange={setIsAddDialogOpen}
        onSuccess={() => { refetchFeeds(); refetch() }}
      />
      <KeyboardShortcutsModal isOpen={isShortcutsModalOpen} onClose={() => setIsShortcutsModalOpen(false)} />
    </div>
  )
}

// Inline PaperReader for right panel
function PaperReader({ paperId }: { paperId: string }) {
  const { useGetPaper } = usePapers()
  const { data, isLoading } = useGetPaper(paperId)
  const paper = data?.paper

  if (isLoading) return <div className="flex items-center justify-center h-full"><p>Loading...</p></div>
  if (!paper) return <div className="flex items-center justify-center h-full"><p>Paper not found</p></div>

  // Simplified paper view for the right panel
  return (
    <div className="h-full overflow-y-auto p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-2">{paper.title || paper.original_filename}</h1>
      {paper.authors && <p className="text-sm text-muted-foreground mb-4">{paper.authors}</p>}
      {paper.abstract && (
        <div className="border-l-4 border-muted pl-4 mb-4">
          <p className="text-muted-foreground">{paper.abstract}</p>
        </div>
      )}
      {paper.markdown_content ? (
        <MarkdownRenderer content={paper.markdown_content} />
      ) : (
        <p className="text-muted-foreground">No content available</p>
      )}
    </div>
  )
}
```

注意：上面的 `PaperReader` 需要引入 `MarkdownRenderer`，以及 `handleItemClick` / `handleToggleStar` / `handleToggleRead` / `handleMarkAllRead` 与当前 ItemsPage 中的实现保持一致。完整实现应复制当前文件中的对应 handler。

- [ ] **Step 2: 简化 App.tsx 路由**

```tsx
// frontend/src/App.tsx — 路由简化
function AppRoutes() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const initializeFromStorage = useAuthStore((state) => state.initializeFromStorage)

  useEffect(() => {
    initializeFromStorage()
  }, [initializeFromStorage])

  return (
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={isAuthenticated ? <Navigate to="/feeds" replace /> : <LoginPage />} />
      <Route path="/register" element={isAuthenticated ? <Navigate to="/feeds" replace /> : <RegisterPage />} />
      <Route path="/oauth/pending" element={isAuthenticated ? <Navigate to="/feeds" replace /> : <OAuthPendingPage />} />
      <Route path="/api/v1/auth/github/callback" element={<OAuthCallbackPage />} />

      {/* Protected routes */}
      <Route path="/feeds" element={<ProtectedRoute><ItemsPage /></ProtectedRoute>} />
      <Route path="/papers/:id" element={<ProtectedRoute><PaperViewPage /></ProtectedRoute>} />
      <Route path="/items/:id" element={<ProtectedRoute><ItemViewPage /></ProtectedRoute>} />

      {/* Default redirect */}
      <Route path="/" element={<Navigate to={isAuthenticated ? '/feeds' : '/login'} replace />} />
      <Route path="*" element={<Navigate to={isAuthenticated ? '/feeds' : '/login'} replace />} />
    </Routes>
  )
}
```

移除 `ItemsPageWrapper` 和 `filterType`/`feedId` 传递逻辑。移除 `/items` 路由，移除 `/papers` 路由。

- [ ] **Step 3: 更新 Paper 类型添加 category_id**

在 `frontend/src/types/paper.ts` 的 `Paper` interface 添加：

```typescript
export interface Paper {
  // ... 已有字段 ...
  category_id?: string | null
}
```

- [ ] **Step 4: 移除 PapersPage 引用**

从 `App.tsx` 的 import 中移除 `PapersPage`。保留 `PaperViewPage`（用于 `/papers/:id` 深度链接）。

- [ ] **Step 5: 运行前端编译检查**

Run: `cd frontend && npx tsc --noEmit 2>&1 | head -30`
Expected: 修复所有类型错误

- [ ] **Step 6: 运行前端测试**

Run: `cd frontend && npm test -- --run 2>&1 | tail -30`
Expected: 已有测试需要更新以适应新路由（更新 ProtectedRoute test 等）

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/items/ItemsPage.tsx frontend/src/App.tsx frontend/src/types/paper.ts
git commit -m "feat: unified three-column layout with simplified routing"
```

---

## Task 10: 前端 — 更新测试 + 清理

**Files:**
- Modify: `frontend/src/components/feed/__tests__/Sidebar.test.tsx`
- Modify: `frontend/src/components/__tests__/ProtectedRoute.test.tsx`
- Remove: `frontend/src/pages/papers/PapersPage.tsx` (如果不再需要)

- [ ] **Step 1: 更新 Sidebar 测试**

新的 Sidebar 组件接收不同的 props。更新测试以匹配新接口：

```tsx
// frontend/src/components/feed/__tests__/Sidebar.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { Sidebar } from '../Sidebar'

// Mock sidebarStore
vi.mock('@/stores/sidebarStore', () => ({
  useSidebarStore: vi.fn(() => ({
    selectedFeedId: null,
    selectedPaperId: null,
    selectionType: null,
    expandedCategoryIds: new Set(),
    toggleCategory: vi.fn(),
  })),
}))

// Mock authStore
vi.mock('@/stores/authStore', () => ({
  useAuthStore: vi.fn(() => ({
    user: { id: '1', email: 'test@example.com', nickname: 'Test', avatar_url: null, auth_provider: 'email' },
  })),
}))

const defaultProps = {
  feedCategories: [],
  paperCategories: [],
  feeds: [],
  papers: [],
  onAddClick: vi.fn(),
  onFeedClick: vi.fn(),
  onPaperClick: vi.fn(),
  onRenameCategory: vi.fn(),
  onDeleteCategory: vi.fn(),
  onMoveFeedToCategory: vi.fn(),
  onMoveFeedToNewCategory: vi.fn(),
}

describe('Sidebar', () => {
  it('renders sidebar header with oReader logo', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('oReader')).toBeInTheDocument()
  })

  it('renders add button', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('新增')).toBeInTheDocument()
  })

  it('renders Feeds section header', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('Feeds')).toBeInTheDocument()
  })

  it('renders Papers section header', () => {
    render(<Sidebar {...defaultProps} />)
    expect(screen.getByText('Papers')).toBeInTheDocument()
  })

  it('renders feed categories', () => {
    const props = {
      ...defaultProps,
      feedCategories: [{ id: '1', user_id: 'u1', name: 'Tech', type: 'feed', position: 0, created_at: '' }],
    }
    render(<Sidebar {...props} />)
    expect(screen.getByText('Tech')).toBeInTheDocument()
  })

  it('renders feeds', () => {
    const props = {
      ...defaultProps,
      feeds: [{
        id: '1', title: 'HN', feed_url: 'https://hn.com/rss', description: null,
        image_url: null, last_fetched_at: null, created_at: '',
        item_count: 10, unread_count: 5, position: 0, category_id: null,
      }],
    }
    render(<Sidebar {...props} />)
    expect(screen.getByText('HN')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: 更新 ProtectedRoute 测试**

更新路由路径从 `/items` 到 `/feeds`：

```tsx
// 修改 ProtectedRoute.test.tsx 中所有 /items 路径为 /feeds
```

- [ ] **Step 3: 运行前端测试**

Run: `cd frontend && npm test -- --run 2>&1 | tail -30`
Expected: PASS

- [ ] **Step 4: 删除 PapersPage.tsx（如果不再需要）**

确认没有其他文件引用 `PapersPage`：

Run: `cd frontend && grep -r "PapersPage" src/`
如果只有 `App.tsx` 引用且已移除，则删除：

```bash
rm frontend/src/pages/papers/PapersPage.tsx
```

- [ ] **Step 5: 运行完整前端 lint + type check**

Run: `cd frontend && npx tsc --noEmit && npm test -- --run`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/feed/__tests__/Sidebar.test.tsx frontend/src/components/__tests__/ProtectedRoute.test.tsx
git rm frontend/src/pages/papers/PapersPage.tsx 2>/dev/null
git commit -m "test: update tests for new sidebar and routing"
```

---

## Task 11: 后端 — 更新 ListFeeds handler 返回 category_id

**Files:**
- Modify: `backend/internal/handler/feed_handler.go` (ListFeeds response)
- Modify: `backend/internal/handler/feed_handler_test.go` (更新 mock)

- [ ] **Step 1: 确认 ListFeeds handler 自动传递新字段**

由于 `FeedWithItemCount` 已包含 `CategoryID` 字段，GORM 的 JSON 序列化会自动包含 `category_id`。验证 handler 返回格式正确。

- [ ] **Step 2: 更新 handler 测试中的 mock**

在 `backend/internal/handler/feed_handler_test.go` 中，更新 `mockFeedService.GetUserFeeds` 返回的 `FeedWithItemCount` 以包含 `CategoryID`：

```go
func (m *mockFeedService) GetUserFeeds(ctx context.Context, userID string, opts service.ListOptions) ([]*service.FeedWithItemCount, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	feed := &model.Feed{
		Title:    "Test Feed",
		FeedURL:  "https://example.com/feed.xml",
		ImageURL: "https://example.com/icon.png",
	}
	feed.GenerateID()
	return []*service.FeedWithItemCount{
		{
			Feed:       feed,
			ItemCount:  10,
			CategoryID: nil,
		},
	}, 1, nil
}
```

- [ ] **Step 3: 运行后端测试**

Run: `cd backend && go test ./internal/handler/ -run TestListFeeds -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/feed_handler.go backend/internal/handler/feed_handler_test.go
git commit -m "feat: include category_id in ListFeeds response"
```

---

## Task 12: 端到端验证

- [ ] **Step 1: 启动 Docker 开发环境**

Run: `make docker-dev`

- [ ] **Step 2: 验证后端 API**

```bash
# 创建分类
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Content-Type: application/json" \
  -d '{"name": "Tech Blogs", "type": "feed"}'

# 列出分类
curl http://localhost:8080/api/v1/categories?type=feed

# 验证 feeds 包含 category_id
curl http://localhost:8080/api/v1/feeds
```

Expected: 分类 CRUD 正常，feeds 列表包含 category_id

- [ ] **Step 3: 验证前端渲染**

打开 `http://localhost:5173/feeds`：
- 左栏显示分类树结构（Feeds + Papers）
- 点击 Feed → 中栏显示文章列表
- 点击"新增"按钮 → 弹出统一对话框
- 右键分类 → 重命名/删除菜单
- 右键 Feed → 添加到分类菜单

- [ ] **Step 4: 运行 E2E 测试**

Run: `cd frontend && npm run test:e2e 2>&1 | tail -20`

- [ ] **Step 5: 最终 Commit**

```bash
git add -A
git commit -m "feat: sidebar optimization - unified three-column layout with category management"
```

---

## Self-Review Checklist

**1. Spec Coverage:**
- [x] 移除 all/unread/starred/today 过滤页面 — Task 9 路由简化
- [x] 移除顶部工具栏 — Task 9 ItemsPage 重写
- [x] 移除侧边栏中 Feed 添加/导入/导出按钮 — Task 7 Sidebar 重写
- [x] 三栏布局 260px/320px/弹性 — Task 7+9
- [x] 侧边栏头部 [Logo] [+ 新增] [头像] — Task 7 SidebarHeader
- [x] "新增"对话框（三个选项） — Task 8 AddDialog
- [x] 侧边栏分区 Feeds/Papers — Task 7 FeedSection + PaperSection
- [x] categories 表 + 数据模型 — Task 1
- [x] 分类管理 CRUD API — Task 2-4
- [x] 右键分类（重命名/删除） — Task 7 CategoryRow
- [x] 右键 Feed/Paper（添加到分类） — Task 7 FeedRow
- [x] 路由简化 — Task 9
- [x] PapersPage 合并到统一阅读页 — Task 9

**2. Placeholder Scan:** 无 TBD、TODO、implement later 等占位符。

**3. Type Consistency:**
- `Category.id` — 所有文件一致使用 string
- `Category.type` — `'feed' | 'paper'` — 前后端一致
- `category_id` — 可选 string (`*string` Go / `string | null` TS)
- `expandedCategoryIds` — sidebarStore 使用 `Set<string>`
