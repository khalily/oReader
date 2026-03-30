package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeArticleContent_SafeHTML(t *testing.T) {
	safeHTML := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "paragraphs",
			input:    "<p>Hello world</p>",
			expected: "<p>Hello world</p>",
		},
		{
			name:     "links",
			input:    `<a href="https://example.com">Link</a>`,
			expected: `<a href="https://example.com" rel="nofollow">Link</a>`,
		},
		{
			name:     "images",
			input:    `<img src="https://example.com/img.jpg" alt="Image">`,
			expected: `<img src="https://example.com/img.jpg" alt="Image">`,
		},
		{
			name:     "bold",
			input:    "<strong>Bold text</strong>",
			expected: "<strong>Bold text</strong>",
		},
		{
			name:     "italic",
			input:    "<em>Italic text</em>",
			expected: "<em>Italic text</em>",
		},
		{
			name:     "lists",
			input:    "<ul><li>Item</li></ul>",
			expected: "<ul><li>Item</li></ul>",
		},
		{
			name:     "headings",
			input:    "<h1>Heading</h1>",
			expected: "<h1>Heading</h1>",
		},
		{
			name:     "code blocks",
			input:    "<pre><code>const x = 1;</code></pre>",
			expected: "<pre><code>const x = 1;</code></pre>",
		},
		{
			name:     "blockquotes",
			input:    "<blockquote>Quote</blockquote>",
			expected: "<blockquote>Quote</blockquote>",
		},
		{
			name:     "line breaks",
			input:    "First line<br>Second line",
			expected: "First line<br>Second line",
		},
	}

	for _, tc := range safeHTML {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeArticleContent(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeArticleContent(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSanitizeArticleContent_XSS(t *testing.T) {
	xssPayloads := []struct {
		name  string
		input string
	}{
		{
			name:  "script tag",
			input: "<script>alert('XSS')</script>",
		},
		{
			name:  "script with src",
			input: `<script src="https://evil.com/exploit.js"></script>`,
		},
		{
			name:  "onclick handler",
			input: `<div onclick="alert('XSS')">Click me</div>`,
		},
		{
			name:  "onerror handler",
			input: `<img src="x" onerror="alert('XSS')">`,
		},
		{
			name:  "javascript href",
			input: `<a href="javascript:alert('XSS')">Click</a>`,
		},
		{
			name:  "data URI with script",
			input: `<iframe src="data:text/html,<script>alert('XSS')</script>"></iframe>`,
		},
		{
			name:  "SVG with script",
			input: `<svg onload="alert('XSS')">`,
		},
		{
			name:  "style tag",
			input: `<style>body { background: red; }</style>`,
		},
		{
			name:  "form tag",
			input: `<form action="https://evil.com">`,
		},
		{
			name:  "input tag",
			input: `<input type="text" name="data">`,
		},
		{
			name:  "object tag",
			input: `<object data="exploit.swf">`,
		},
		{
			name:  "embed tag",
			input: `<embed src="exploit.swf">`,
		},
		{
			name:  "iframe",
			input: `<iframe src="https://evil.com">`,
		},
		{
			name:  "meta refresh",
			input: `<meta http-equiv="refresh" content="0;url=https://evil.com">`,
		},
	}

	for _, tc := range xssPayloads {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeArticleContent(tc.input)
			// Check that dangerous elements/attributes are removed
			if containsScript(result) {
				t.Errorf("SanitizeArticleContent(%q) = %q, still contains dangerous content", tc.input, result)
			}
		})
	}
}

func TestSanitizeArticleContent_EventHandlers(t *testing.T) {
	eventHandlers := []struct {
		name  string
		input string
	}{
		{"onclick", `<div onclick="alert('XSS')">Text</div>`},
		{"onload", `<img src="x" onload="alert('XSS')">`},
		{"onerror", `<img src="x" onerror="alert('XSS')">`},
		{"onmouseover", `<div onmouseover="alert('XSS')">Text</div>`},
		{"onfocus", `<input onfocus="alert('XSS')">`},
		{"onblur", `<input onblur="alert('XSS')">`},
		{"onkeydown", `<body onkeydown="alert('XSS')">`},
	}

	for _, tc := range eventHandlers {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeArticleContent(tc.input)
			if containsEventHandler(result) {
				t.Errorf("SanitizeArticleContent(%q) = %q, still contains event handler", tc.input, result)
			}
		})
	}
}

func TestSanitizeFeedTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "My Blog",
			expected: "My Blog",
		},
		{
			name:     "HTML tags stripped",
			input:    "<script>evil</script>My Blog",
			expected: "My Blog",
		},
		{
			name:     "HTML entities kept as-is (strict policy)",
			input:    "My &amp; Blog",
			expected: "My &amp; Blog",
		},
		{
			name:     "XSS attempt",
			input:    `<img src="x" onerror="alert('XSS')">Blog`,
			expected: "Blog",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeFeedTitle(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeFeedTitle(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tags",
			input:    "<p>Hello <strong>world</strong></p>",
			expected: "Hello world",
		},
		{
			name:     "nested tags",
			input:    "<div><p>Text</p></div>",
			expected: "Text",
		},
		{
			name:     "no tags",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "tags with attributes",
			input:    `<a href="https://example.com">Link</a>`,
			expected: "Link",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StripHTML(tc.input)
			if result != tc.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short text",
			input:    "Short",
			maxLen:   200,
			expected: "Short",
		},
		{
			name:     "exact length",
			input:    "12345",
			maxLen:   5,
			expected: "12345",
		},
		{
			name:     "truncate",
			input:    "This is a long text that should be truncated",
			maxLen:   20,
			expected: "This is a long te...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TruncateText(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

// Helper functions
func containsScript(s string) bool {
	return len(s) > 0 && containsAny(s, "<script", "javascript:", "data:text/html", "onload=", "onerror=", "onclick=")
}

func containsEventHandler(s string) bool {
	return containsAny(s, "onload=", "onerror=", "onclick=", "onmouseover=", "onfocus=", "onblur=", "onkeydown=")
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// ===== Tests for enhanced sanitization (class attributes) =====

func TestSanitizeArticleContent_ClassAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeep string
		wantDrop string
	}{
		{
			name:     "preserve language class on pre",
			input:    `<pre class="language-go">code</pre>`,
			wantKeep: `class="language-go"`,
		},
		{
			name:     "preserve token class on span",
			input:    `<span class="token keyword">func</span>`,
			wantKeep: `class="token keyword"`,
		},
		{
			name:     "drop malicious class",
			input:    `<div class="onclick-alert">text</div>`,
			wantDrop: `onclick`,
		},
		{
			name:     "preserve prose class",
			input:    `<p class="prose-lg">text</p>`,
			wantKeep: `prose`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeArticleContent(tt.input)

			if tt.wantKeep != "" && !strings.Contains(got, tt.wantKeep) {
				t.Errorf("Expected to keep %q, got %q", tt.wantKeep, got)
			}
			if tt.wantDrop != "" && strings.Contains(got, tt.wantDrop) {
				t.Errorf("Expected to drop %q, got %q", tt.wantDrop, got)
			}
		})
	}
}

func TestSanitizeArticleContent_ImageAttributes(t *testing.T) {
	input := `<img src="https://example.com/img.jpg" alt="test" loading="lazy" referrerpolicy="no-referrer">`
	got := SanitizeArticleContent(input)

	if !strings.Contains(got, `loading="lazy"`) {
		t.Error("Expected loading attribute to be preserved")
	}
	if !strings.Contains(got, `referrerpolicy="no-referrer"`) {
		t.Error("Expected referrerpolicy attribute to be preserved")
	}
}

func TestSanitizeArticleContent_DataAttributes(t *testing.T) {
	input := `<pre data-language="go" data-line="5">code</pre>`
	got := SanitizeArticleContent(input)

	if !strings.Contains(got, `data-language="go"`) {
		t.Error("Expected data-language attribute to be preserved")
	}
}
