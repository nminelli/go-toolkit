package otelhttp

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// maxPathSegments is the threshold for falling back to method-only span names.
	// Paths with more segments are likely to contain dynamic data.
	maxPathSegments = 10
)

// PathPattern represents a regex pattern for matching and sanitizing URL paths.
// Patterns are evaluated in the order they appear, with first match winning.
type PathPattern struct {
	// Matcher is the compiled regex used to match URL paths
	Matcher *regexp.Regexp
	// Replacement is the template string used to replace matched paths
	Replacement string
}

// SpanNameConfig configures how span names are generated from HTTP requests.
type SpanNameConfig struct {
	// AdditionalPatterns are client-specific patterns evaluated in order after built-in patterns
	AdditionalPatterns []PathPattern
	// CustomSanitizer completely overrides the default sanitization logic
	CustomSanitizer func(method, path string) string
}

// builtInPatterns are pre-compiled regex patterns for common high-cardinality cases.
// These patterns are applied in order, with all matching patterns being applied sequentially.
var builtInPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	// OpenSearch/Elasticsearch document operations - MUST come first to preserve index names
	// These match the full path structure and prevent other patterns from matching individual segments
	{regexp.MustCompile(`^(/[^/]+)(/_doc/)[^/]+(/|$)`), "$1$2{id}$3"},
	{regexp.MustCompile(`^(/[^/]+)(/_update/)[^/]+(/|$)`), "$1$2{id}$3"},
	{regexp.MustCompile(`^(/[^/]+)(/_create/)[^/]+(/|$)`), "$1$2{id}$3"},
	// OpenSearch/Elasticsearch query operations (without document IDs) - preserve index and operation
	{regexp.MustCompile(`^(/[^/]+)(/_(?:search|count|bulk|mget|msearch|refresh|flush|analyze|validate|explain))(/|$)`), "$1$2$3"},
	// UUID v4/v5 (with hyphens)
	{regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(/|$)`), "/{uuid}$1"},
	// UUID without hyphens (32 hex chars)
	{regexp.MustCompile(`/[0-9a-f]{32}(/|$)`), "/{uuid}$1"},
	// Company IDs (2-3 letters + underscore + 6+ alphanumeric chars, e.g., acc_123asd, ev_12345asdnkjb23, cli_2134asdn2n3)
	{regexp.MustCompile(`/[a-zA-Z]{2,3}_[a-zA-Z0-9]{6,}(/|$)`), "/{id}$1"},
	// Hash strings (MD5: 32 chars, SHA-1: 40 chars, SHA-256: 64 chars)
	{regexp.MustCompile(`/[0-9a-f]{40}(/|$)`), "/{hash}$1"},
	{regexp.MustCompile(`/[0-9a-f]{64}(/|$)`), "/{hash}$1"},
	// Base64/URL-safe encoded strings (20+ alphanumeric chars, common in tokens)
	{regexp.MustCompile(`/[A-Za-z0-9_-]{20,}(/|$)`), "/{token}$1"},
	// ISO 8601 dates
	{regexp.MustCompile(`/\d{4}-\d{2}-\d{2}(/|$)`), "/{date}$1"},
	// Unix timestamps (10 digits)
	{regexp.MustCompile(`/\d{10}(/|$)`), "/{timestamp}$1"},
	// Numeric IDs (checked last as it's more general)
	{regexp.MustCompile(`/\d+(/|$)`), "/{id}$1"},
}

// SanitizeSpanName generates a low-cardinality span name for the HTTP request.
// It applies sanitization using first-match strategy in the following order:
// 1. Custom sanitizer (if configured) - complete override
// 2. Built-in heuristics (UUIDs, numeric IDs, hashes, etc.) - first match returns
// 3. Per-client patterns (from config.AdditionalPatterns) - first match returns
// 4. Fallback: if path has >6 segments, use method-only
// 5. Default: return "METHOD path" unchanged
func SanitizeSpanName(method, path string, config *SpanNameConfig) string {
	// If custom sanitizer is provided, use it exclusively
	if config != nil && config.CustomSanitizer != nil {
		return config.CustomSanitizer(method, path)
	}

	// Apply built-in heuristics - first match wins and returns immediately
	for _, pattern := range builtInPatterns {
		if pattern.pattern.MatchString(path) {
			sanitizedPath := pattern.pattern.ReplaceAllString(path, pattern.replacement)
			return fmt.Sprintf("%s %s", method, sanitizedPath)
		}
	}

	// Apply per-client patterns if provided - first match wins and returns immediately
	if config != nil && len(config.AdditionalPatterns) > 0 {
		for _, pattern := range config.AdditionalPatterns {
			if pattern.Matcher.MatchString(path) {
				sanitizedPath := pattern.Matcher.ReplaceAllString(path, pattern.Replacement)
				return fmt.Sprintf("%s %s", method, sanitizedPath)
			}
		}
	}

	// No patterns matched - check fallback condition on original path
	if strings.Count(path, "/") > maxPathSegments {
		return method
	}

	// Return original path unchanged
	return fmt.Sprintf("%s %s", method, path)
}
