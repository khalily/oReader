package rss

import (
	"strings"
	"testing"
)

func TestFixRelativeImageURLs(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		baseURL        string
		wantContains   string
		wantNotContains string
	}{
		{
			name:         "fix relative path",
			html:         `<img src="/images/photo.jpg">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="https://example.com/images/photo.jpg"`,
		},
		{
			name:         "preserve absolute URL",
			html:         `<img src="https://other.com/img.png">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="https://other.com/img.png"`,
		},
		{
			name:         "preserve data URI",
			html:         `<img src="data:image/png;base64,abc">`,
			baseURL:      "https://example.com/article",
			wantContains: `src="data:image/png;base64,abc"`,
		},
		{
			name:         "fix relative with subdirectory",
			html:         `<img src="../images/photo.jpg">`,
			baseURL:      "https://example.com/blog/article",
			wantContains: `src="https://example.com/images/photo.jpg"`,
		},
		{
			name:         "empty content returns unchanged",
			html:         "",
			baseURL:      "https://example.com",
			wantContains: "",
		},
		{
			name:         "empty baseURL returns unchanged",
			html:         `<img src="/img.jpg">`,
			baseURL:      "",
			wantContains: `<img src="/img.jpg">`,
		},
		{
			name:            "multiple images - all fixed",
			html:            `<img src="/a.jpg"><img src="https://other.com/b.jpg"><img src="/c.jpg">`,
			baseURL:         "https://example.com",
			wantContains:    `src="https://example.com/a.jpg"`,
			wantNotContains: `src="/a.jpg"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FixRelativeImageURLs(tt.html, tt.baseURL)

			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("FixRelativeImageURLs() = %q, want to contain %q", got, tt.wantContains)
			}
			if tt.wantNotContains != "" && strings.Contains(got, tt.wantNotContains) {
				t.Errorf("FixRelativeImageURLs() = %q, should NOT contain %q", got, tt.wantNotContains)
			}
		})
	}
}
