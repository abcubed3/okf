package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/skills"
)

func TestReadSkill_Success(t *testing.T) {
	data, err := skills.ReadSkill(skills.OKF)
	if err != nil {
		t.Fatalf("ReadSkill(OKF) failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: okf") {
		t.Errorf("Expected SKILL.md to contain 'name: okf'")
	}
	if !strings.Contains(content, "Open Knowledge Format") {
		t.Errorf("Expected SKILL.md to contain 'Open Knowledge Format'")
	}
}

func TestReadSkill_NotFound(t *testing.T) {
	_, err := skills.ReadSkill("nonexistent")
	if err == nil {
		t.Errorf("Expected error for non-existent skill, got nil")
	}
}

func TestReadSkill_InvalidName(t *testing.T) {
	testCases := []string{
		"../escape",
		"okf/sub",
		"/root",
		"",
	}

	for _, name := range testCases {
		_, err := skills.ReadSkill(skills.Skill(name))
		if err == nil {
			t.Errorf("Expected error for invalid skill name %q, got nil", name)
		}
	}
}

func TestCopySkill_Success(t *testing.T) {
	tmpDir := t.TempDir()

	err := skills.CopySkill(skills.OKF, tmpDir)
	if err != nil {
		t.Fatalf("CopySkill(OKF) failed: %v", err)
	}

	// Verify root SKILL.md
	skillPath := filepath.Join(tmpDir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read copied SKILL.md: %v", err)
	}
	if !strings.Contains(string(data), "name: okf") {
		t.Errorf("Expected SKILL.md to contain 'name: okf'")
	}

	// Verify references
	refPath := filepath.Join(tmpDir, "references", "spec-v02.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Errorf("Expected references/spec-v02.md to exist: %v", err)
	}

	// Verify examples
	examplePath := filepath.Join(tmpDir, "examples", "attestations", "mrr_query.md")
	if _, err := os.Stat(examplePath); err != nil {
		t.Errorf("Expected examples/attestations/mrr_query.md to exist: %v", err)
	}
}

func TestCopySkill_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()

	if err := skills.CopySkill("../escape", tmpDir); err == nil {
		t.Errorf("Expected error for traversal skill name, got nil")
	}

	if err := skills.CopySkill("nonexistent", tmpDir); err == nil {
		t.Errorf("Expected error for non-existent skill, got nil")
	}
}
