package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/khalily/oreader/internal/model"
)

// --- Mock Repositories ---

// mockCategoryRepository is a mock implementation of CategoryRepository for testing
type mockCategoryRepository struct {
	createFunc      func(ctx context.Context, category *model.Category) error
	getByIDFunc     func(ctx context.Context, userID, id string) (*model.Category, error)
	listByUserIDFunc func(ctx context.Context, userID string, categoryType string) ([]*model.Category, error)
	updateFunc      func(ctx context.Context, category *model.Category) error
	deleteFunc      func(ctx context.Context, userID, id string) error
	getMaxPositionFunc func(ctx context.Context, userID string, categoryType string) (int, error)
}

func (m *mockCategoryRepository) Create(ctx context.Context, category *model.Category) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, category)
	}
	return nil
}

func (m *mockCategoryRepository) GetByID(ctx context.Context, userID, id string) (*model.Category, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, id)
	}
	return nil, ErrCategoryNotFound
}

func (m *mockCategoryRepository) ListByUserID(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
	if m.listByUserIDFunc != nil {
		return m.listByUserIDFunc(ctx, userID, categoryType)
	}
	return nil, nil
}

func (m *mockCategoryRepository) Update(ctx context.Context, category *model.Category) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, category)
	}
	return nil
}

func (m *mockCategoryRepository) Delete(ctx context.Context, userID, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID, id)
	}
	return nil
}

func (m *mockCategoryRepository) GetMaxPosition(ctx context.Context, userID string, categoryType string) (int, error) {
	if m.getMaxPositionFunc != nil {
		return m.getMaxPositionFunc(ctx, userID, categoryType)
	}
	return 0, nil
}

// mockUserFeedRepositoryForCategory is a mock implementation of UserFeedRepository for category tests
type mockUserFeedRepositoryForCategory struct {
	getByUserAndFeedFunc func(ctx context.Context, userID, feedID string) (*model.UserFeed, error)
	updateCategoryFunc   func(ctx context.Context, userID, feedID string, categoryID *string) error
}

func (m *mockUserFeedRepositoryForCategory) Create(ctx context.Context, userFeed *model.UserFeed) error {
	return nil
}

func (m *mockUserFeedRepositoryForCategory) GetByUserAndFeed(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	if m.getByUserAndFeedFunc != nil {
		return m.getByUserAndFeedFunc(ctx, userID, feedID)
	}
	return nil, ErrFeedNotFound
}

func (m *mockUserFeedRepositoryForCategory) GetByUserAndFeedIncludingDeleted(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
	return nil, ErrFeedNotFound
}

func (m *mockUserFeedRepositoryForCategory) ListByUserID(ctx context.Context, userID string) ([]*model.UserFeed, error) {
	return nil, nil
}

func (m *mockUserFeedRepositoryForCategory) Delete(ctx context.Context, userID, feedID string) error {
	return nil
}

func (m *mockUserFeedRepositoryForCategory) GetMaxPosition(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (m *mockUserFeedRepositoryForCategory) UpdateCategory(ctx context.Context, userID, feedID string, categoryID *string) error {
	if m.updateCategoryFunc != nil {
		return m.updateCategoryFunc(ctx, userID, feedID, categoryID)
	}
	return nil
}

// mockPaperRepositoryForCategory is a mock implementation of PaperRepository for category tests
type mockPaperRepositoryForCategory struct {
	getByIDFunc         func(ctx context.Context, id string) (*model.Paper, error)
	updateCategoryFunc  func(ctx context.Context, userID, paperID string, categoryID *string) error
}

func (m *mockPaperRepositoryForCategory) Create(ctx context.Context, paper *model.Paper) error {
	return nil
}

func (m *mockPaperRepositoryForCategory) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockPaperRepositoryForCategory) ListByUserID(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error) {
	return nil, 0, nil
}

func (m *mockPaperRepositoryForCategory) Update(ctx context.Context, paper *model.Paper) error {
	return nil
}

