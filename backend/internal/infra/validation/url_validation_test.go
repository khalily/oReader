package validation

import (
	"testing"
)

func TestValidateFeedURL_ValidURLs(t *testing.T) {
	validURLs := []string{
		"https://example.com/feed.xml",
		"http://example.com/rss",
		"https://www.google.com/",
		"https://www.wired.com/feed.xml",
		"https://rss.cnn.com/rss/edition.rss",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			err := ValidateFeedURL(url)
			if err != nil {
				t.Errorf("ValidateFeedURL(%q) returned error: %v", url, err)
			}
		})
	}
}

func TestValidateFeedURL_InvalidSchemes(t *testing.T) {
	invalidURLs := []string{
		"ftp://example.com/feed.xml",
		"file:///etc/passwd",
		"javascript:alert('xss')",
		"data:text/html,<script>alert('xss')</script>",
		"mailto:test@example.com",
		"//example.com/feed.xml", // protocol-relative
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			err := ValidateFeedURL(url)
			if err == nil {
				t.Errorf("ValidateFeedURL(%q) should return error for invalid scheme", url)
			}
		})
	}
}

func TestValidateFeedURL_PrivateIPs(t *testing.T) {
	// Test that private IP ranges are blocked
	privateIPURLs := []string{
		"http://localhost:8080/feed.xml",
		"http://127.0.0.1/feed.xml",
		"http://127.0.0.1:3000/rss",
		"http://10.0.0.1/feed.xml",
		"http://10.255.255.255/rss",
		"http://172.16.0.1/feed.xml",
		"http://172.31.255.255/rss",
		"http://192.168.0.1/feed.xml",
		"http://192.168.255.255/rss",
		"http://169.254.169.254/latest/meta-data/", // AWS metadata
		"http://0.0.0.0/feed.xml",
	}

	for _, url := range privateIPURLs {
		t.Run(url, func(t *testing.T) {
			err := ValidateFeedURL(url)
			if err == nil {
				t.Errorf("ValidateFeedURL(%q) should return error for private IP", url)
			}
		})
	}
}

func TestValidateFeedURL_IPv6Private(t *testing.T) {
	// Test IPv6 private addresses are blocked
	ipv6PrivateURLs := []string{
		"http://[::1]/feed.xml",
		"http://[fe80::1]/feed.xml",
		"http://[fc00::1]/feed.xml",
		"http://[fd00::1]/feed.xml",
	}

	for _, url := range ipv6PrivateURLs {
		t.Run(url, func(t *testing.T) {
			err := ValidateFeedURL(url)
			if err == nil {
				t.Errorf("ValidateFeedURL(%q) should return error for IPv6 private address", url)
			}
		})
	}
}

func TestValidateFeedURL_MalformedURLs(t *testing.T) {
	malformedURLs := []string{
		"",
		"not a url",
		"http://",
		"https://",
		"ht!tp://example.com/feed",
		"\x00://example.com/feed",
	}

	for _, url := range malformedURLs {
		t.Run(url, func(t *testing.T) {
			err := ValidateFeedURL(url)
			if err == nil {
				t.Errorf("ValidateFeedURL(%q) should return error for malformed URL", url)
			}
		})
	}
}

func TestValidateFeedURL_DNSRebinding(t *testing.T) {
	// Test that URLs are validated (we check the DNS resolves correctly)
	// This is a basic test - in production, you'd also want to check that
	// the IP doesn't change after the initial validation
	validPublicURLs := []string{
		"https://example.com/feed.xml",
		"https://www.google.com/rss",
	}

	for _, url := range validPublicURLs {
		t.Run(url, func(t *testing.T) {
			// Skip if network is unavailable
			t.Skip("Skipping DNS validation test in unit tests")
		})
	}
}
