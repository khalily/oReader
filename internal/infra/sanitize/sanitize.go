package sanitize

import (
	"regexp"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

// ugcpolicy is a policy that allows safe HTML for user-generated content
// This allows article content like paragraphs, links, images, lists, headings, etc.
// while removing dangerous elements like scripts, iframes, forms, etc.
var ugcpolicy *bluemonday.Policy

// strictpolicy removes ALL HTML - used for titles and descriptions
var strictpolicy *bluemonday.Policy

// regexpSafeRel is a simple pattern for safe rel attribute values
var regexpSafeRel = regexp.MustCompile(`(?i)^nofollow\s*(noopener\s*noreferrer?|noopener|noreferrer?)?$|^noopener\s*(noreferrer?|nofollow)?$|^noreferrer?$`)

func init() {
	// Create UGC policy for article content
	ugcpolicy = bluemonday.UGCPolicy()

	// Allow safe HTML elements
	ugcpolicy.AllowElements("p", "br", "hr")
	ugcpolicy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	ugcpolicy.AllowElements("strong", "b", "em", "i", "u", "s", "strike")
	ugcpolicy.AllowElements("ul", "ol", "li")
	ugcpolicy.AllowElements("blockquote", "pre", "code")
	ugcpolicy.AllowElements("table", "thead", "tbody", "tr", "th", "td")

	// Allow links with security attributes
	ugcpolicy.AllowAttrs("href").OnElements("a")
	ugcpolicy.AllowAttrs("rel").Matching(regexpSafeRel).OnElements("a")

	// Allow images with source and alt
	ugcpolicy.AllowAttrs("src").OnElements("img")
	ugcpolicy.AllowAttrs("alt").OnElements("img")
	ugcpolicy.AllowAttrs("width", "height").OnElements("img")

	// Create strict policy for titles/descriptions (removes all HTML)
	strictpolicy = bluemonday.StrictPolicy()
}

// SanitizeArticleContent sanitizes HTML content for article display.
// It allows safe HTML elements (p, a, img, ul, ol, li, h1-h6, strong, em, blockquote, pre, code)
// while removing dangerous elements (script, iframe, form, style, etc.) and attributes.
func SanitizeArticleContent(html string) string {
	return ugcpolicy.Sanitize(html)
}

// SanitizeFeedTitle sanitizes feed titles by removing ALL HTML.
// Titles should be plain text only.
func SanitizeFeedTitle(title string) string {
	return strictpolicy.Sanitize(title)
}

// StripHTML removes all HTML tags from a string, leaving only text content.
func StripHTML(html string) string {
	return strictpolicy.Sanitize(html)
}

// TruncateText truncates text to a maximum length, adding "..." if truncated.
// It respects UTF-8 character boundaries.
func TruncateText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// If text is short enough, return as-is
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}

	// Truncate to maxLen - 3 (for "...")
	runes := []rune(text)
	if len(runes) > maxLen-3 {
		return string(runes[:maxLen-3]) + "..."
	}
	return text
}

// GenerateDescription creates a plain text description from HTML content.
// It strips HTML tags and truncates to 200 characters.
func GenerateDescription(html string) string {
	stripped := StripHTML(html)
	return TruncateText(stripped, 200)
}
