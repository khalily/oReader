package validation

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// Blocked IP ranges for SSRF protection
var blockedIPRanges = []string{
	"10.0.0.0/8",       // Private class A
	"172.16.0.0/12",    // Private class B
	"192.168.0.0/16",   // Private class C
	"127.0.0.0/8",      // Loopback
	"169.254.0.0/16",   // AWS metadata and link-local
	"0.0.0.0/8",        // Invalid/metadata
	"::1/128",          // IPv6 loopback
	"fe80::/10",        // IPv6 link-local
	"fc00::/7",         // IPv6 private
}

var blockedNets []*net.IPNet

func init() {
	// Pre-parse the IP networks for efficient checking
	for _, cidr := range blockedIPRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err) // Invalid CIDR in source code - fail fast
		}
		blockedNets = append(blockedNets, network)
	}
}

// ValidateFeedURL validates a feed URL for security (SSRF protection)
// It checks:
// 1. URL is well-formed
// 2. Scheme is http or https only
// 3. Host does not resolve to a blocked IP range (private, loopback, link-local)
func ValidateFeedURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL cannot be empty")
	}

	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid URL format")
	}

	// Check scheme - only http and https allowed
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("only http and https schemes are allowed")
	}

	// Check host is present
	host := u.Hostname()
	if host == "" {
		return errors.New("URL must have a host")
	}

	// Check if host is an IP address
	ip := net.ParseIP(host)
	if ip != nil {
		// Direct IP in URL
		if isBlockedIP(ip) {
			return errors.New("URL with private IP addresses is not allowed")
		}
		return nil
	}

	// For hostname, we need to resolve it to check IPs
	// This is a basic check - in production you might want to cache this
	ips, err := net.LookupIP(host)
	if err != nil {
		return errors.New("unable to resolve hostname")
	}

	// Check if any resolved IP is in a blocked range
	for _, resolvedIP := range ips {
		if isBlockedIP(resolvedIP) {
			return errors.New("URL resolves to a blocked IP range")
		}
	}

	return nil
}

// isBlockedIP checks if an IP address is in a blocked range
func isBlockedIP(ip net.IP) bool {
	// Convert to 4-byte or 16-byte representation for comparison
	ip = ip.To16()
	if ip == nil {
		return false
	}

	for _, network := range blockedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// IsPrivateIP checks if an IP is in a private range (exported for testing)
func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return isBlockedIP(ip)
}
