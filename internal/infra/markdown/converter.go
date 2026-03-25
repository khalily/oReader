// Package markdown provides HTML to Markdown conversion
package markdown

import (
	"github.com/JohannesKaufmann/html-to-markdown"
)

// Converter wraps the html-to-markdown converter
type Converter struct {
	conv *md.Converter
}

// NewConverter creates a new Converter instance with configured options
func NewConverter() *Converter {
	conv := md.NewConverter("", true, &md.Options{
		HeadingStyle:     "atx",     // Use # style headings
		HorizontalRule:   "---",     // Horizontal rule style
		BulletListMarker: "-",       // Unordered list marker
		CodeBlockStyle:   "fenced",  // Use ``` code blocks
		Fence:            "```",     // Use backticks for code blocks
		EmDelimiter:      "*",       // Italic delimiter
		StrongDelimiter:  "**",      // Bold delimiter
		LinkStyle:        "inlined", // Inline links
	})

	return &Converter{conv: conv}
}

// Convert converts HTML content to Markdown format
func (c *Converter) Convert(html string) (string, error) {
	if html == "" {
		return "", nil
	}
	return c.conv.ConvertString(html)
}
