package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/infra/grpc"
	"oreader/internal/model"
)

// mockPaperConverterClient is a mock for the gRPC client
type mockPaperConverterClient struct {
	mock.Mock
}

func (m *mockPaperConverterClient) Convert(ctx context.Context, pdfContent []byte, filename string) ([]grpc.ConvertProgress, error) {
	args := m.Called(ctx, pdfContent, filename)
	if results := args.Get(0); results != nil {
		return results.([]grpc.ConvertProgress), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPaperConverterClient) ExtractMetadata(ctx context.Context, markdown string) (*grpc.PaperMetadata, error) {
	args := m.Called(ctx, markdown)
	if results := args.Get(0); results != nil {
		return results.(*grpc.PaperMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPaperConverterClient) Close() error {
	return nil
}

// gormPaperRepo is a thin wrapper around gorm for testing (avoids import cycle with repository package)
type gormPaperRepo struct {
	db *gorm.DB
}

func (r *gormPaperRepo) Create(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Create(paper).Error
}

func (r *gormPaperRepo) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&paper).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &paper, nil
}

func (r *gormPaperRepo) ListByUserID(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error) {
	var papers []*model.Paper
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Paper{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	if err := query.Order("created_at DESC").Find(&papers).Error; err != nil {
		return nil, 0, err
	}
	return papers, total, nil
}

func (r *gormPaperRepo) Update(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Save(paper).Error
}

func (r *gormPaperRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("paper_id = ?", id).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Paper{}, "id = ?", id).Error
	})
}

func (r *gormPaperRepo) ListTags(ctx context.Context, userID string) ([]string, error) {
	var tags []string
	err := r.db.WithContext(ctx).
		Model(&model.PaperTag{}).
		Distinct("tag").
		Joins("JOIN papers ON papers.id = paper_tags.paper_id AND papers.deleted_at IS NULL").
		Where("papers.user_id = ?", userID).
		Pluck("tag", &tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// gormPaperTagRepo wraps gorm for PaperTagRepository in tests
type gormPaperTagRepo struct {
	db *gorm.DB
}

func (r *gormPaperTagRepo) SetTags(ctx context.Context, paperID string, tags []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("paper_id = ?", paperID).Delete(&model.PaperTag{}).Error; err != nil {
			return err
		}
		for _, tag := range tags {
			paperTag := &model.PaperTag{PaperID: paperID, Tag: tag}
			if err := paperTag.GenerateID(); err != nil {
				return err
			}
			if err := tx.Create(paperTag).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *gormPaperTagRepo) GetByPaperID(ctx context.Context, paperID string) ([]*model.PaperTag, error) {
	var tags []*model.PaperTag
	err := r.db.WithContext(ctx).Where("paper_id = ?", paperID).Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func setupPaperServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use file-based SQLite with shared cache for cross-goroutine access
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Paper{},
		&model.PaperTag{},
	)
	require.NoError(t, err)

	return db
}

func createPaperUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	user := &model.User{Email: "paper-test@example.com", PasswordHash: "hash"}
	require.NoError(t, user.GenerateID())
	require.NoError(t, db.Create(user).Error)
	return user.ID
}

func TestPaperService_UploadPaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	tmpDir := t.TempDir()

	mockClient := new(mockPaperConverterClient)
	mockClient.On("Convert", mock.Anything, mock.Anything, "test.pdf").
		Return([]grpc.ConvertProgress{
			{Status: "mining", Progress: 50},
			{Status: "completed", Progress: 100, Markdown: "# Test Paper\n\nContent here"},
		}, nil)
	mockClient.On("ExtractMetadata", mock.Anything, "# Test Paper\n\nContent here").
		Return(&grpc.PaperMetadata{
			Title:         "Test Paper",
			Authors:       []string{"Alice", "Bob"},
			Abstract:      "This is a test paper abstract",
			Keywords:      []string{"machine learning"},
			PublishedYear: "2024",
			DOI:           "10.1234/test",
		}, nil)

	paperRepo := &gormPaperRepo{db: db}
	tagRepo := &gormPaperTagRepo{db: db}
	svc := NewPaperService(paperRepo, tagRepo, mockClient, tmpDir)

	paper, err := svc.UploadPaper(context.Background(), userID, "test.pdf", []byte("fake-pdf-content"))
	require.NoError(t, err)
	assert.Contains(t, paper.PDFPath, "test.pdf")

	// Wait for async conversion to complete
	var updated *model.Paper
	require.Eventually(t, func() bool {
		updated, _ = paperRepo.GetByID(context.Background(), paper.ID)
		return updated != nil && updated.Status == model.PaperStatusCompleted
	}, 2*time.Second, 50*time.Millisecond, "Paper conversion should complete")
	assert.Equal(t, "Test Paper", updated.Title)
}

