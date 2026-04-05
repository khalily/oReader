package service

import (
	"context"
	"fmt"
	"time"

	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/infra/rss"
	"github.com/khalily/oreader/internal/model"
)

// feedService implements FeedService interface
type feedService struct {
	feedRepo       FeedRepository
	itemRepo       ItemRepository
	userFeedRepo   UserFeedRepository
	parser         *rss.Parser
}

// NewFeedService creates a new feed service
func NewFeedService(
	feedRepo FeedRepository,
	itemRepo ItemRepository,
	userFeedRepo UserFeedRepository,
	parser *rss.Parser,
) FeedService {
	return &feedService{
		feedRepo:     feedRepo,
		itemRepo:     itemRepo,
		userFeedRepo: userFeedRepo,
		parser:       parser,
	}
}

// Subscribe subscribes a user to a feed
func (s *feedService) Subscribe(ctx context.Context, userID, feedURL string) (*SubscribeResult, error) {
	logger.Info().
		Str("user_id", userID).
		Str("feed_url", feedURL).
		Msg("Subscribing to feed")

	// Check if user is already subscribed to this feed
	existingFeed, err := s.feedRepo.GetByURL(ctx, feedURL)
	if err == nil && existingFeed != nil {
		// Feed exists, check if user is subscribed
		userFeed, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, existingFeed.ID)
		if err == nil && userFeed != nil {
			return nil, ErrFeedAlreadySubscribed
		}
		// Subscribe to existing feed
		return s.subscribeToExistingFeed(ctx, userID, existingFeed)
	}

	// Fetch and parse the feed
	parsedFeed, err := s.parser.Parse(ctx, feedURL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", userID).
			Str("feed_url", feedURL).
			Msg("Failed to fetch feed")
		return nil, fmt.Errorf("%w: %v", ErrFeedFetchFailed, err)
	}

	// Sanitize feed content
	rss.SanitizeFeed(parsedFeed)

	// Create feed in database
	feed := &model.Feed{
		FeedURL:     feedURL,
		Title:       parsedFeed.Title,
		Description: parsedFeed.Description,
		ImageURL:    parsedFeed.ImageURL,
	}
	if err := feed.GenerateID(); err != nil {
		return nil, err
	}

	now := time.Now()
	feed.LastFetchedAt = &now
	feed.LastFetchStatus = "success"

	if err := s.feedRepo.Create(ctx, feed); err != nil {
		return nil, err
	}

	// Create items
	items := make([]*model.Item, 0, len(parsedFeed.Items))
	for _, parsedItem := range parsedFeed.Items {
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
			return nil, err
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.itemRepo.CreateBatch(ctx, items); err != nil {
			return nil, err
		}
	}

	// Create user-feed subscription
	maxPos, err := s.userFeedRepo.GetMaxPosition(ctx, userID)
	if err != nil {
		return nil, err
	}

	userFeed := &model.UserFeed{
		UserID:   userID,
		FeedID:   feed.ID,
		Position: maxPos + 1,
	}
	if err := userFeed.GenerateID(); err != nil {
		return nil, err
	}

	if err := s.userFeedRepo.Create(ctx, userFeed); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID).
		Str("feed_id", feed.ID).
		Int("new_item_count", len(items)).
		Msg("Feed subscription created")

	return &SubscribeResult{
		Feed:         feed,
		NewItemCount: len(items),
	}, nil
}

// subscribeToExistingFeed subscribes a user to an existing feed
func (s *feedService) subscribeToExistingFeed(ctx context.Context, userID string, feed *model.Feed) (*SubscribeResult, error) {
	// Get existing item count
	itemCount, err := s.itemRepo.CountByFeedID(ctx, feed.ID)
	if err != nil {
		return nil, err
	}

	// Create user-feed subscription
	maxPos, err := s.userFeedRepo.GetMaxPosition(ctx, userID)
	if err != nil {
		return nil, err
	}

	userFeed := &model.UserFeed{
		UserID:   userID,
		FeedID:   feed.ID,
		Position: maxPos + 1,
	}
	if err := userFeed.GenerateID(); err != nil {
		return nil, err
	}

	if err := s.userFeedRepo.Create(ctx, userFeed); err != nil {
		return nil, err
	}

	return &SubscribeResult{
		Feed:         feed,
		NewItemCount: int(itemCount),
	}, nil
}

