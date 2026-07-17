package links

import (
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
