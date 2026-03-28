package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"oreader/internal/infra/rss"
	"oreader/internal/model"
)

const (
	// MaxConcurrentFeeds is the maximum number of feeds to refresh concurrently
	MaxConcurrentFeeds = 10
	// DefaultFeedTimeout is the timeout for individual feed fetches
	DefaultFeedTimeout = 30 * time.Second
)

// refreshWorkerService implements RefreshWorkerService interface
type refreshWorkerService struct {
	feedRepo     FeedRepository
	itemRepo     ItemRepository
	userFeedRepo UserFeedRepository
	parser       *rss.Parser
}

// NewRefreshWorkerService creates a new refresh worker service
func NewRefreshWorkerService(
	feedRepo FeedRepository,
	itemRepo ItemRepository,
	userFeedRepo UserFeedRepository,
) RefreshWorkerService {
	// Create a default HTTP fetcher with 30 second timeout
	fetcher := rss.NewHTTPFetcher(DefaultFeedTimeout)
	parser := rss.NewParser(fetcher)

	return &refreshWorkerService{
		feedRepo:     feedRepo,
		itemRepo:     itemRepo,
		userFeedRepo: userFeedRepo,
		parser:       parser,
	}
}

// RefreshAllFeeds refreshes all feeds concurrently with bounded semaphore
func (s *refreshWorkerService) RefreshAllFeeds(ctx context.Context) (*RefreshAllResult, error) {
	startTime := time.Now()

	log.Info().Msg("Starting refresh of all feeds")

	// Get all feeds
	feeds, err := s.feedRepo.ListAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list feeds for refresh")
		return nil, fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Info().Msg("No feeds to refresh")
		return &RefreshAllResult{
			TotalFeeds: 0,
			Duration:   time.Since(startTime),
		}, nil
	}

	log.Info().Int("count", len(feeds)).Msg("Refreshing feeds")

	// Create a bounded semaphore to limit concurrent refreshes
	sem := make(chan struct{}, MaxConcurrentFeeds)
	var wg sync.WaitGroup

	// Channel to collect results
	results := make(chan SingleFeedResult, len(feeds))

	// Refresh each feed concurrently
	for _, feed := range feeds {
		wg.Add(1)
		go func(f *model.Feed) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create context with timeout for this feed
			feedCtx, cancel := context.WithTimeout(ctx, DefaultFeedTimeout)
			defer cancel()

			// Refresh the feed
			result := s.refreshSingleFeedWithContext(feedCtx, f)
			results <- result
		}(feed)
	}

	// Wait for all refreshes to complete
	wg.Wait()
	close(results)

	// Collect results
	var successCount, failureCount, totalNewItems int
	var failedFeedIDs []string
	var allResults []SingleFeedResult

	for result := range results {
		allResults = append(allResults, result)
		if result.Success {
			successCount++
			totalNewItems += result.NewItems
		} else {
			failureCount++
			failedFeedIDs = append(failedFeedIDs, result.FeedID)
		}
	}

	duration := time.Since(startTime)

	log.Info().
		Int("total", len(feeds)).
		Int("success", successCount).
		Int("failure", failureCount).
		Int("new_items", totalNewItems).
		Dur("duration", duration).
		Msg("Refresh completed")

	// Count total items across all feeds
	totalItems := s.countTotalItems(ctx, feeds)

	return &RefreshAllResult{
		TotalFeeds:    len(feeds),
		SuccessCount:  successCount,
		FailureCount:  failureCount,
		TotalItems:    totalItems,
		NewItems:      totalNewItems,
		Duration:      duration,
		FailedFeedIDs: failedFeedIDs,
		Results:       allResults,
	}, nil
}

// RefreshSingleFeed refreshes a single feed
func (s *refreshWorkerService) RefreshSingleFeed(ctx context.Context, feed *model.Feed) (SingleFeedResult, error) {
	return s.refreshSingleFeedWithContext(ctx, feed), nil
}