func (m *mockPaperRepositoryForCategory) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockPaperRepositoryForCategory) ListTags(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func (m *mockPaperRepositoryForCategory) UpdateCategory(ctx context.Context, userID, paperID string, categoryID *string) error {
	if m.updateCategoryFunc != nil {
		return m.updateCategoryFunc(ctx, userID, paperID, categoryID)
	}
	return nil
}

// --- Tests ---

func TestCreateCategory(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("success - creates feed category", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getMaxPositionFunc: func(ctx context.Context, userID string, categoryType string) (int, error) {
				assert.Equal(t, "feed", categoryType)
				return 2, nil
			},
			createFunc: func(ctx context.Context, category *model.Category) error {
				assert.Equal(t, "Tech", category.Name)
				assert.Equal(t, "feed", category.Type)
				assert.Equal(t, userID, category.UserID)
				assert.Equal(t, 3, category.Position) // maxPos + 1
				return nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		result, err := svc.CreateCategory(ctx, userID, "Tech", "feed")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Tech", result.Name)
		assert.Equal(t, "feed", result.Type)
		assert.Equal(t, 3, result.Position)
	})

	t.Run("duplicate name error", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getMaxPositionFunc: func(ctx context.Context, userID string, categoryType string) (int, error) {
				return 0, nil
			},
			createFunc: func(ctx context.Context, category *model.Category) error {
				// Simulate duplicate key error from DB
				return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		_, err := svc.CreateCategory(ctx, userID, "Tech", "feed")

		require.Error(t, err)
		assert.Equal(t, ErrCategoryDuplicate, err)
	})

	t.Run("repository error propagates", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			createFunc: func(ctx context.Context, category *model.Category) error {
				return errors.New("db connection lost")
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		_, err := svc.CreateCategory(ctx, userID, "Tech", "feed")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "db connection lost")
	})
}

func TestListCategories(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("success - lists feed categories", func(t *testing.T) {
		expected := []*model.Category{
			{Base: model.Base{ID: "cat-1"}, UserID: userID, Name: "Tech", Type: "feed", Position: 0},
			{Base: model.Base{ID: "cat-2"}, UserID: userID, Name: "News", Type: "feed", Position: 1},
		}

		catRepo := &mockCategoryRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
				assert.Equal(t, userID, userID)
				assert.Equal(t, "feed", categoryType)
				return expected, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		result, err := svc.ListCategories(ctx, userID, "feed")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "cat-1", result[0].ID)
		assert.Equal(t, "cat-2", result[1].ID)
	})

	t.Run("success - lists all categories when type is empty", func(t *testing.T) {
		expected := []*model.Category{
			{Base: model.Base{ID: "cat-1"}, Name: "Tech", Type: "feed"},
			{Base: model.Base{ID: "cat-2"}, Name: "ML", Type: "paper"},
		}

		catRepo := &mockCategoryRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
				assert.Equal(t, "", categoryType)
				return expected, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		result, err := svc.ListCategories(ctx, userID, "")

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("repository error propagates", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			listByUserIDFunc: func(ctx context.Context, userID string, categoryType string) ([]*model.Category, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		_, err := svc.ListCategories(ctx, userID, "feed")

		require.Error(t, err)
	})
}

func TestRenameCategory(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	categoryID := "cat-1"

	t.Run("success", func(t *testing.T) {
		existing := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, category *model.Category) error {
				assert.Equal(t, "AI", category.Name)
				return nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		result, err := svc.RenameCategory(ctx, userID, categoryID, "AI")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "AI", result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return nil, ErrCategoryNotFound
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		_, err := svc.RenameCategory(ctx, userID, categoryID, "NewName")

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})

	t.Run("duplicate name error on update", func(t *testing.T) {
		existing := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, category *model.Category) error {
				return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		_, err := svc.RenameCategory(ctx, userID, categoryID, "DuplicateName")

		require.Error(t, err)
		assert.Equal(t, ErrCategoryDuplicate, err)
	})
}

func TestDeleteCategory(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	categoryID := "cat-1"

	t.Run("success", func(t *testing.T) {
		existing := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return existing, nil
			},
			deleteFunc: func(ctx context.Context, userID, id string) error {
				assert.Equal(t, userID, userID)
				assert.Equal(t, categoryID, id)
				return nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		err := svc.DeleteCategory(ctx, userID, categoryID)

		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return nil, ErrCategoryNotFound
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, &mockPaperRepositoryForCategory{})
		err := svc.DeleteCategory(ctx, userID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})
}

func TestMoveFeedToCategory(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	feedID := "feed-1"
	categoryID := "cat-1"

	t.Run("success - assign feed to category", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return &model.UserFeed{UserID: userID, FeedID: feedID}, nil
			},
			updateCategoryFunc: func(ctx context.Context, userID, feedID string, catID *string) error {
				assert.Equal(t, categoryID, *catID)
				return nil
			},
		}

		svc := NewCategoryService(catRepo, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, categoryID)

		require.NoError(t, err)
	})

	t.Run("error - category not found", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return nil, ErrCategoryNotFound
			},
		}

		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return &model.UserFeed{UserID: userID, FeedID: feedID}, nil
			},
		}

		svc := NewCategoryService(catRepo, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})

	t.Run("error - wrong category type", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "ML",
			Type:   "paper", // wrong type for feeds
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return &model.UserFeed{UserID: userID, FeedID: feedID}, nil
			},
		}

		svc := NewCategoryService(catRepo, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})

	t.Run("error - feed not found", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return nil, ErrFeedNotFound
			},
		}

		svc := NewCategoryService(catRepo, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrFeedNotFound, err)
	})

	t.Run("success - uncategorize feed (empty category_id)", func(t *testing.T) {
		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return &model.UserFeed{UserID: userID, FeedID: feedID}, nil
			},
			updateCategoryFunc: func(ctx context.Context, userID, feedID string, catID *string) error {
				assert.Nil(t, catID)
				return nil
			},
		}

		svc := NewCategoryService(&mockCategoryRepository{}, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, "")

		require.NoError(t, err)
	})

	t.Run("error - uncategorize feed not found", func(t *testing.T) {
		feedRepo := &mockUserFeedRepositoryForCategory{
			getByUserAndFeedFunc: func(ctx context.Context, userID, feedID string) (*model.UserFeed, error) {
				return nil, ErrFeedNotFound
			},
		}

		svc := NewCategoryService(&mockCategoryRepository{}, feedRepo, &mockPaperRepositoryForCategory{})
		err := svc.MoveFeedToCategory(ctx, userID, feedID, "")

		require.Error(t, err)
		assert.Equal(t, ErrFeedNotFound, err)
	})
}

