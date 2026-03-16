package opml

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"oreader/internal/model"
)

// TestOPMLStructure tests that the generated OPML has the correct structure
func TestOPMLStructure(t *testing.T) {
	feeds := []*model.Feed{
		{
			Base:       model.Base{ID: "feed1"},
			FeedURL:    "https://example.com/feed.xml",
			Title:      "Example Feed",
			Description: "An example feed",
		},
		{
			Base:       model.Base{ID: "feed2"},
			FeedURL:    "https://blog.test/rss",
			Title:      "Test Blog",
			Description: "A test blog",
		},
	}

	output, err := Export(feeds)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Verify it's valid XML
	var opml OPML
	if err := xml.Unmarshal(output, &opml); err != nil {
		t.Fatalf("Generated OPML is not valid XML: %v", err)
	}

	// Check version
	if opml.Version != "2.0" {
		t.Errorf("Expected OPML version 2.0, got %s", opml.Version)
	}

	// Check head
	if opml.Head.Title != "oReader Subscriptions" {
		t.Errorf("Expected title 'oReader Subscriptions', got %s", opml.Head.Title)
	}
	// Check that dateCreated is present and non-empty (as string)
	if opml.Head.DateCreated == "" {
		t.Error("Expected dateCreated to be set")
	}

	// Check body outlines
	if len(opml.Body.Outlines) != 2 {
		t.Fatalf("Expected 2 outlines, got %d", len(opml.Body.Outlines))
	}

	// Check first outline
	outline1 := opml.Body.Outlines[0]
	if outline1.Type != "rss" {
		t.Errorf("Expected type 'rss', got %s", outline1.Type)
	}
	if outline1.Text != "Example Feed" {
		t.Errorf("Expected text 'Example Feed', got %s", outline1.Text)
	}
	if outline1.XMLURL != "https://example.com/feed.xml" {
		t.Errorf("Expected xmlUrl 'https://example.com/feed.xml', got %s", outline1.XMLURL)
	}
}

// TestEmptyExport tests that exporting with no feeds produces valid OPML
func TestEmptyExport(t *testing.T) {
	feeds := []*model.Feed{}

	output, err := Export(feeds)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Verify it's valid XML
	var opml OPML
	if err := xml.Unmarshal(output, &opml); err != nil {
		t.Fatalf("Generated OPML is not valid XML: %v", err)
	}

	// Check version and head exist
	if opml.Version != "2.0" {
		t.Errorf("Expected OPML version 2.0, got %s", opml.Version)
	}
	if opml.Head.Title != "oReader Subscriptions" {
		t.Errorf("Expected title 'oReader Subscriptions', got %s", opml.Head.Title)
	}

	// Body should have no outlines
	if len(opml.Body.Outlines) != 0 {
		t.Errorf("Expected 0 outlines, got %d", len(opml.Body.Outlines))
	}
}

// TestXMLDeclaration tests that the output starts with XML declaration
func TestXMLDeclaration(t *testing.T) {
	feeds := []*model.Feed{
		{
			Base:    model.Base{ID: "feed1"},
			FeedURL: "https://example.com/feed.xml",
			Title:   "Example Feed",
		},
	}

	output, err := Export(feeds)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	strOutput := string(output)
	if !strings.HasPrefix(strOutput, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Errorf("Expected XML declaration, got: %s", strOutput[:50])
	}
}

// TestHTMLEscaping tests that special characters are properly escaped
func TestHTMLEscaping(t *testing.T) {
	feeds := []*model.Feed{
		{
			Base:    model.Base{ID: "feed1"},
			FeedURL: "https://example.com/feed.xml",
			Title:   "Feed with <script> & \"quotes\"",
		},
	}

	output, err := Export(feeds)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Verify it's valid XML (no unescaped special characters)
	var opml OPML
	if err := xml.Unmarshal(output, &opml); err != nil {
		t.Fatalf("Generated OPML is not valid XML: %v", err)
	}

	if opml.Body.Outlines[0].Text != "Feed with <script> & \"quotes\"" {
		t.Errorf("Text not properly preserved: %s", opml.Body.Outlines[0].Text)
	}
}

// TestFeedWithHTMLURL tests that feeds with a website link include htmlUrl
func TestFeedWithHTMLURL(t *testing.T) {
	// This test will be enabled once we add Link field to Feed model
	// For now, we test that xmlUrl is always present
	feeds := []*model.Feed{
		{
			Base:    model.Base{ID: "feed1"},
			FeedURL: "https://example.com/feed.xml",
			Title:   "Example Feed",
		},
	}

	output, err := Export(feeds)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var opml OPML
	if err := xml.Unmarshal(output, &opml); err != nil {
		t.Fatalf("Generated OPML is not valid XML: %v", err)
	}

	if opml.Body.Outlines[0].XMLURL == "" {
		t.Error("Expected xmlUrl to be set")
	}
}

// TestOPML defines the expected OPML structure
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    OPMLHead `xml:"head"`
	Body    OPMLBody `xml:"body"`
}

// OPMLHead represents the head section of OPML
type OPMLHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated"`
}

// OPMLBody represents the body section of OPML
type OPMLBody struct {
	Outlines []OPMLOutline `xml:"outline"`
}

// OPMLOutline represents an outline element in OPML
type OPMLOutline struct {
	Type   string `xml:"type,attr"`
	Text   string `xml:"text,attr"`
	XMLURL string `xml:"xmlUrl,attr"`
	HTMLURL string `xml:"htmlUrl,attr"`
}

// BenchmarkExport benchmarks the export function
func BenchmarkExport(b *testing.B) {
	feeds := make([]*model.Feed, 100)
	for i := 0; i < 100; i++ {
		feeds[i] = &model.Feed{
			Base:    model.Base{ID: "feed" + string(rune(i))},
			FeedURL: "https://example.com/feed.xml",
			Title:   "Feed Title",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Export(feeds)
		if err != nil {
			b.Fatalf("Export() failed: %v", err)
		}
	}
}

// TestExportConsistentOutput tests that multiple exports produce consistent output
func TestExportConsistentOutput(t *testing.T) {
	feeds := []*model.Feed{
		{
			Base:    model.Base{ID: "feed1"},
			FeedURL: "https://example.com/feed.xml",
			Title:   "Example Feed",
		},
	}

	output1, err1 := Export(feeds)
	output2, err2 := Export(feeds)

	if err1 != nil || err2 != nil {
		t.Fatalf("Export() failed: %v, %v", err1, err2)
	}

	if !bytes.Equal(output1, output2) {
		// DateCreated will differ, so check just the structure
		var opml1, opml2 OPML
		xml.Unmarshal(output1, &opml1)
		xml.Unmarshal(output2, &opml2)

		if opml1.Body.Outlines[0].XMLURL != opml2.Body.Outlines[0].XMLURL {
			t.Error("Inconsistent feed URLs")
		}
		if opml1.Body.Outlines[0].Text != opml2.Body.Outlines[0].Text {
			t.Error("Inconsistent titles")
		}
	}
}
