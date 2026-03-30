package opml

import (
	"encoding/xml"
	"errors"
	"strings"
)

// ErrInvalidOPML is returned when the OPML file is invalid
var ErrInvalidOPML = errors.New("invalid OPML format")

// ParseResult contains the result of parsing an OPML file
type ParseResult struct {
	Feeds []*FeedInfo
}

// FeedInfo contains information about a feed from OPML
type FeedInfo struct {
	Title   string
	FeedURL string
	SiteURL string
}

// Parse parses an OPML file and extracts feed information
func Parse(content string) (*ParseResult, error) {
	var doc opmlDocument

	// Decode XML
	decoder := xml.NewDecoder(strings.NewReader(content))
	if err := decoder.Decode(&doc); err != nil {
		return nil, ErrInvalidOPML
	}

	// Validate structure - Body must exist
	// Empty outlines is valid (no feeds to import)
	outlines := doc.Body.Outlines
	if outlines == nil {
		// Initialize empty slice if nil
		outlines = []opmlOutline{}
	}

	result := &ParseResult{
		Feeds: extractFeeds(outlines),
	}

	return result, nil
}

// extractFeeds recursively extracts RSS feeds from outlines
func extractFeeds(outlines []opmlOutline) []*FeedInfo {
	var feeds []*FeedInfo

	for _, outline := range outlines {
		// Check if this is an RSS feed outline
		if outline.Type == "rss" && outline.XMLURL != "" {
			title := outline.Title
			if title == "" {
				title = outline.Text
			}
			if title == "" {
				title = outline.XMLURL
			}

			feeds = append(feeds, &FeedInfo{
				Title:   title,
				FeedURL: outline.XMLURL,
				SiteURL: outline.HTMLURL,
			})
		}

		// Recursively extract from nested outlines
		if len(outline.Outlines) > 0 {
			nestedFeeds := extractFeeds(outline.Outlines)
			feeds = append(feeds, nestedFeeds...)
		}
	}

	return feeds
}