func TestMovePaperToCategory(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	paperID := "paper-1"
	categoryID := "cat-1"

	t.Run("success - assign paper to category", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "ML",
			Type:   "paper",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return &model.Paper{Base: model.Base{ID: paperID}, UserID: userID}, nil
			},
			updateCategoryFunc: func(ctx context.Context, uid, pid string, catID *string) error {
				assert.Equal(t, categoryID, *catID)
				return nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, categoryID)

		require.NoError(t, err)
	})

	t.Run("error - category not found", func(t *testing.T) {
		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return nil, ErrCategoryNotFound
			},
		}

		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return &model.Paper{Base: model.Base{ID: paperID}, UserID: userID}, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})

	t.Run("error - wrong category type", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "Tech",
			Type:   "feed", // wrong type for papers
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return &model.Paper{Base: model.Base{ID: paperID}, UserID: userID}, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrCategoryNotFound, err)
	})

	t.Run("error - paper not found", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "ML",
			Type:   "paper",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return nil, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrPaperNotFound, err)
	})

	t.Run("error - paper belongs to another user", func(t *testing.T) {
		category := &model.Category{
			Base: model.Base{ID: categoryID},
			UserID: userID,
			Name:   "ML",
			Type:   "paper",
		}

		catRepo := &mockCategoryRepository{
			getByIDFunc: func(ctx context.Context, userID, id string) (*model.Category, error) {
				return category, nil
			},
		}

		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return &model.Paper{Base: model.Base{ID: paperID}, UserID: "other-user"}, nil
			},
		}

		svc := NewCategoryService(catRepo, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, categoryID)

		require.Error(t, err)
		assert.Equal(t, ErrPaperNotFound, err)
	})

	t.Run("success - uncategorize paper (empty category_id)", func(t *testing.T) {
		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return &model.Paper{Base: model.Base{ID: paperID}, UserID: userID}, nil
			},
			updateCategoryFunc: func(ctx context.Context, uid, pid string, catID *string) error {
				assert.Nil(t, catID)
				return nil
			},
		}

		svc := NewCategoryService(&mockCategoryRepository{}, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, "")

		require.NoError(t, err)
	})

	t.Run("error - uncategorize paper not found", func(t *testing.T) {
		paperRepo := &mockPaperRepositoryForCategory{
			getByIDFunc: func(ctx context.Context, id string) (*model.Paper, error) {
				return nil, nil
			},
		}

		svc := NewCategoryService(&mockCategoryRepository{}, &mockUserFeedRepositoryForCategory{}, paperRepo)
		err := svc.MovePaperToCategory(ctx, userID, paperID, "")

		require.Error(t, err)
		assert.Equal(t, ErrPaperNotFound, err)
	})
}
