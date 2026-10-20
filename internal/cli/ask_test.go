package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/abcubed3/okf/internal/cli"
)

func TestRunAsk_MissingArgs(t *testing.T) {
	err := cli.RunAsk([]string{})
	if err == nil {
		t.Errorf("Expected error when no question argument provided, got nil")
	}
}

func TestRunAsk_Offline(t *testing.T) {
	examplesPath := filepath.Join("..", "..", "skills", "okf", "examples")

	err := cli.RunAsk([]string{"--bundle", examplesPath, "How is MRR calculated?"})
	if err != nil {
		t.Fatalf("RunAsk failed in offline mode: %v", err)
	}
}
