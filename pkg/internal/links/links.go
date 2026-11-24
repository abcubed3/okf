package links

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// IsExternal checks if a URL is an external link (http://, https://, //, mailto:, ftp://, or file://).
func IsExternal(rawURL string) bool {
	trimmed := strings.TrimSpace(rawURL)
	if strings.HasPrefix(trimmed, "//") {
		return true
	}
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "ftp://") ||
		strings.HasPrefix(lower, "file://")
}

// IsSchemeURI reports whether value has an explicit URI scheme (e.g. https://, postgresql://, hub://),
// excluding Windows drive-letter paths (e.g. C:\foo).
func IsSchemeURI(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if filepath.VolumeName(filepath.FromSlash(strings.ReplaceAll(trimmed, "\\", "/"))) != "" {
		return false
	}
	if IsExternal(trimmed) || strings.HasPrefix(strings.ToLower(trimmed), "hub://") {
		return true
	}
	parsed, err := url.Parse(trimmed)
	return err == nil && parsed.Scheme != ""
}

// IsLegacyVersion reports whether okfVersion explicitly targets OKF v0.1 or v0.2.
func IsLegacyVersion(okfVersion string) bool {
	v := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(okfVersion)), "v")
	return v == "0.1" || v == "0.2"
}

// ResolveBundlePath resolves an internal bundle path or markdown link according to OKF Spec v0.3 §6:
//   - Leading "/" anchors at the bundle root.
//   - Relative paths resolve from the directory containing containingRelPath.
//   - Legacy fallback: when IsLegacyVersion(okfVersion) is true and bundleRoot is provided, a bare
//     relative path (no leading "./" or "../") that does not exist relative to containingRelPath
//     falls back to bundle-root resolution if the target exists under bundleRoot.
//
// Returns the cleaned bundle-relative forward-slash path and escaped=true if the path traverses above bundle root.
func ResolveBundlePath(bundleRoot, containingRelPath, rawTarget, okfVersion string) (resolvedRelSlash string, escaped bool) {
	clean := strings.TrimSpace(strings.SplitN(rawTarget, "#", 2)[0])
	clean = strings.ReplaceAll(clean, "\\", "/")
	if clean == "" {
		return "", false
	}
	if filepath.VolumeName(filepath.FromSlash(clean)) != "" {
		return clean, true
	}

	if strings.HasPrefix(clean, "/") {
		trimmedRoot := strings.TrimLeft(clean, "/")
		if trimmedRoot == "" {
			return "", false
		}
		resolved := path.Clean(trimmedRoot)
		if resolved == ".." || strings.HasPrefix(resolved, "../") {
			return resolved, true
		}
		return resolved, false
	}

	currentDir := path.Dir(strings.ReplaceAll(containingRelPath, "\\", "/"))
	resolved := path.Clean(path.Join(currentDir, clean))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return resolved, true
	}

	// OKF v0.3 §6 / §12 backward compatibility fallback for v0.1/v0.2 bare root-relative paths
	if IsLegacyVersion(okfVersion) &&
		strings.TrimSpace(bundleRoot) != "" &&
		currentDir != "." &&
		!strings.HasPrefix(clean, "./") &&
		!strings.HasPrefix(clean, "../") {
		if !targetExistsUnderRoot(bundleRoot, resolved) {
			rootCandidate := path.Clean(clean)
			if rootCandidate != ".." && !strings.HasPrefix(rootCandidate, "../") && targetExistsUnderRoot(bundleRoot, rootCandidate) {
				return rootCandidate, false
			}
		}
	}

	return resolved, false
}

func targetExistsUnderRoot(bundleRoot, relSlash string) bool {
	absRoot, err := filepath.Abs(bundleRoot)
	if err != nil {
		return false
	}
	candidates := []string{relSlash}
	if path.Ext(relSlash) == "" {
		candidates = append(candidates, relSlash+".md", path.Join(relSlash, "index.md"))
	}
	for _, cand := range candidates {
		target := filepath.Join(absRoot, filepath.FromSlash(cand))
		rel, err := filepath.Rel(absRoot, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
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
