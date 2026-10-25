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

func TestRunLint_StrictFlag(t *testing.T) {
	tmp := t.TempDir()
	concept := `---
type: Metric
---
# Metric
`
	if err := os.WriteFile(filepath.Join(tmp, "metric.md"), []byte(concept), 0644); err != nil {
		t.Fatal(err)
	}

	// Without --strict, warnings only -> nil error
	if err := RunLint([]string{tmp}); err != nil {
		t.Fatalf("expected nil error without strict mode, got %v", err)
	}

	// With --strict, warnings trigger an error
	err := RunLint([]string{"--strict", tmp})
	if err == nil {
		t.Fatal("expected error with strict mode on warnings, got nil")
	}
	if !strings.Contains(err.Error(), "strict mode") {
		t.Errorf("expected strict mode error message, got %v", err)
	}
}
