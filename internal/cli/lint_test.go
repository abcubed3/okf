package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLint_InvalidPath(t *testing.T) {
	invalidPath := filepath.Join(t.TempDir(), "missing-bundle")
	err := RunLint([]string{invalidPath})
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse bundle") || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected bundle parse error for missing path, got %v", err)
	}
}

func TestRunLint_HelpFlag(t *testing.T) {
	err := RunLint([]string{"--help"})
	if err == nil {
		t.Errorf("Expected error from flag.ErrHelp, got nil")
	}
}