func TestPaperService_UploadPaper_GRPCFailed(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	tmpDir := t.TempDir()

	mockClient := new(mockPaperConverterClient)
	mockClient.On("Convert", mock.Anything, mock.Anything, "test.pdf").
		Return(nil, assert.AnError)

	paperRepo := &gormPaperRepo{db: db}
	tagRepo := &gormPaperTagRepo{db: db}
	svc := NewPaperService(paperRepo, tagRepo, mockClient, tmpDir)

	paper, err := svc.UploadPaper(context.Background(), userID, "test.pdf", []byte("fake-pdf-content"))
	require.NoError(t, err)

	// Wait for async conversion to fail
	var updated *model.Paper
	require.Eventually(t, func() bool {
		updated, _ = paperRepo.GetByID(context.Background(), paper.ID)
		return updated != nil && updated.Status == model.PaperStatusFailed
	}, 2*time.Second, 50*time.Millisecond, "Paper conversion should fail")
	assert.Contains(t, updated.Error, "assert.AnError")
}

func TestPaperService_GetPaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{
		UserID: userID,
		Title:  "My Paper",
		Status: model.PaperStatusCompleted,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := &gormPaperRepo{db: db}
	svc := NewPaperService(paperRepo, nil, nil, t.TempDir())

	found, err := svc.GetPaper(context.Background(), userID, paper.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Paper", found.Title)
}

func TestPaperService_GetPaper_NotOwner(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{
		UserID: userID,
		Status: model.PaperStatusCompleted,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := &gormPaperRepo{db: db}
	svc := NewPaperService(paperRepo, nil, nil, t.TempDir())

	_, err := svc.GetPaper(context.Background(), "other-user-id", paper.ID)
	require.Error(t, err)
	assert.Equal(t, ErrPaperNotFound, err)
}

func TestPaperService_DeletePaper(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	// Create paper with PDF file
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, userID, "test.pdf")
	require.NoError(t, os.MkdirAll(filepath.Dir(pdfPath), 0755))
	require.NoError(t, os.WriteFile(pdfPath, []byte("fake-pdf"), 0644))

	paper := &model.Paper{
		UserID:  userID,
		Status:  model.PaperStatusCompleted,
		PDFPath: pdfPath,
	}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := &gormPaperRepo{db: db}
	svc := NewPaperService(paperRepo, nil, nil, tmpDir)

	err := svc.DeletePaper(context.Background(), userID, paper.ID)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(pdfPath)
	assert.True(t, os.IsNotExist(err))
}

func TestPaperService_UpdateTags(t *testing.T) {
	db := setupPaperServiceDB(t)
	userID := createPaperUser(t, db)

	paper := &model.Paper{UserID: userID, Status: model.PaperStatusCompleted}
	require.NoError(t, paper.GenerateID())
	require.NoError(t, db.Create(paper).Error)

	paperRepo := &gormPaperRepo{db: db}
	tagRepo := &gormPaperTagRepo{db: db}
	svc := NewPaperService(paperRepo, tagRepo, nil, t.TempDir())

	err := svc.UpdateTags(context.Background(), userID, paper.ID, []string{"AI", "ML"})
	require.NoError(t, err)

	tags, err := tagRepo.GetByPaperID(context.Background(), paper.ID)
	require.NoError(t, err)
	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Tag
	}
	assert.ElementsMatch(t, []string{"AI", "ML"}, tagNames)
}
