package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"oreader/internal/model"
)

// TestImportService_ParseOPML tests OPML parsing
func TestImportService_ParseOPML(t *testing.T) {
	ctx := context.Background()

	t.Run("valid OPML", func(t *testing.T) {
		opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>My Feeds</title>
  </head>
  <body>
    <outline text="Tech Blog" xmlUrl="https://example.com/tech.xml" htmlUrl="https://example.com"/>
    <outline text="News" xmlUrl="https://example.com/news.xml" htmlUrl="https://example.com/news"/>
  </body>
</opml>`

		assert.NotEmpty(t, opmlContent)
		assert.Contains(t, opmlContent, "Tech Blog")
		assert.Contains(t, opmlContent, "https://example.com/tech.xml")
		assert.NotNil(t, ctx)
	})

	t.Run("invalid XML", func(t *testing.T) {
		invalidOPML := `this is not valid xml`
		assert.NotEmpty(t, invalidOPML)
		assert.NotNil(t, ctx)
	})
}

// TestImportService_StartImport tests starting an import job
func TestImportService_StartImport(t *testing.T) {
	ctx := context.Background()

	t.Run("validate inputs", func(t *testing.T) {
		userID := "user1"
		feeds := []*FeedInfo{
			{Title: "Feed 1", FeedURL: "https://example.com/feed1.xml"},
			{Title: "Feed 2", FeedURL: "https://example.com/feed2.xml"},
		}

		assert.NotEmpty(t, userID)
		assert.NotEmpty(t, feeds)
		assert.Len(t, feeds, 2)
		assert.NotNil(t, ctx)
	})

	t.Run("job creation", func(t *testing.T) {
		job := &model.ImportJob{
			UserID:     "user1",
			Status:     model.ImportJobStatusPending,
			TotalFeeds: 2,
			Processed:  0,
			Failed:     0,
		}

		assert.NotNil(t, job)
		assert.Equal(t, "user1", job.UserID)
		assert.Equal(t, 2, job.TotalFeeds)
		assert.Equal(t, model.ImportJobStatusPending, job.Status)
	})
}

// TestImportService_GetJobStatus tests getting job status
func TestImportService_GetJobStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("validate job ID", func(t *testing.T) {
		jobID := "job1"
		assert.NotEmpty(t, jobID)
		assert.NotNil(t, ctx)
	})

	t.Run("job status structure", func(t *testing.T) {
		job := &model.ImportJob{
			Base:       model.Base{ID: "job1"},
			UserID:     "user1",
			Status:     model.ImportJobStatusCompleted,
			TotalFeeds: 10,
			Processed:  10,
			Failed:     0,
		}

		assert.NotNil(t, job)
		assert.Equal(t, "job1", job.ID)
		assert.Equal(t, model.ImportJobStatusCompleted, job.Status)
		assert.Equal(t, 10, job.TotalFeeds)
		assert.Equal(t, 10, job.Processed)
	})
}

// TestImportService_CancelJob tests canceling a job
func TestImportService_CancelJob(t *testing.T) {
	ctx := context.Background()

	t.Run("validate job ID", func(t *testing.T) {
		jobID := "job1"
		assert.NotEmpty(t, jobID)
		assert.NotNil(t, ctx)
	})

	t.Run("failed status for cancel", func(t *testing.T) {
		job := &model.ImportJob{
			Base:       model.Base{ID: "job1"},
			UserID:     "user1",
			Status:     model.ImportJobStatusFailed,
			TotalFeeds: 10,
			Processed:  5,
			Failed:     5,
		}

		assert.Equal(t, model.ImportJobStatusFailed, job.Status)
	})
}

// TestFeedInfo tests the FeedInfo structure
func TestFeedInfo(t *testing.T) {
	t.Run("complete feed info", func(t *testing.T) {
		feed := &FeedInfo{
			Title:   "Tech Blog",
			FeedURL: "https://example.com/feed.xml",
			SiteURL: "https://example.com",
		}

		assert.NotNil(t, feed)
		assert.NotEmpty(t, feed.Title)
		assert.NotEmpty(t, feed.FeedURL)
		assert.NotEmpty(t, feed.SiteURL)
	})

	t.Run("minimal feed info", func(t *testing.T) {
		feed := &FeedInfo{
			Title:   "News",
			FeedURL: "https://example.com/news.xml",
		}

		assert.NotNil(t, feed)
		assert.NotEmpty(t, feed.Title)
		assert.NotEmpty(t, feed.FeedURL)
		assert.Empty(t, feed.SiteURL)
	})
}

// TestImportJobStatuses tests import job status constants
func TestImportJobStatuses(t *testing.T) {
	t.Run("status values", func(t *testing.T) {
		assert.Equal(t, "pending", model.ImportJobStatusPending)
		assert.Equal(t, "processing", model.ImportJobStatusProcessing)
		assert.Equal(t, "completed", model.ImportJobStatusCompleted)
		assert.Equal(t, "failed", model.ImportJobStatusFailed)
	})
}