// refreshSingleFeedWithContext refreshes a single feed with the given context
func (s *refreshWorkerService) refreshSingleFeedWithContext(ctx context.Context, feed *model.Feed) SingleFeedResult {
	startTime := time.Now()

	log.Info().
		Str("feed_id", feed.ID).
		Str("feed_url", feed.FeedURL).
		Msg("Refreshing feed")

	// Parse the feed
	parsedFeed, err := s.parser.Parse(ctx, feed.FeedURL)
	if err != nil {
		// Update feed with error status
		s.updateFeedErrorStatus(context.Background(), feed, err)

		log.Error().
			Err(err).
			Str("feed_id", feed.ID).
			Str("feed_url", feed.FeedURL).
			Dur("duration", time.Since(startTime)).
			Msg("Failed to refresh feed")

		return SingleFeedResult{
			FeedID:   feed.ID,
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}
	}

	// Sanitize feed content
	rss.SanitizeFeed(parsedFeed)

	// Update feed metadata
	now := time.Now()
	feed.Title = parsedFeed.Title
	feed.Description = parsedFeed.Description
	feed.ImageURL = parsedFeed.ImageURL
	feed.LastFetchedAt = &now
	feed.LastFetchStatus = "success"
	feed.ConsecutiveFailures = 0

	if updateErr := s.feedRepo.Update(context.Background(), feed); updateErr != nil {
		log.Error().
			Err(updateErr).
			Str("feed_id", feed.ID).
			Msg("Failed to update feed metadata")
	}

	// Create new items or update existing ones with improved content (deduplicated by GUID)
	newItemCount := 0
	for _, parsedItem := range parsedFeed.Items {
		// Check if item already exists
		existingItem, err := s.itemRepo.GetByGUID(context.Background(), feed.ID, parsedItem.GUID)
		if err == nil {
			// Item exists — update content if it differs (e.g. improved language preservation)
			if existingItem.Content != parsedItem.Content {
				existingItem.Content = parsedItem.Content
				existingItem.Description = parsedItem.Description
				_ = s.itemRepo.Update(context.Background(), existingItem)
			}
			continue
		}

		// Create new item
		item := &model.Item{
			FeedID:      feed.ID,
			GUID:        parsedItem.GUID,
			Title:       parsedItem.Title,
			Link:        parsedItem.Link,
			Description: parsedItem.Description,
			Content:     parsedItem.Content,
			PubDate:     parsedItem.PubDate,
			Creator:     parsedItem.Creator,
		}
		if err := item.GenerateID(); err != nil {
			log.Error().
				Err(err).
				Str("feed_id", feed.ID).
				Msg("Failed to generate item ID")
			continue
		}

		if err := s.itemRepo.Create(context.Background(), item); err != nil {
			log.Error().
				Err(err).
				Str("feed_id", feed.ID).
				Str("item_guid", parsedItem.GUID).
				Msg("Failed to create item")
			continue
		}
		newItemCount++
	}

	duration := time.Since(startTime)

	log.Info().
		Str("feed_id", feed.ID).
		Str("feed_url", feed.FeedURL).
		Int("new_items", newItemCount).
		Dur("duration", duration).
		Msg("Feed refreshed successfully")

	return SingleFeedResult{
		FeedID:   feed.ID,
		Success:  true,
		NewItems: newItemCount,
		Duration: duration,
	}
}

// updateFeedErrorStatus updates the feed with error status after a failed refresh
func (s *refreshWorkerService) updateFeedErrorStatus(ctx context.Context, feed *model.Feed, err error) {
	now := time.Now()
	feed.LastFetchedAt = &now
	feed.ConsecutiveFailures++
	feed.LastFetchStatus = "error"

	// Check if it was a timeout
	if errors.Is(err, context.DeadlineExceeded) {
		feed.LastFetchStatus = "timeout"
	}

	if updateErr := s.feedRepo.Update(ctx, feed); updateErr != nil {
		log.Error().
			Err(updateErr).
			Str("feed_id", feed.ID).
			Msg("Failed to update feed error status")
	}
}

// countTotalItems counts the total number of items across all feeds
func (s *refreshWorkerService) countTotalItems(ctx context.Context, feeds []*model.Feed) int {
	total := 0
	for _, feed := range feeds {
		count, err := s.itemRepo.CountByFeedID(ctx, feed.ID)
		if err != nil {
			log.Error().
				Err(err).
				Str("feed_id", feed.ID).
				Msg("Failed to count items for feed")
			continue
		}
		total += int(count)
	}
	return total
}
