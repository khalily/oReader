package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"oreader/internal/model"
	"oreader/internal/service"
	"oreader/internal/testutil"
)

func setupPaperDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.SetupTestDB(t)

	err := db.AutoMigrate(
		&model.User{},
		&model.Paper{},
		&model.PaperTag{},
		&model.PaperCollection{},
		&model.PaperCollectionItem{},
	)
	require.NoError(t, err)

	return db
}

func createTestUserForPaper(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, NewUserRepository(db).Create(context.Background(), user))
	return user
}

func TestPaperRepository_CreateAndGet(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	paper := &model.Paper{
		UserID:           user.ID,
		Title:            "Test Paper",
		OriginalFilename: "test.pdf",
		PDFPath:          "uploads/papers/test.pdf",
		PDFSize:          1024,
		Status:           model.PaperStatusPending,
	}
	require.NoError(t, paper.GenerateID())

	err := repo.Create(ctx, paper)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "Test Paper", found.Title)
	assert.Equal(t, model.PaperStatusPending, found.Status)
}

func TestPaperRepository_GetByID_NotFound(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaperRepository_ListByUserID(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	// Create 3 papers
	for i := 0; i < 3; i++ {
		paper := &model.Paper{
			UserID:           user.ID,
			Title:            "Paper " + string(rune('A'+i)),
			OriginalFilename: "test.pdf",
			PDFPath:          "uploads/papers/test.pdf",
			Status:           model.PaperStatusCompleted,
		}
		require.NoError(t, paper.GenerateID())
		require.NoError(t, repo.Create(ctx, paper))
	}

	papers, total, err := repo.ListByUserID(ctx, user.ID, service.PaperListOptions{
		Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, papers, 3)
}

func TestPaperRepository_ListByUserID_WithSearch(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	// Create papers with different titles
	for _, title := range []string{"Attention Is All You Need", "BERT: Pre-training of Deep Bidirectional Transformers"} {
		paper := &model.Paper{
			UserID:           user.ID,
			Title:            title,
			OriginalFilename: "test.pdf",
			PDFPath:          "uploads/papers/test.pdf",
			Status:           model.PaperStatusCompleted,
		}
		require.NoError(t, paper.GenerateID())
		require.NoError(t, repo.Create(ctx, paper))
	}

	papers, _, err := repo.ListByUserID(ctx, user.ID, service.PaperListOptions{
		Limit: 10, Query: "Attention",
	})
	require.NoError(t, err)
	assert.Len(t, papers, 1)
	assert.Equal(t, "Attention Is All You Need", papers[0].Title)
}

func TestPaperRepository_Update(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	paper := &model.Paper{
		UserID: user.ID,
		Status: model.PaperStatusPending,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	paper.Title = "Updated Title"
	paper.Status = model.PaperStatusCompleted
	paper.MarkdownContent = "# Hello\n\nWorld"
	require.NoError(t, repo.Update(ctx, paper))

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", found.Title)
	assert.Equal(t, model.PaperStatusCompleted, found.Status)
}

func TestPaperRepository_Delete(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	paper := &model.Paper{UserID: user.ID, Status: model.PaperStatusPending}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	require.NoError(t, repo.Delete(ctx, paper.ID))

	found, err := repo.GetByID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaperRepository_ListTags(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	tagRepo := NewPaperTagRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	paper := &model.Paper{UserID: user.ID, Status: model.PaperStatusCompleted}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	require.NoError(t, tagRepo.SetTags(ctx, paper.ID, []string{"deep learning", "transformer"}))

	tags, err := repo.ListTags(ctx, user.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"deep learning", "transformer"}, tags)
}

func TestPaperTagRepository_SetTags(t *testing.T) {
	db := setupPaperDB(t)
	repo := NewPaperRepository(db)
	tagRepo := NewPaperTagRepository(db)
	user := createTestUserForPaper(t, db)
	ctx := context.Background()

	paper := &model.Paper{UserID: user.ID, Status: model.PaperStatusCompleted}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, repo.Create(ctx, paper))

	// Set initial tags
	require.NoError(t, tagRepo.SetTags(ctx, paper.ID, []string{"nlp", "bert"}))

	tags, err := tagRepo.GetByPaperID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 2)

	// Replace tags
	require.NoError(t, tagRepo.SetTags(ctx, paper.ID, []string{"cv", "resnet", "classification"}))

	tags, err = tagRepo.GetByPaperID(ctx, paper.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 3)

	tagValues := make([]string, len(tags))
	for i, tag := range tags {
		tagValues[i] = tag.Tag
	}
	assert.ElementsMatch(t, []string{"cv", "resnet", "classification"}, tagValues)
}
