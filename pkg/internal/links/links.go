package links

import (
	"fmt"
	"strings"
)

// IsExternal checks if a URL is an external link (http://, https://, or mailto:).
func IsExternal(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "ftp://") ||
		strings.HasPrefix(lower, "file://")
}

// ParseBundleID validates a canonical 2-segment "<org>/<bundle>" identifier
// (optionally prefixed with "hub://") and returns "<org>/<bundle>".
func ParseBundleID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "hub://")
	trimmed = strings.Trim(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid bundle ID %q: expected canonical 2-segment <org>/<bundle> format", raw)
	}
	return parts[0] + "/" + parts[1], nil
}

// ParseHubURI parses a hub://<org>/<bundle>[/<concept/path>] URI.
// Optional #fragment suffixes are stripped before parsing, and an optional .md suffix on
// concept paths is stripped.
func ParseHubURI(raw string) (bundleID string, conceptID string, err error) {
	if !strings.HasPrefix(raw, "hub://") {
		return "", "", fmt.Errorf("invalid hub URI %q: must start with hub://", raw)
	}
	rest := strings.TrimPrefix(raw, "hub://")
	if idx := strings.Index(rest, "#"); idx != -1 {
		rest = rest[:idx]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasSuffix(rest, "/") {
		return "", "", fmt.Errorf("invalid hub URI %q: expected hub://<org>/<bundle>[/<concept/path>]", raw)
	}

	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid hub URI %q: bundle ID must be 2 segments (<org>/<bundle>)", raw)
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", "", fmt.Errorf("invalid hub URI %q: empty or relative path segment", raw)
		}
	}

	bundleID = parts[0] + "/" + parts[1]
	if len(parts) > 2 {
		conceptID = strings.TrimSuffix(strings.Join(parts[2:], "/"), ".md")
		if conceptID == "" {
			return "", "", fmt.Errorf("invalid hub URI %q: empty concept path", raw)
		}
	}
	return bundleID, conceptID, nil
}
