package repository

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oreader/internal/model"
)

// setupImportJobDB creates an in-memory database for testing
func setupImportJobDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate tables
	if err := db.AutoMigrate(
		&model.User{},
		&model.ImportJob{},
	); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

// TestCreate_WithLargePayload tests creating an import job with a large error payload
func TestCreate_WithLargePayload(t *testing.T) {
	db := setupImportJobDB(t)
	repo := NewImportJobRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a large error message (simulate complex OPML parse error)
	largeError := strings.Repeat("Failed to parse feed: https://example.com/very/long/path/to/feed.xml - ", 100)

	job := &model.ImportJob{
		UserID:     user.ID,
		Status:     model.ImportJobStatusFailed,
		TotalFeeds: 100,
		Processed:  50,
		Failed:     50,
		Error:      largeError,
	}
	if err := job.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	// Create should handle large payload
	err := repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("Create with large payload returned error: %v", err)
	}

	// Verify the job was created
	found, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if found == nil {
		t.Fatal("Job should be found")
	}
	if found.Error != largeError {
		t.Errorf("Error message not preserved correctly, got length %d, want %d",
			len(found.Error), len(largeError))
	}
}

// TestGetByID_NotFound tests GetByID when job doesn't exist
func TestGetByID_NotFound(t *testing.T) {
	db := setupImportJobDB(t)
	repo := NewImportJobRepository(db)

	ctx := context.Background()

	// Try to get a non-existent job
	job, err := repo.GetByID(ctx, "non-existent-id")
	if err != nil {
		t.Errorf("GetByID should not return error for not found, got: %v", err)
	}
	if job != nil {
		t.Error("GetByID should return nil for non-existent job")
	}
}

// TestGetByUserID_MultipleJobs tests retrieving multiple jobs for a user
func TestGetByUserID_MultipleJobs(t *testing.T) {
	db := setupImportJobDB(t)
	repo := NewImportJobRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create multiple jobs with different statuses
	statuses := []string{
		model.ImportJobStatusCompleted,
		model.ImportJobStatusFailed,
		model.ImportJobStatusPending,
	}

	for i, status := range statuses {
		job := &model.ImportJob{
			UserID:     user.ID,
			Status:     status,
			TotalFeeds: i + 1,
			Processed:  i,
			Failed:     0,
		}
		if err := job.GenerateID(); err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}
		if err := repo.Create(ctx, job); err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}
	}

	// Get all jobs for user
	jobs, err := repo.GetByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}

	// Jobs should be ordered by created_at DESC (newest first)
	// Since we created them in order, the last one should be first
	if len(jobs) > 0 && jobs[0].Status != model.ImportJobStatusPending {
		t.Errorf("First job should be pending (most recent), got %s", jobs[0].Status)
	}
}

// TestUpdate_StatusProgression tests updating job status through its lifecycle
func TestUpdate_StatusProgression(t *testing.T) {
	db := setupImportJobDB(t)
	repo := NewImportJobRepository(db)
	userRepo := NewUserRepository(db)

	ctx := context.Background()

	// Create test user
	user := &model.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := user.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create pending job
	job := &model.ImportJob{
		UserID:     user.ID,
		Status:     model.ImportJobStatusPending,
		TotalFeeds: 10,
		Processed:  0,
		Failed:     0,
	}
	if err := job.GenerateID(); err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}
	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Update to processing
	job.Status = model.ImportJobStatusProcessing
	job.Processed = 5
	if err := repo.Update(ctx, job); err != nil {
		t.Fatalf("Update to processing failed: %v", err)
	}

	found, _ := repo.GetByID(ctx, job.ID)
	if found.Status != model.ImportJobStatusProcessing {
		t.Errorf("Status = %s, want %s", found.Status, model.ImportJobStatusProcessing)
	}
	if found.Processed != 5 {
		t.Errorf("Processed = %d, want 5", found.Processed)
	}

	// Update to completed
	job.Status = model.ImportJobStatusCompleted
	job.Processed = 10
	if err := repo.Update(ctx, job); err != nil {
		t.Fatalf("Update to completed failed: %v", err)
	}

	found, _ = repo.GetByID(ctx, job.ID)
	if found.Status != model.ImportJobStatusCompleted {
		t.Errorf("Status = %s, want %s", found.Status, model.ImportJobStatusCompleted)
	}
}
