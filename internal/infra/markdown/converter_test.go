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

func TestConverter_Convert_LaTeXDelimiters(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "inline math Pandoc delimiter",
			input:    `\(E = mc^2\)`,
			expected: `$E = mc^2$`,
		},
		{
			name:     "block math Pandoc delimiter",
			input:    `\[\frac{\partial L}{\partial w} = \nabla\]`,
			expected: `$$\frac{\partial L}{\partial w} = \nabla$$`,
		},
		{
			name:     "inline math in HTML paragraph",
			input:    `<p>The formula \(a + b\) equals c.</p>`,
			expected: `$a + b$`,
		},
		{
			name:     "block math with newlines",
			input:    "\\[\n\\begin{aligned}\na &= b \\\\\nc &= d\n\\end{aligned}\n\\]",
			expected: "$$",
		},
		{
			name:     "multiple inline formulas",
			input:    `\(x\) and \(y\) are variables`,
			expected: `$x$ and $y$ are variables`,
		},
		{
			name:     "mixed inline and block",
			input:    `Inline \(a + b\) and block \[c = d\]`,
			expected: `$a + b$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Errorf("Convert() error = %v", err)
				return
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Convert() = %v, want to contain %v", result, tt.expected)
			}
		})
	}
}

func TestConvertLaTeXDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single inline",
			input:    `\(E = mc^2\)`,
			expected: `$E = mc^2$`,
		},
		{
			name:     "single block",
			input:    `\[\nabla\]`,
			expected: `$$\nabla$$`,
		},
		{
			name:     "multiple inline",
			input:    `\(a\) + \(b\) = \(c\)`,
			expected: `$a$ + $b$ = $c$`,
		},
		{
			name:     "block with newlines",
			input:    "\\[\na = b\n\\]",
			expected: "$$\na = b\n$$",
		},
		{
			name:     "no delimiters",
			input:    `plain text`,
			expected: `plain text`,
		},
		{
			name:     "already dollar syntax",
			input:    `$E = mc^2$`,
			expected: `$E = mc^2$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLaTeXDelimiters(tt.input)
			if result != tt.expected {
				t.Errorf("convertLaTeXDelimiters() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConverter_Convert_TableWithMath(t *testing.T) {
	converter := NewConverter()

	html := `<table>
<thead><tr><th>Operation</th><th>Forward</th><th>Local gradients</th></tr></thead>
<tbody>
<tr><td><code>a + b</code></td><td>\(a + b\)</td><td>\(\frac{\partial}{\partial a} = 1\)</td></tr>
<tr><td><code>a * b</code></td><td>\(a \cdot b\)</td><td>\(\frac{\partial}{\partial a} = b\)</td></tr>
</tbody>
</table>`

	result, err := converter.Convert(html)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Verify table structure is preserved
	if !strings.Contains(result, "| Operation |") {
		t.Error("Expected table header with | separators")
	}
	if !strings.Contains(result, "| --- |") {
		t.Error("Expected table divider with | separators")
	}

	// Verify inline math is converted
	if !strings.Contains(result, "$a + b$") {
		t.Errorf("Expected inline math $a + b$, got: %s", result)
	}
	if !strings.Contains(result, `$\frac{\partial}{\partial a} = 1$`) {
		t.Errorf("Expected LaTeX fraction converted, got: %s", result)
	}

	t.Logf("Converted table:\n%s", result)
}

func TestConverter_Convert_TableWithLaTeXCommands(t *testing.T) {
	converter := NewConverter()

	// Test \ln, \max, \mathbf and other LaTeX commands in tables
	html := `<table>
<thead><tr><th>Operation</th><th>Forward</th><th>Local gradients</th></tr></thead>
<tbody>
<tr><td><code>log(a)</code></td><td>\(\ln(a)\)</td><td>\(\frac{\partial}{\partial a} = \frac{1}{a}\)</td></tr>
<tr><td><code>exp(a)</code></td><td>\(e^a\)</td><td>\(\frac{\partial}{\partial a} = e^a\)</td></tr>
<tr><td><code>relu(a)</code></td><td>\(\max(0, a)\)</td><td>\(\frac{\partial}{\partial a} = \mathbf{1}_{a > 0}\)</td></tr>
</tbody>
</table>`

	result, err := converter.Convert(html)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	t.Logf("Converted table:\n%s", result)

	// Verify LaTeX commands are correctly converted
	// Note: _ in math is escaped as \_ by html-to-markdown (Markdown escaping)
	tests := []struct {
		expected string
		desc     string
	}{
		{`$\ln(a)$`, "ln command"},
		{`$\max(0, a)$`, "max command"},
		{`\mathbf{1}\_{a > 0}$`, "mathbf command (underscore escaped)"},  // Without leading $ since there's space before \mathbf
		{`$\frac{\partial}{\partial a} = \frac{1}{a}$`, "fraction with partial"},
	}

	for _, tt := range tests {
		if !strings.Contains(result, tt.expected) {
			t.Errorf("Expected %s: %q, got: %q", tt.desc, tt.expected, result)
		}
	}
}

func TestConverter_Convert_CodeWithBackticks(t *testing.T) {
	converter := NewConverter()

	// Test Jekyll/Rouge style: <code>`content`</code> should become `content`, not `` `content` ``
	tests := []struct {
		name     string
		input    string
		expected string
		notWant  string
	}{
		{
			name:     "code with inner backticks",
			input:    `<code>` + "`" + `log(a)` + "`" + `</code>`,
			expected: "`log(a)`",
			notWant:  "`` `log(a)` ``",
		},
		{
			name:     "code without inner backticks",
			input:    `<code>log(a)</code>`,
			expected: "`log(a)`",
			notWant:  "`` `log(a)` ``",
		},
		{
			name: "code in table with inner backticks",
			input: `<table><tbody><tr><td><code>` + "`" + `a + b` + "`" + `</code></td></tr></tbody></table>`,
			expected: "| `a + b` |",
			notWant:  "`` `a + b` ``",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected to contain %q, got: %q", tt.expected, result)
			}
			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("Should not contain %q, got: %q", tt.notWant, result)
			}
		})
	}
}

