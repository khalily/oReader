package testutil

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var (
	// openapiParamRe matches OpenAPI path parameters like {feedId}, {itemId}
	openapiParamRe = regexp.MustCompile(`\{[^}]+\}`)
	// ginParamRe matches Gin path parameters like :id, :job_id
	ginParamRe = regexp.MustCompile(`:[a-zA-Z_][a-zA-Z0-9_]*`)
)

// openapiSpec represents the minimal structure we need from an OpenAPI spec.
type openapiSpec struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

// TestOpenAPIRoutes verifies that every route in docs/openapi.yaml has a
// corresponding handler in cmd/server/main.go and vice versa.
//
// This is a static analysis test — no MySQL or running server is required.
// It works by parsing the YAML spec and comparing against a hardcoded list of
// routes extracted from the Gin router registration in main.go.
func TestOpenAPIRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	specRoutes := extractSpecRoutes(t, spec)
	backendRoutes := expectedBackendRoutes()

	t.Logf("OpenAPI spec defines %d routes", len(specRoutes))
	t.Logf("Backend defines %d routes", len(backendRoutes))

	// Normalize both sets for comparison
	specNormalized := normalizeRoutes(specRoutes)
	backendNormalized := normalizeRoutes(backendRoutes)

	// Find differences
	var specOnly, backendOnly, matched []string

	for k := range specNormalized {
		if _, ok := backendNormalized[k]; ok {
			matched = append(matched, k)
		} else {
			specOnly = append(specOnly, k)
		}
	}
	for k := range backendNormalized {
		if _, ok := specNormalized[k]; !ok {
			backendOnly = append(backendOnly, k)
		}
	}

	sort.Strings(matched)
	sort.Strings(specOnly)
	sort.Strings(backendOnly)

	t.Logf("Matched routes (%d):", len(matched))
	for _, r := range matched {
		t.Logf("  ✓ %s", r)
	}

	if len(specOnly) > 0 {
		t.Logf("Routes in spec but NOT in backend (%d):", len(specOnly))
		for _, r := range specOnly {
			t.Logf("  ✗ spec-only: %s", r)
		}
	}

	if len(backendOnly) > 0 {
		t.Logf("Routes in backend but NOT in spec (%d):", len(backendOnly))
		for _, r := range backendOnly {
			t.Logf("  ✗ backend-only: %s", r)
		}
	}

	assert.Empty(t, specOnly, "OpenAPI spec contains routes not registered in backend (spec is stale)")
	assert.Empty(t, backendOnly, "Backend has routes not documented in OpenAPI spec (spec is incomplete)")
}

// expectedBackendRoutes returns the authoritative list of {method, path} pairs
// registered in cmd/server/main.go via the Gin router.
//
// IMPORTANT: When routes are added/removed in main.go, this list must be
// updated accordingly.
func expectedBackendRoutes() map[string]string {
	routes := map[string]string{
		// Health
		"GET /health": "healthCheck",

		// Auth (public)
		"POST /api/v1/auth/register":       "register",
		"POST /api/v1/auth/login":          "login",
		"POST /api/v1/auth/refresh":        "refreshToken",
		"POST /api/v1/auth/logout":         "logout",
		"GET /api/v1/auth/github":          "githubOAuthInitiate",
		"GET /api/v1/auth/github/callback": "githubOAuthCallback",
		"GET /api/v1/auth/oauth/pending":   "getPendingOAuth",
		"POST /api/v1/auth/oauth/complete": "completeOAuth",

		// Auth (protected)
		"GET /api/v1/auth/me": "getCurrentUser",

		// Feeds (protected)
		"GET /api/v1/feeds":              "listFeeds",
		"POST /api/v1/feeds":             "createFeed",
		"GET /api/v1/feeds/:id":          "getFeed",
		"DELETE /api/v1/feeds/:id":       "deleteFeed",
		"POST /api/v1/feeds/:id/refresh": "refreshFeed",
		"POST /api/v1/feeds/:id/mark-all-read": "markAllRead",

		// Items (protected)
		"GET /api/v1/items":          "listItems",
		"GET /api/v1/items/:id":      "getItem",
		"PUT /api/v1/items/:id/star": "setStar",
		"PUT /api/v1/items/:id/read": "setRead",

		// Stats (protected)
		"GET /api/v1/stats": "getStats",

		// OPML (protected)
		"POST /api/v1/opml/import": "importOpml",

		// OPML (optional auth)
		"GET /api/v1/opml/export":          "exportOpml",
		"GET /api/v1/opml/import/:job_id":  "getImportStatus",

		// Papers (protected)
		"POST /api/v1/papers/upload":       "uploadPaper",
		"GET /api/v1/papers":               "listPapers",
		"GET /api/v1/papers/tags":          "listPaperTags",
		"GET /api/v1/papers/:id":           "getPaper",
		"GET /api/v1/papers/:id/status":    "getPaperStatus",
		"PUT /api/v1/papers/:id":           "updatePaper",
		"PUT /api/v1/papers/:id/tags":      "updatePaperTags",
		"POST /api/v1/papers/:id/retry":    "retryPaper",
		"DELETE /api/v1/papers/:id":        "deletePaper",
		"GET /api/v1/papers/:id/download":  "downloadPaper",
	}

	return routes
}

// loadOpenAPISpec reads and parses docs/openapi.yaml.
func loadOpenAPISpec(t *testing.T) *openapiSpec {
	t.Helper()

	// Walk up from testutil/ to find docs/openapi.yaml
	specPath := "../../../docs/openapi.yaml"
	data, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read OpenAPI spec at %s", specPath)

	var spec openapiSpec
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err, "Failed to parse OpenAPI spec YAML")

	require.NotEmpty(t, spec.Paths, "OpenAPI spec has no paths defined")

	return &spec
}

// extractSpecRoutes extracts {METHOD path} pairs from the OpenAPI spec.
func extractSpecRoutes(t *testing.T, spec *openapiSpec) map[string]string {
	t.Helper()

	routes := make(map[string]string)
	httpMethods := []string{"get", "post", "put", "delete", "patch", "options", "head"}

	for path, methods := range spec.Paths {
		for _, method := range httpMethods {
			if op, ok := methods[method]; ok {
				opMap, ok := op.(map[string]any)
				if !ok {
					continue
				}
				operationID := ""
				if id, ok := opMap["operationId"]; ok {
					operationID = fmt.Sprintf("%v", id)
				}
				key := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
				routes[key] = operationID
			}
		}
	}

	return routes
}

// normalizeRoutes replaces path parameters with a uniform placeholder.
// Gin uses :param, OpenAPI uses {param}. Both are normalized to :param.
func normalizeRoutes(routes map[string]string) map[string]string {
	normalized := make(map[string]string, len(routes))
	for k, v := range routes {
		// Replace OpenAPI {paramName} with :param
		normalizedKey := replacePathParams(k)
		normalized[normalizedKey] = v
	}
	return normalized
}

// replacePathParams normalizes both Gin (:name) and OpenAPI ({name}) path
// parameters into a canonical form (:param) for comparison.
func replacePathParams(route string) string {
	// Replace OpenAPI {paramName} with :param
	result := openapiParamRe.ReplaceAllString(route, ":param")
	// Replace Gin :paramName with :param (but don't double-replace)
	result = ginParamRe.ReplaceAllString(result, ":param")
	return result
}
