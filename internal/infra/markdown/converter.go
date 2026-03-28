// Package markdown provides HTML to Markdown conversion
package markdown

import (
	"regexp"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
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

	// Enable table plugin for proper Markdown table conversion
	conv.Use(plugin.Table())

	// BeforeHook: propagate language-xxx from ancestor elements to <code>.
	// Many RSS sources (Jekyll/Rouge) put the language class on a wrapper <div>
	// or <pre> instead of on <code> itself. The html-to-markdown library only
	// reads the class from <code>, so we must copy it down before conversion.
	conv.Before(propagateCodeLanguage)

	// Add custom rule to convert Pandoc LaTeX delimiters to standard math syntax
	// \(...\) -> $...$ (inline math)
	// \[...\] -> $$...$$ (block math)
	conv.Keep("$", "$") // Ensure $ is not escaped

	return &Converter{conv: conv}
}

// languageClassRegex extracts a language name from a class attribute containing "language-xxx".
// Matches both standalone "language-xxx" and combined forms like "language-xxx highlighter-rouge".
var languageClassRegex = regexp.MustCompile(`\blanguage-([a-z0-9+_]+(?:-[a-z0-9+_]+)*)\b`)

// extractLanguageFromClass returns the language identifier from a class string,
// e.g. "language-python" -> "python". Returns empty string if not found.
func extractLanguageFromClass(class string) string {
	m := languageClassRegex.FindStringSubmatch(class)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// propagateCodeLanguage is a BeforeHook that copies language-xxx classes from
// ancestor elements down to <code> elements inside <pre> blocks.
//
// Many RSS sources (Jekyll/Rouge blogs, Hugo-generated sites) place the language
// identifier on a wrapper <div> like <div class="language-python highlighter-rouge">
// while the <code> element itself has no class. Since html-to-markdown only reads
// the class from <code>, the language is lost without this hook.
func propagateCodeLanguage(doc *goquery.Selection) {
	doc.Find("pre code").Each(func(_ int, code *goquery.Selection) {
		// If <code> already has a language-xxx class, nothing to do
		if existing := extractLanguageFromClass(code.AttrOr("class", "")); existing != "" {
			return
		}

		// Walk up the DOM tree looking for language-xxx on ancestors
		// Stop at <pre> (immediate parent) and go up to wrapper divs
		parent := code.Parents()
		for i := 0; i < parent.Length(); i++ {
			node := parent.Eq(i)
			if lang := extractLanguageFromClass(node.AttrOr("class", "")); lang != "" {
				existingClass := code.AttrOr("class", "")
				newClass := "language-" + lang
				if existingClass != "" {
					newClass = existingClass + " " + newClass
				}
				code.SetAttr("class", newClass)
				return
			}
		}
	})
}

// latexInlineRegex matches Pandoc inline math delimiters \(...\)
// Pattern: backslash(s) followed by open paren, content, backslash(s) and close paren
// Handles 1-4 backslashes to account for html-to-markdown escaping (tables have extra escaping)
// Note: Uses non-greedy .+? to allow content to contain parentheses like \ln(a)
var latexInlineRegex = regexp.MustCompile(`(?s)\\{1,4}\((.+?)\\{1,4}\)`)

// latexBlockRegex matches Pandoc block math delimiters \[...\]
// Pattern: backslash(s) followed by open bracket, content, backslash(s) and close bracket
// Handles 1-4 backslashes to account for html-to-markdown escaping (tables have extra escaping)
var latexBlockRegex = regexp.MustCompile(`(?s)\\{1,4}\[([\s\S]+?)\\{1,4}\]`)

// quadrupleBackslashRegex matches four consecutive backslashes (from table escaping)
// In regex: \\ matches one literal backslash, so \\\\\\\\ matches four
var quadrupleBackslashRegex = regexp.MustCompile(`\\\\\\\\`)

// codeWithBackticksRegex matches Markdown code spans that contain backticks
// Pattern: `` `content` `` (double backticks wrapping content with single backticks)
// This happens when Jekyll/Rouge generates <code>`content`</code>
var codeWithBackticksRegex = regexp.MustCompile("`` `([^`]+)` ``")

// doubleBackslashRegex matches two consecutive backslashes
// In regex: \\\\ matches two literal backslashes
var doubleBackslashRegex = regexp.MustCompile(`\\\\`)

// convertLaTeXDelimiters converts Pandoc LaTeX delimiters to standard math syntax
// and cleans up extra backticks in code spans from Jekyll/Rouge syntax highlighting
func convertLaTeXDelimiters(content string) string {
	// First, clean up code spans with extra backticks: `` `content` `` -> `content`
	// This happens when Jekyll/Rouge generates <code>`content`</code>
	content = codeWithBackticksRegex.ReplaceAllString(content, "`$1`")

	// Convert block math first: \[...\] or \\[...\\] -> $$...$$
	result := latexBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the inner content
		submatches := latexBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		inner := submatches[1]
		// Unescape backslashes within the math content
		// First try 4 backslashes -> 1 (table escaping)
		inner = quadrupleBackslashRegex.ReplaceAllString(inner, `\`)
		// Then 2 backslashes -> 1 (normal escaping)
		inner = doubleBackslashRegex.ReplaceAllString(inner, `\`)
		return "$$" + inner + "$$"
	})

	// Then convert inline math: \(...\) or \\(...\\) -> $...$
	result = latexInlineRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := latexInlineRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		inner := submatches[1]
		// Unescape backslashes within the math content
		// First try 4 backslashes -> 1 (table escaping)
		inner = quadrupleBackslashRegex.ReplaceAllString(inner, `\`)
		// Then 2 backslashes -> 1 (normal escaping)
		inner = doubleBackslashRegex.ReplaceAllString(inner, `\`)
		return "$" + inner + "$"
	})

	return result
}

// Convert converts HTML content to Markdown format
// It also converts Pandoc LaTeX delimiters to standard math syntax:
// - \(...\) -> $...$ (inline math)
// - \[...\] -> $$...$$ (block math)
func (c *Converter) Convert(html string) (string, error) {
	if html == "" {
		return "", nil
	}
	markdown, err := c.conv.ConvertString(html)
	if err != nil {
		return "", err
	}
	// Convert LaTeX delimiters after HTML to Markdown conversion
	return convertLaTeXDelimiters(markdown), nil
}
