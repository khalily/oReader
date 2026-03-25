package rss

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"oreader/internal/infra/logger"
)

// FixRelativeImageURLs fixes relative image URLs in HTML content by converting them to absolute URLs.
// This ensures images from RSS feeds display correctly regardless of their original path format.
func FixRelativeImageURLs(htmlContent string, baseURL string) string {
	if htmlContent == "" || baseURL == "" {
		return htmlContent
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		logger.Debug().Err(err).Str("base_url", baseURL).Msg("Failed to parse base URL")
		return htmlContent
	}

	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to parse HTML for image URL fixing")
		return htmlContent
	}

	// Track if any changes were made
	modified := false

	// Find all images and fix relative paths
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		// Skip absolute URLs and data URIs
		if strings.HasPrefix(src, "http://") ||
			strings.HasPrefix(src, "https://") ||
			strings.HasPrefix(src, "data:") {
			return
		}

		// Parse relative URL
		relURL, err := url.Parse(src)
		if err != nil {
			return
		}

		// Resolve to absolute URL
		absURL := base.ResolveReference(relURL)
		s.SetAttr("src", absURL.String())
		modified = true
	})

	// If no changes, return original content
	if !modified {
		return htmlContent
	}

	// Return modified HTML
	html, err := doc.Html()
	if err != nil {
		return htmlContent
	}
	return html
}
