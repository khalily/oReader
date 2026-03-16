package rss

import (
	"context"
	"testing"
	"time"
)

// Mock fetcher for testing
type mockFetcher struct {
	feedContent string
	err         error
}

func (m *mockFetcher) Fetch(url string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.feedContent, nil
}

func TestParseFeed_ValidRSS2(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <lastBuildDate>Mon, 16 Mar 2026 10:00:00 GMT</lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>https://example.com/item1</link>
      <description>Description 1</description>
      <pubDate>Mon, 16 Mar 2026 09:00:00 GMT</pubDate>
      <author>author@example.com</author>
      <guid>https://example.com/item1</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser(&mockFetcher{feedContent: rssContent})
	feed, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if feed.Title != "Test Feed" {
		t.Errorf("Title = %q, want %q", feed.Title, "Test Feed")
	}
	if feed.Link != "https://example.com" {
		t.Errorf("Link = %q, want %q", feed.Link, "https://example.com")
	}
	if feed.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", feed.Description, "Test Description")
	}
	if len(feed.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(feed.Items))
	}

	item := feed.Items[0]
	if item.Title != "Item 1" {
		t.Errorf("Item Title = %q, want %q", item.Title, "Item 1")
	}
	if item.GUID != "https://example.com/item1" {
		t.Errorf("Item GUID = %q, want %q", item.GUID, "https://example.com/item1")
	}
}

func TestParseFeed_ValidAtom(t *testing.T) {
	atomContent := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <subtitle>Test Subtitle</subtitle>
  <updated>2026-03-16T10:00:00Z</updated>
  <entry>
    <title>Entry 1</title>
    <link href="https://example.com/entry1"/>
    <summary>Summary 1</summary>
    <content>Content 1</content>
    <published>2026-03-16T09:00:00Z</published>
    <updated>2026-03-16T09:00:00Z</updated>
    <author>
      <name>Author Name</name>
    </author>
    <id>https://example.com/entry1</id>
  </entry>
</feed>`

	parser := NewParser(&mockFetcher{feedContent: atomContent})
	feed, err := parser.Parse(context.Background(), "https://example.com/feed.atom")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if feed.Title != "Test Atom Feed" {
		t.Errorf("Title = %q, want %q", feed.Title, "Test Atom Feed")
	}
	if len(feed.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(feed.Items))
	}

	item := feed.Items[0]
	if item.Title != "Entry 1" {
		t.Errorf("Item Title = %q, want %q", item.Title, "Entry 1")
	}
}

func TestParseFeed_MissingOptionalFields(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Minimal Feed</title>
    <item>
      <title>Item without optional fields</title>
      <link>https://example.com/item</link>
    </item>
  </channel>
</rss>`

	parser := NewParser(&mockFetcher{feedContent: rssContent})
	feed, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if feed.Description != "" {
		t.Errorf("Description should be empty for missing field, got %q", feed.Description)
	}

	item := feed.Items[0]
	if item.Description != "" {
		t.Errorf("Item Description should be empty for missing field, got %q", item.Description)
	}
}

func TestParseFeed_InvalidXML(t *testing.T) {
	// gofeed is lenient and handles many XML errors
	// Test with truly unparseable content
	invalidXML := `this is not xml at all`

	parser := NewParser(&mockFetcher{feedContent: invalidXML})
	_, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err == nil {
		t.Error("Parse() should return error for invalid XML")
	}
}

func TestParseFeed_ItemWithoutGUID(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed without GUID</title>
    <item>
      <title>Item without GUID</title>
      <link>https://example.com/item1</link>
    </item>
  </channel>
</rss>`

	parser := NewParser(&mockFetcher{feedContent: rssContent})
	feed, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if len(feed.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(feed.Items))
	}

	// Item should use link as GUID when GUID is missing
	item := feed.Items[0]
	if item.GUID != "https://example.com/item1" {
		t.Errorf("Item GUID should be link when GUID is missing, got %q", item.GUID)
	}
}

func TestParseFeed_EmbeddedContent(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Feed with content:encoded</title>
    <item>
      <title>Item with Content</title>
      <link>https://example.com/item</link>
      <description>Short description</description>
      <content:encoded><![CDATA[<p>Full HTML content here</p>]]></content:encoded>
      <guid>https://example.com/item</guid>
    </item>
  </channel>
</rss>`

	parser := NewParser(&mockFetcher{feedContent: rssContent})
	feed, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	item := feed.Items[0]
	if item.Content != "<p>Full HTML content here</p>" {
		t.Errorf("Content = %q, want %q", item.Content, "<p>Full HTML content here</p>")
	}
}

