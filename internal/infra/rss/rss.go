package rss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"oreader/internal/infra/logger"
	"oreader/internal/infra/sanitize"
	"oreader/internal/infra/validation"
)

var (
	// ErrFeedFetchFailed is returned when the feed cannot be fetched
	ErrFeedFetchFailed = errors.New("failed to fetch feed")
	// ErrFeedParseFailed is returned when the feed cannot be parsed
	ErrFeedParseFailed = errors.New("failed to parse feed")
	// ErrFeedInvalidURL is returned when the URL is invalid
	ErrFeedInvalidURL = errors.New("invalid feed URL")
)

// FeedFetcher defines the interface for fetching feed content
type FeedFetcher interface {
	Fetch(url string) (string, error)
}

// HTTPFetcher implements FeedFetcher using HTTP client
type HTTPFetcher struct {
	client  *http.Client
	timeout time.Duration
}

// NewHTTPFetcher creates a new HTTP fetcher with configurable timeout
func NewHTTPFetcher(timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Fetch fetches the feed content from the given URL
func (f *HTTPFetcher) Fetch(url string) (string, error) {
	// Validate URL before fetching (SSRF protection)
	if err := validation.ValidateFeedURL(url); err != nil {
		logger.Error().
			Err(err).
			Str("feed_url", url).
			Msg("URL validation failed")
		return "", fmt.Errorf("%w: %v", ErrFeedInvalidURL, err)
	}

	logger.Debug().
		Str("feed_url", url).
		Dur("timeout", f.timeout).
		Msg("Fetching RSS feed")

	resp, err := f.client.Get(url)
	if err != nil {
		logger.Error().
			Err(err).
			Str("feed_url", url).
			Msg("HTTP request failed")
		return "", fmt.Errorf("%w: %v", ErrFeedFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error().
			Int("status_code", resp.StatusCode).
			Str("feed_url", url).
			Msg("Non-OK HTTP status")
		return "", fmt.Errorf("%w: HTTP %d", ErrFeedFetchFailed, resp.StatusCode)
	}

	// Limit response body size to 1MB
	limitedReader := io.LimitReader(resp.Body, 1*1024*1024)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		logger.Error().
			Err(err).
			Str("feed_url", url).
			Msg("Failed to read response body")
		return "", fmt.Errorf("%w: %v", ErrFeedFetchFailed, err)
	}

	logger.Debug().
		Str("feed_url", url).
		Int("bytes", len(content)).
		Msg("Successfully fetched RSS feed")

	return string(content), nil
}

// ParsedItem represents a parsed feed item
type ParsedItem struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PubDate     *time.Time
	Creator     string
}

// ParsedFeed represents a parsed feed
type ParsedFeed struct {
	Title       string
	Link        string
	Description string
	ImageURL    string
	Items       []*ParsedItem
}

// Parser wraps gofeed parser with additional functionality
type Parser struct {
	fetcher FeedFetcher
	parser  *gofeed.Parser
}

// NewParser creates a new RSS/Atom parser
func NewParser(fetcher FeedFetcher) *Parser {
	return &Parser{
		fetcher: fetcher,
		parser:  gofeed.NewParser(),
	}
}

// Parse fetches and parses a feed from the given URL
func (p *Parser) Parse(ctx context.Context, feedURL string) (*ParsedFeed, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Fetch feed content
	content, err := p.fetcher.Fetch(feedURL)
	if err != nil {
		return nil, err
	}

	// Parse the feed
	feed, err := p.parser.ParseString(content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFeedParseFailed, err)
	}

	// Convert to our ParsedFeed format
	parsedFeed := &ParsedFeed{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		ImageURL:    extractImageURL(feed),
		Items:       make([]*ParsedItem, 0, len(feed.Items)),
	}

	// Limit items to 1000
	maxItems := 1000
	itemCount := len(feed.Items)
	if itemCount > maxItems {
		itemCount = maxItems
	}

	for i := 0; i < itemCount; i++ {
		item := feed.Items[i]
		parsedItem := &ParsedItem{
			GUID:        getGUID(item),
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     getContent(item),
			PubDate:     parsePubDate(item.PublishedParsed),
			Creator:     getCreator(item),
		}
		parsedFeed.Items = append(parsedFeed.Items, parsedItem)
	}

	return parsedFeed, nil
}

// SanitizeFeed sanitizes all feed and item content
func SanitizeFeed(feed *ParsedFeed) {
	feed.Title = sanitize.SanitizeFeedTitle(feed.Title)
	feed.Description = sanitize.SanitizeFeedTitle(feed.Description)

	for _, item := range feed.Items {
		// 1. Fix relative image URLs before sanitization (using feed.Link as baseURL)
		if item.Content != "" && feed.Link != "" {
			item.Content = FixRelativeImageURLs(item.Content, feed.Link)
		}

		// 2. Generate description from content
		item.Description = sanitize.GenerateDescription(item.Content)

		// 3. Sanitize content
		item.Title = sanitize.SanitizeFeedTitle(item.Title)
		item.Content = sanitize.SanitizeArticleContent(item.Content)
	}
}

// extractImageURL extracts the image URL from a feed
func extractImageURL(feed *gofeed.Feed) string {
	if feed.Image != nil && feed.Image.URL != "" {
		return feed.Image.URL
	}
	if feed.ITunesExt != nil && feed.ITunesExt.Image != "" {
		return feed.ITunesExt.Image
	}
	return ""
}

// getGUID returns the GUID for an item, using link as fallback
func getGUID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}

// getContent returns the content of an item, preferring content:encoded
func getContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	return item.Description
}

// getCreator returns the author/creator of an item
func getCreator(item *gofeed.Item) string {
	if item.Author != nil && item.Author.Name != "" {
		return item.Author.Name
	}
	return ""
}

// parsePubDate parses a publication date
func parsePubDate(date *time.Time) *time.Time {
	if date != nil && !date.IsZero() {
		return date
	}
	return nil
}

// truncateContent truncates content to maxLen bytes
func truncateContent(content string, maxLen int64) string {
	if int64(len(content)) <= maxLen {
		return content
	}
	return content[:maxLen]
}

// TryGetFavicon attempts to get the favicon URL for a feed
func TryGetFavicon(feedURL string) string {
	// Parse the feed URL to get the base URL
	if !strings.HasPrefix(feedURL, "http://") && !strings.HasPrefix(feedURL, "https://") {
		return ""
	}

	// Simple favicon URL construction
	// In production, you might want to fetch and check if the favicon exists
	parts := strings.Split(feedURL, "/")
	if len(parts) >= 3 {
		scheme := parts[0]
		host := parts[2]
		return scheme + "//" + host + "/favicon.ico"
	}

	return ""
}
