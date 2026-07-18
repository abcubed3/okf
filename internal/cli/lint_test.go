package cli

import (
	"strings"
	"testing"
)

func TestRunLint_InvalidPath(t *testing.T) {
	err := RunLint([]string{"/path/does/not/exist/12345"})
	if err == nil {
		t.Errorf("Expected error for invalid path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to resolve bundle path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected resolution error, got %v", err)
	}
}

func TestRunLint_HelpFlag(t *testing.T) {
	err := RunLint([]string{"--help"})
	if err == nil {
		t.Errorf("Expected error from flag.ErrHelp, got nil")
	}
}
