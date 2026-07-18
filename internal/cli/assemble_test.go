package cli

import (
	"strings"
	"testing"
)

func TestRunAssemble_MissingID(t *testing.T) {
	err := RunAssemble([]string{})
	if err == nil {
		t.Errorf("Expected error for missing start Concept ID, got nil")
	}
	if !strings.Contains(err.Error(), "start Concept ID is required") {
		t.Errorf("Expected missing ID error, got %v", err)
	}
}

func TestRunAssemble_InvalidDirection(t *testing.T) {
	err := RunAssemble([]string{"--direction=invalid-dir", "my-concept"})
	if err == nil {
		t.Errorf("Expected error for invalid direction, got nil")
	}
	if !strings.Contains(err.Error(), "invalid direction") {
		t.Errorf("Expected invalid direction error, got %v", err)
	}
}
