package cli

import (
	"strings"
	"testing"
)

func TestRunDiff_MissingArgs(t *testing.T) {
	err := RunDiff([]string{"only-one-path"})
	if err == nil {
		t.Errorf("Expected error for missing paths, got nil")
	}
	if !strings.Contains(err.Error(), "two bundle paths are required") {
		t.Errorf("Expected missing paths error, got %v", err)
	}
}

func TestRunMerge_MissingArgs(t *testing.T) {
	err := RunMerge([]string{"only-one-path"})
	if err == nil {
		t.Errorf("Expected error for missing paths, got nil")
	}
	if !strings.Contains(err.Error(), "two bundle paths are required") {
		t.Errorf("Expected missing paths error, got %v", err)
	}
}