func TestConverter_Convert_CodeBlockLanguage(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name        string
		input       string
		wantContain string
		wantNot     string
	}{
		{
			name:        "language on code element (standard)",
			input:       `<pre><code class="language-go">fmt.Println("Hello")</code></pre>`,
			wantContain: "```go",
			wantNot:     "```language-go",
		},
		{
			name:        "Jekyll/Rouge: language on wrapper div",
			input:       `<div class="language-python highlighter-rouge"><div class="highlight"><pre class="highlight"><code>print("Hello")</code></pre></div></div>`,
			wantContain: "```python",
			wantNot:     "```\n",
		},
		{
			name:        "Rouge: language on pre element",
			input:       `<pre class="language-rust"><code>fn main() {}</code></pre>`,
			wantContain: "```rust",
		},
		{
			name:        "no language at all",
			input:       `<pre><code>fmt.Println("Hello")</code></pre>`,
			wantContain: "```\nfmt.Println",
		},
		{
			name:        "language on code takes priority over parent",
			input:       `<div class="language-python highlighter-rouge"><pre><code class="language-go">fmt.Println("Hello")</code></pre></div>`,
			wantContain: "```go",
			wantNot:     "```python",
		},
		{
			name:        "Jekyll/Rouge with javascript language",
			input:       `<div class="language-javascript highlighter-rouge"><div class="highlight"><pre class="highlight"><code>console.log("Hi")</code></pre></div></div>`,
			wantContain: "```javascript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if !strings.Contains(result, tt.wantContain) {
				t.Errorf("Expected to contain %q, got: %q", tt.wantContain, result)
			}
			if tt.wantNot != "" && strings.Contains(result, tt.wantNot) {
				t.Errorf("Should not contain %q, got: %q", tt.wantNot, result)
			}
		})
	}
}
