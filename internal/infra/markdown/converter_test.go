package markdown

import (
	"strings"
	"testing"
)

func TestConverter_Convert(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "converts paragraph",
			input:    "<p>Hello World</p>",
			contains: "Hello World",
		},
		{
			name:     "converts heading",
			input:    "<h1>Title</h1>",
			contains: "# Title",
		},
		{
			name:     "converts link",
			input:    `<a href="https://example.com">Link</a>`,
			contains: "[Link](https://example.com)",
		},
		{
			name:     "converts code block",
			input:    "<pre><code>func main() {}</code></pre>",
			contains: "```",
		},
		{
			name:     "converts image",
			input:    `<img src="https://example.com/img.png" alt="test" />`,
			contains: "![test](https://example.com/img.png)",
		},
		{
			name:     "converts list",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: "- Item",
		},
		{
			name:     "converts bold and italic",
			input:    "<strong>bold</strong> <em>italic</em>",
			contains: "**bold**",
		},
		{
			name:     "handles empty input",
			input:    "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Errorf("Convert() error = %v", err)
				return
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Convert() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestConverter_Convert_ComplexHTML(t *testing.T) {
	converter := NewConverter()

	html := `
	<article>
		<h2>Article Title</h2>
		<p>This is a <strong>bold</strong> paragraph with a <a href="https://example.com">link</a>.</p>
		<pre><code class="language-go">fmt.Println("Hello")</code></pre>
		<img src="https://example.com/image.png" alt="An image" />
	</article>
	`

	result, err := converter.Convert(html)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify key conversions
	if !strings.Contains(result, "## Article Title") {
		t.Error("Expected heading to be converted")
	}
	if !strings.Contains(result, "**bold**") {
		t.Error("Expected bold to be converted")
	}
	if !strings.Contains(result, "[link](https://example.com)") {
		t.Error("Expected link to be converted")
	}
	if !strings.Contains(result, "![An image]") {
		t.Error("Expected image to be converted")
	}
}
