package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
)

// IsGitURL checks if a path/string is a Git remote repository URL.
func IsGitURL(path string) bool {
	p := strings.ToLower(path)
	return strings.HasPrefix(p, "http://") ||
		strings.HasPrefix(p, "https://") ||
		strings.HasPrefix(p, "git@") ||
		strings.HasPrefix(p, "ssh://") ||
		strings.HasPrefix(p, "git://") ||
		strings.HasSuffix(p, ".git")
}

// ResolvePath checks if the given path is a Git URL. If so, it clones the
// repository to a temporary directory and returns the local directory path
// along with a cleanup function to delete it when done.
// If it is a local path, it returns the path as-is with a no-op cleanup function.
func ResolvePath(path string) (string, func(), error) {
	if !IsGitURL(path) {
		return path, func() {}, nil
	}

	tempDir, err := os.MkdirTemp("", "okf-git-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create temporary directory for git clone: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      path,
		Depth:    1,
		Progress: nil,
	})
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("failed to clone remote git repository %q: %w", path, err)
	}

	return tempDir, cleanup, nil
}
