package harvester

import (
	"context"
	"os"
	"testing"
)

func TestGitHarvester_Harvest(t *testing.T) {
	// The okf directory contains the .git folder, so from pkg/harvester it is ../..
	repoPath := "../.."

	// Verify we can find a git repository at repoPath
	if _, err := os.Stat(repoPath + "/.git"); os.IsNotExist(err) {
		t.Skip("Skipping test: repoPath is not inside a git repository")
	}

	gh := NewGitHarvester(repoPath)
	concepts, err := gh.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	if len(concepts) == 0 {
		t.Errorf("Expected to harvest concepts, got 0")
	}

	var hasRepo, hasCommit, hasContributor bool
	for _, c := range concepts {
		if c.Frontmatter.Type == "Git Repository" {
			hasRepo = true
		}
		if c.Frontmatter.Type == "Git Commit" {
			hasCommit = true
		}
		if c.Frontmatter.Type == "Git Contributor" {
			hasContributor = true
		}
	}

	if !hasRepo {
		t.Errorf("Expected to harvest a Git Repository concept")
	}
	if !hasCommit {
		t.Errorf("Expected to harvest Git Commit concepts")
	}
	if !hasContributor {
		t.Errorf("Expected to harvest Git Contributor concepts")
	}
}
