package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"oreader/internal/infra/logger"
	"oreader/internal/infra/opml"
	"oreader/internal/model"
)

// importService implements ImportService interface
type importService struct {
	feedService     FeedService
	importJobRepo   ImportJobRepository
	userFeedRepo    UserFeedRepository
	feedRepo        FeedRepository
	runningJobs     map[string]context.CancelFunc
	runningJobsMu   sync.RWMutex
}

// NewImportService creates a new import service
func NewImportService(
	feedService FeedService,
	importJobRepo ImportJobRepository,
	userFeedRepo UserFeedRepository,
	feedRepo FeedRepository,
) ImportService {
	return &importService{
		feedService:   feedService,
		importJobRepo: importJobRepo,
		userFeedRepo:  userFeedRepo,
		feedRepo:      feedRepo,
		runningJobs:   make(map[string]context.CancelFunc),
	}
}

// ParseOPML parses an OPML file and returns feed information
func (s *importService) ParseOPML(ctx context.Context, content string) ([]*FeedInfo, error) {
	result, err := opml.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	// Convert opml.FeedInfo to service.FeedInfo
	feeds := make([]*FeedInfo, len(result.Feeds))
	for i, f := range result.Feeds {
		feeds[i] = &FeedInfo{
			Title:   f.Title,
			FeedURL: f.FeedURL,
			SiteURL: f.SiteURL,
		}
	}

	return feeds, nil
}

// StartImport starts an async import job for the user
func (s *importService) StartImport(ctx context.Context, userID string, feeds []*FeedInfo) (*model.ImportJob, error) {
	logger.Info().
		Str("user_id", userID).
		Int("total_feeds", len(feeds)).
		Msg("Starting import job")

	// Create import job
	job := &model.ImportJob{
		UserID:     userID,
		Status:     model.ImportJobStatusPending,
		TotalFeeds: len(feeds),
		Processed:  0,
		Failed:     0,
	}
	if err := job.GenerateID(); err != nil {
		return nil, err
	}

	// Save job to database
	if err := s.importJobRepo.Create(ctx, job); err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to create import job")
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	logger.Info().
		Str("user_id", userID).
		Str("job_id", job.ID).
		Int("total_feeds", job.TotalFeeds).
		Msg("Import job created")

	// Start async processing
	go s.processImportAsync(context.Background(), job.ID, userID, feeds)

	return job, nil
}

// GetJobStatus retrieves the status of an import job
func (s *importService) GetJobStatus(ctx context.Context, jobID string) (*model.ImportJob, error) {
	job, err := s.importJobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}
	if job == nil {
		return nil, fmt.Errorf("job not found")
	}
	return job, nil
}

// ProcessImport processes an import job (called by worker)
func (s *importService) ProcessImport(ctx context.Context, jobID string) error {
	job, err := s.importJobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Parse OPML content - but we need the feeds list
	// This method is designed to be called by a worker that has the job details
	// For now, we'll use the async method directly
	return fmt.Errorf("use StartImport for new imports")
}

// processImportAsync processes an import job asynchronously
func (s *importService) processImportAsync(ctx context.Context, jobID, userID string, feeds []*FeedInfo) {
	logger.Info().
		Str("job_id", jobID).
		Str("user_id", userID).
		Int("total_feeds", len(feeds)).
		Msg("Processing import job")

	// Create cancelable context for this job
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Track the running job
	s.runningJobsMu.Lock()
	s.runningJobs[jobID] = cancel
	s.runningJobsMu.Unlock()

	defer func() {
		s.runningJobsMu.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMu.Unlock()
	}()

	// Update job status to processing
	job, err := s.importJobRepo.GetByID(jobCtx, jobID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("job_id", jobID).
			Msg("Failed to get import job")
		return
	}

	now := time.Now()
	job.Status = model.ImportJobStatusProcessing
	job.StartedAt = &now
	if err := s.importJobRepo.Update(jobCtx, job); err != nil {
		logger.Error().
			Err(err).
			Str("job_id", jobID).
			Msg("Failed to update job status to processing")
		return
	}

	// Process each feed
	processed := 0
	failed := 0

	for _, feedInfo := range feeds {
		// Check if context is cancelled
		select {
		case <-jobCtx.Done():
			logger.Info().
				Str("job_id", jobID).
				Msg("Import job cancelled")
			return
		default:
		}

		// Subscribe to the feed
		_, err := s.feedService.Subscribe(jobCtx, userID, feedInfo.FeedURL)
		processed++

		if err != nil {
			// Check if it's already subscribed (not an error for import)
			if err != ErrFeedAlreadySubscribed {
				failed++
				logger.Warn().
					Err(err).
					Str("job_id", jobID).
					Str("feed_url", feedInfo.FeedURL).
					Msg("Failed to import feed")
			}
		}

		// Update progress every 10 feeds or on the last feed
		if processed%10 == 0 || processed == len(feeds) {
			job, _ = s.importJobRepo.GetByID(jobCtx, jobID)
			if job != nil {
				job.Processed = processed
				job.Failed = failed
				s.importJobRepo.Update(jobCtx, job)
			}
		}
	}

	// Mark job as completed
	job, _ = s.importJobRepo.GetByID(jobCtx, jobID)
	if job != nil {
		job.Status = model.ImportJobStatusCompleted
		job.Processed = processed
		job.Failed = failed
		endTime := time.Now()
		job.EndedAt = &endTime
		s.importJobRepo.Update(jobCtx, job)

		logger.Info().
			Str("job_id", jobID).
			Int("processed", processed).
			Int("failed", failed).
			Msg("Import job completed")
	}
}

// CancelJob cancels a running import job
func (s *importService) CancelJob(jobID string) {
	s.runningJobsMu.RLock()
	cancel, exists := s.runningJobs[jobID]
	s.runningJobsMu.RUnlock()

	if exists {
		cancel()
	}
}