// GetUserFeeds retrieves all feeds for a user with item counts
func (s *feedService) GetUserFeeds(ctx context.Context, userID string, opts ListOptions) ([]*FeedWithItemCount, int64, error) {
	feeds, total, err := s.feedRepo.ListByUserID(ctx, userID, opts)
	if err != nil {
		return nil, 0, err
	}

	// Build category_id mapping from user_feeds
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

	// Get item counts for each feed
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

// GetFeed retrieves a specific feed for a user
func (s *feedService) GetFeed(ctx context.Context, userID, feedID string) (*model.Feed, int, error) {
	// Check if user is subscribed to this feed
	_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		return nil, 0, ErrFeedNotFound
	}

	// Get feed
	feed, err := s.feedRepo.GetByID(ctx, feedID)
	if err != nil {
		return nil, 0, ErrFeedNotFound
	}

	// Get item count
	itemCount, err := s.itemRepo.CountByFeedID(ctx, feedID)
	if err != nil {
		return nil, 0, err
	}

	return feed, int(itemCount), nil
}

// DeleteFeed deletes a feed subscription for a user
func (s *feedService) DeleteFeed(ctx context.Context, userID, feedID string) error {
	logger.Debug().
		Str("user_id", userID).
		Str("feed_id", feedID).
		Msg("Deleting feed subscription")

	// Check if user is subscribed to this feed
	userFeed, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		logger.Debug().
			Err(err).
			Str("user_id", userID).
			Str("feed_id", feedID).
			Msg("Feed subscription not found")

		// Check if the record exists but is soft-deleted
		deletedUserFeed, deletedErr := s.userFeedRepo.GetByUserAndFeedIncludingDeleted(ctx, userID, feedID)
		if deletedErr == nil && deletedUserFeed != nil && deletedUserFeed.DeletedAt.Valid {
			logger.Debug().
				Str("user_id", userID).
				Str("feed_id", feedID).
				Msg("Feed already unsubscribed")
			return ErrFeedAlreadyUnsubscribed
		}

		return ErrFeedNotFound
	}

	// Delete the user-feed relationship
	if err := s.userFeedRepo.Delete(ctx, userID, feedID); err != nil {
		return err
	}
	_ = userFeed // Used for future reference

	logger.Info().
		Str("user_id", userID).
		Str("feed_id", feedID).
		Msg("Feed subscription deleted")

	// Check if any other users are subscribed to this feed
	// If not, delete the feed and its items
	// For now, we'll keep the feed for potential future subscriptions

	return nil
}

// RefreshFeed manually refreshes a feed
func (s *feedService) RefreshFeed(ctx context.Context, userID, feedID string) (*RefreshResult, error) {
	// Check if user is subscribed to this feed
	_, err := s.userFeedRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		return nil, ErrFeedNotFound
	}

	// Get feed
	feed, err := s.feedRepo.GetByID(ctx, feedID)
	if err != nil {
		return nil, ErrFeedNotFound
	}

	// Fetch and parse the feed
	parsedFeed, err := s.parser.Parse(ctx, feed.FeedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFeedFetchFailed, err)
	}

	// Sanitize feed content
	rss.SanitizeFeed(parsedFeed)

	// Update feed metadata
	feed.Title = parsedFeed.Title
	feed.Description = parsedFeed.Description
	feed.ImageURL = parsedFeed.ImageURL
	now := time.Now()
	feed.LastFetchedAt = &now
	feed.LastFetchStatus = "success"
	feed.ConsecutiveFailures = 0

	if err := s.feedRepo.Update(ctx, feed); err != nil {
		return nil, err
	}

	// Create new items or update existing ones with improved content
	newItemCount := 0
	for _, parsedItem := range parsedFeed.Items {
		// Check if item already exists
		existingItem, err := s.itemRepo.GetByGUID(ctx, feedID, parsedItem.GUID)
		if err == nil {
			// Item exists — update content if it differs (e.g. improved language preservation)
			if existingItem.Content != parsedItem.Content {
				existingItem.Content = parsedItem.Content
				existingItem.Description = parsedItem.Description
				_ = s.itemRepo.Update(ctx, existingItem)
			}
			continue
		}

		// Create new item
		item := &model.Item{
			FeedID:      feedID,
			GUID:        parsedItem.GUID,
			Title:       parsedItem.Title,
			Link:        parsedItem.Link,
			Description: parsedItem.Description,
			Content:     parsedItem.Content,
			PubDate:     parsedItem.PubDate,
			Creator:     parsedItem.Creator,
		}
		if err := item.GenerateID(); err != nil {
			return nil, err
		}

		if err := s.itemRepo.Create(ctx, item); err != nil {
			return nil, err
		}
		newItemCount++
	}

	return &RefreshResult{
		NewItemCount: newItemCount,
	}, nil
}
