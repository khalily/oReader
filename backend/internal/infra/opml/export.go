package opml

import (
	"bytes"
	"encoding/xml"
	"time"

	"github.com/khalily/oreader/internal/model"
)

// Export generates OPML 2.0 format XML from user's feeds
func Export(feeds []*model.Feed) ([]byte, error) {
	// Create outlines from feeds
	outlines := make([]opmlOutline, len(feeds))
	for i, feed := range feeds {
		outlines[i] = opmlOutline{
			Type:   "rss",
			Text:   feed.Title,
			Title:  feed.Title,
			XMLURL: feed.FeedURL,
		}
	}

	// Create OPML document
	doc := opmlDocument{
		Version: "2.0",
		Head: opmlHead{
			Title:       "oReader Subscriptions",
			DateCreated: time.Now().UTC().Format(time.RFC1123),
		},
		Body: opmlBody{
			Outlines: outlines,
		},
	}

	// Marshal to XML with proper formatting
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
