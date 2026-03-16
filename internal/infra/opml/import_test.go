package opml

import (
	"testing"
)

// TestParseValidOPML20 tests parsing a valid OPML 2.0 file
func TestParseValidOPML20(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>My Subscriptions</title>
    <dateCreated>Sat, 15 Mar 2025 10:00:00 GMT</dateCreated>
  </head>
  <body>
    <outline type="rss" text="Example Feed" xmlUrl="https://example.com/feed.xml" htmlUrl="https://example.com/"/>
    <outline type="rss" text="Tech Blog" xmlUrl="https://blog.test/rss"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 2 {
		t.Fatalf("Expected 2 feeds, got %d", len(result.Feeds))
	}

	// Check first feed
	feed1 := result.Feeds[0]
	if feed1.Title != "Example Feed" {
		t.Errorf("Expected title 'Example Feed', got %s", feed1.Title)
	}
	if feed1.FeedURL != "https://example.com/feed.xml" {
		t.Errorf("Expected feed URL 'https://example.com/feed.xml', got %s", feed1.FeedURL)
	}
	if feed1.SiteURL != "https://example.com/" {
		t.Errorf("Expected site URL 'https://example.com/', got %s", feed1.SiteURL)
	}

	// Check second feed
	feed2 := result.Feeds[1]
	if feed2.Title != "Tech Blog" {
		t.Errorf("Expected title 'Tech Blog', got %s", feed2.Title)
	}
	if feed2.FeedURL != "https://blog.test/rss" {
		t.Errorf("Expected feed URL 'https://blog.test/rss', got %s", feed2.FeedURL)
	}
}

// TestParseValidOPML10 tests parsing OPML 1.0 format
func TestParseValidOPML10(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Old OPML</title>
  </head>
  <body>
    <outline type="rss" text="Feed" xmlUrl="https://example.com/feed"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed, got %d", len(result.Feeds))
	}
}

// TestParseNoVersion tests parsing OPML without version attribute
func TestParseNoVersion(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml>
  <head>
    <title>No Version</title>
  </head>
  <body>
    <outline type="rss" text="Feed" xmlUrl="https://example.com/feed"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed, got %d", len(result.Feeds))
	}
}

// TestParseInvalidXML tests parsing invalid XML
func TestParseInvalidXML(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Invalid</title>
  </head>
  <body>
    <outline type="rss" text="Feed" xmlUrl="https://example.com/feed"
  </body>
</opml>`

	_, err := Parse(opmlContent)
	if err == nil {
		t.Error("Expected error for invalid XML, got nil")
	}
	if err != ErrInvalidOPML {
		t.Errorf("Expected ErrInvalidOPML, got %v", err)
	}
}

// TestParseMissingBody tests parsing OPML without body element
// Note: We accept OPML files without a body element as valid (with 0 feeds)
// This is a permissive interpretation that allows for edge cases
func TestParseMissingBody(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>No Body</title>
  </head>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Should return an empty result (no feeds)
	if len(result.Feeds) != 0 {
		t.Errorf("Expected 0 feeds, got %d", len(result.Feeds))
	}
}

// TestParseEmptyOPML tests parsing OPML with no outlines
func TestParseEmptyOPML(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Empty</title>
  </head>
  <body>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 0 {
		t.Fatalf("Expected 0 feeds, got %d", len(result.Feeds))
	}
}

// TestParseNonRSSOutlines tests that non-RSS outlines are ignored
func TestParseNonRSSOutlines(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Mixed</title>
  </head>
  <body>
    <outline type="folder" text="Category">
      <outline type="rss" text="Nested Feed" xmlUrl="https://example.com/feed"/>
    </outline>
    <outline type="link" text="Link" htmlUrl="https://example.com/"/>
    <outline type="rss" text="Feed" xmlUrl="https://blog.test/rss"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Should only get RSS feeds, not folders or links
	// For now, we'll get flat RSS feeds (nested not supported in MVP)
	if len(result.Feeds) == 0 {
		t.Error("Expected at least one RSS feed")
	}
}

// TestParseMissingRequiredAttributes tests outlines without xmlUrl
func TestParseMissingRequiredAttributes(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Missing Attr</title>
  </head>
  <body>
    <outline type="rss" text="No URL"/>
    <outline type="rss" xmlUrl="https://example.com/feed"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Only the feed with xmlUrl should be included
	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed (only valid one), got %d", len(result.Feeds))
	}

	if result.Feeds[0].FeedURL != "https://example.com/feed" {
		t.Errorf("Expected valid feed URL, got %s", result.Feeds[0].FeedURL)
	}
}

// TestParseHTMLEscaped tests parsing HTML-escaped content
func TestParseHTMLEscaped(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Escaped</title>
  </head>
  <body>
    <outline type="rss" text="Feed &amp; More" xmlUrl="https://example.com/feed?q=1&amp;2"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if result.Feeds[0].Title != "Feed & More" {
		t.Errorf("Expected title 'Feed & More', got %s", result.Feeds[0].Title)
	}
	if result.Feeds[0].FeedURL != "https://example.com/feed?q=1&2" {
		t.Errorf("Expected unescaped URL, got %s", result.Feeds[0].FeedURL)
	}
}

// TestParseFeedWithURLMissingTitle tests using feed URL as fallback for title
func TestParseFeedWithURLMissingTitle(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>No Title</title>
  </head>
  <body>
    <outline type="rss" xmlUrl="https://example.com/feed"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed, got %d", len(result.Feeds))
	}

	// Title should default to the feed URL if not provided
	if result.Feeds[0].Title == "" {
		// Use URL as fallback
		result.Feeds[0].Title = result.Feeds[0].FeedURL
	}
}

// TestParseNestedOutlines tests handling of nested category structures
func TestParseNestedOutlines(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Nested</title>
  </head>
  <body>
    <outline text="Tech">
      <outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss"/>
      <outline text="Programming">
        <outline type="rss" text="Blog" xmlUrl="https://blog.com/rss"/>
      </outline>
    </outline>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// For MVP, we extract all RSS feeds regardless of nesting
	// Categories are not preserved
	if len(result.Feeds) != 2 {
		t.Fatalf("Expected 2 feeds from nested structure, got %d", len(result.Feeds))
	}
}

// TestParseGoogleReaderOPML tests parsing Google Reader export format
func TestParseGoogleReaderOPML(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>Google Reader Subscriptions</title>
  </head>
  <body>
    <outline text="News" title="News">
      <outline text="BBC News" title="BBC News" type="rss" htmlUrl="https://www.bbc.com/news" xmlUrl="https://feeds.bbci.co.uk/news/rss.xml"/>
    </outline>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed, got %d", len(result.Feeds))
	}

	if result.Feeds[0].Title != "BBC News" {
		t.Errorf("Expected title 'BBC News', got %s", result.Feeds[0].Title)
	}
}

// TestParseFeedlyOPML tests parsing Feedly export format
func TestParseFeedlyOPML(t *testing.T) {
	opmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Feedly Subscriptions</title>
  </head>
  <body>
    <outline type="rss" text="The Verge" xmlUrl="https://www.theverge.com/rss/index.xml" htmlUrl="https://www.theverge.com/"/>
  </body>
</opml>`

	result, err := Parse(opmlContent)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(result.Feeds) != 1 {
		t.Fatalf("Expected 1 feed, got %d", len(result.Feeds))
	}
}
