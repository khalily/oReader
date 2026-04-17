package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
	"github.com/khalily/oreader/internal/testutil"
)

func setupCategoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)

	err := db.AutoMigrate(
		&model.User{},
		&model.Category{},
	)
	require.NoError(t, err)

	return db
}

func createTestUserForCategory(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, NewUserRepository(db).Create(context.Background(), user))
	return user
}

func TestCategoryRepository_Create(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	cat := &model.Category{
		UserID: user.ID,
		Name:   "Tech Blogs",
		Type:   model.CategoryTypeFeed,
	}
	require.NoError(t, cat.GenerateID())

	err := repo.Create(ctx, cat)
	require.NoError(t, err)
	assert.NotEmpty(t, cat.ID)
}

func TestCategoryRepository_ListByUserID(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	feedCat := &model.Category{UserID: user.ID, Name: "Tech", Type: model.CategoryTypeFeed}
	feedCat.GenerateID()
	repo.Create(ctx, feedCat)

	paperCat := &model.Category{UserID: user.ID, Name: "ML", Type: model.CategoryTypePaper}
	paperCat.GenerateID()
	repo.Create(ctx, paperCat)

	feedCats, err := repo.ListByUserID(ctx, user.ID, model.CategoryTypeFeed)
	require.NoError(t, err)
	assert.Len(t, feedCats, 1)
	assert.Equal(t, "Tech", feedCats[0].Name)

	allCats, err := repo.ListByUserID(ctx, user.ID, "")
	require.NoError(t, err)
	assert.Len(t, allCats, 2)
}

func TestCategoryRepository_GetMaxPosition(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	cat1 := &model.Category{UserID: user.ID, Name: "A", Type: model.CategoryTypeFeed, Position: 0}
	cat1.GenerateID()
	repo.Create(ctx, cat1)

	cat2 := &model.Category{UserID: user.ID, Name: "B", Type: model.CategoryTypeFeed, Position: 5}
	cat2.GenerateID()
	repo.Create(ctx, cat2)

	maxPos, err := repo.GetMaxPosition(ctx, user.ID, model.CategoryTypeFeed)
	require.NoError(t, err)
	assert.Equal(t, 5, maxPos)
}

func TestCategoryRepository_GetMaxPosition_Empty(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	maxPos, err := repo.GetMaxPosition(ctx, "nonexistent-user", model.CategoryTypeFeed)
	require.NoError(t, err)
	assert.Equal(t, 0, maxPos)
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	cat := &model.Category{UserID: user.ID, Name: "Tech", Type: model.CategoryTypeFeed}
	cat.GenerateID()
	repo.Create(ctx, cat)

	found, err := repo.GetByID(ctx, user.ID, cat.ID)
	require.NoError(t, err)
	assert.Equal(t, "Tech", found.Name)
}

func TestCategoryRepository_GetByID_NotFound(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, user.ID, "nonexistent-id")
	assert.ErrorIs(t, err, service.ErrCategoryNotFound)
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	cat := &model.Category{UserID: user.ID, Name: "Tech", Type: model.CategoryTypeFeed}
	cat.GenerateID()
	repo.Create(ctx, cat)

	err := repo.Delete(ctx, user.ID, cat.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, user.ID, cat.ID)
	assert.ErrorIs(t, err, service.ErrCategoryNotFound)
}

func TestCategoryRepository_Update(t *testing.T) {
	db := setupCategoryDB(t)
	repo := NewCategoryRepository(db)
	user := createTestUserForCategory(t, db)
	ctx := context.Background()

	cat := &model.Category{UserID: user.ID, Name: "Tech", Type: model.CategoryTypeFeed, Position: 1}
	cat.GenerateID()
	repo.Create(ctx, cat)

	cat.Name = "Science"
	cat.Position = 10
	err := repo.Update(ctx, cat)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, user.ID, cat.ID)
	require.NoError(t, err)
	assert.Equal(t, "Science", found.Name)
	assert.Equal(t, 10, found.Position)
}