func TestParseFeed_ContextCancellation(t *testing.T) {
	rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
  </channel>
</rss>`

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	parser := NewParser(&mockFetcher{feedContent: rssContent})
	_, err := parser.Parse(ctx, "https://example.com/feed.xml")
	if err == nil {
		t.Error("Parse() should return error when context is cancelled")
	}
}

func TestParseFeed_FetchError(t *testing.T) {
	parser := NewParser(&mockFetcher{err: ErrFeedFetchFailed})
	_, err := parser.Parse(context.Background(), "https://example.com/feed.xml")
	if err != ErrFeedFetchFailed {
		t.Errorf("Parse() should return fetch error, got %v", err)
	}
}

func TestParseItem_PubDateParsing(t *testing.T) {
	testCases := []struct {
		name    string
		dateStr string
		wantNil bool
	}{
		{"Valid RFC1123", "Mon, 16 Mar 2026 10:00:00 GMT", false},
		{"Valid ISO8601", "2026-03-16T10:00:00Z", false},
		{"Valid ISO8601 with offset", "2026-03-16T10:00:00+00:00", false},
		{"Invalid date", "Invalid Date", true},
		{"Empty string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pubDate *time.Time
			if !tc.wantNil {
				// Parse the date string properly for valid dates
				if parsed, err := time.Parse(time.RFC1123, tc.dateStr); err == nil {
					pubDate = &parsed
				} else if parsed, err := time.Parse(time.RFC3339, tc.dateStr); err == nil {
					pubDate = &parsed
				}
			}

			result := parsePubDate(pubDate)
			if (result == nil) != tc.wantNil {
				t.Errorf("parsePubDate() = %v, wantNil=%v", result, tc.wantNil)
			}
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int64
		expected string
	}{
		{
			name:     "short content",
			input:    "Short content",
			maxLen:   1000,
			expected: "Short content",
		},
		{
			name:     "exact length",
			input:    "12345",
			maxLen:   5,
			expected: "12345",
		},
		{
			name:     "truncate",
			input:    string(make([]byte, 2000)),
			maxLen:   1000,
			expected: string(make([]byte, 1000)),
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   1000,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateContent(tc.input, tc.maxLen)
			if len(result) > int(tc.maxLen) {
				t.Errorf("truncateContent() result length %d exceeds maxLen %d", len(result), tc.maxLen)
			}
		})
	}
}

func TestSanitizeFeed(t *testing.T) {
	feed := &ParsedFeed{
		Title:       "<script>alert('xss')</script>Feed Title",
		Description: "<img src=x onerror=alert('xss')>Description",
		Items: []*ParsedItem{
			{
				Title:       "<b>Item Title</b>",
				Description: "<p>Item <script>evil</script> Description</p>",
				Content:     "<div>Safe <strong>content</strong></div><script>alert('xss')</script>",
			},
		},
	}

	SanitizeFeed(feed)

	if feed.Title != "Feed Title" {
		t.Errorf("Title after sanitization = %q, want %q", feed.Title, "Feed Title")
	}

	if feed.Description != "Description" {
		t.Errorf("Description after sanitization = %q, want %q", feed.Description, "Description")
	}

	item := feed.Items[0]
	if item.Title != "Item Title" {
		t.Errorf("Item Title after sanitization = %q, want %q", item.Title, "Item Title")
	}

	if item.Description != "Safe content" {
		t.Errorf("Item Description after sanitization = %q, want %q", item.Description, "Safe content")
	}

	// Content should have safe HTML but not scripts
	if containsScript(item.Content) {
		t.Errorf("Item Content after sanitization still contains script")
	}
}

func containsScript(s string) bool {
	return len(s) > 0 && (containsSub(s, "<script") || containsSub(s, "onerror="))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFetchWithTimeout(t *testing.T) {
	// Test that timeout works
	longContent := string(make([]byte, 10*1024*1024)) // 10MB

	parser := NewParser(&mockFetcher{feedContent: longContent})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := parser.Parse(ctx, "https://example.com/feed.xml")
	duration := time.Since(start)

	// Should complete quickly (not parse 10MB)
	if duration > 100*time.Millisecond {
		t.Errorf("Fetch should timeout quickly, took %v", duration)
	}

	// May error due to timeout or may succeed if parsing is fast
	// We're mainly testing that it doesn't hang
	_ = err
}
