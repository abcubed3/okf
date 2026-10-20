package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/internal/cli"
)

func TestRunSkill_Help(t *testing.T) {
	err := cli.RunSkill([]string{})
	if err == nil {
		t.Errorf("Expected error when no subcommand provided, got nil")
	}

	err = cli.RunSkill([]string{"unknown"})
	if err == nil {
		t.Errorf("Expected error for unknown subcommand, got nil")
	}
}

func TestRunSkill_ExportAndInstall(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Test skill install
	targetSkillDir := filepath.Join(tmpDir, "installed-skill")
	err := cli.RunSkill([]string{"install", "--dir", targetSkillDir})
	if err != nil {
		t.Fatalf("cli.RunSkill('install') failed: %v", err)
	}

	skillFile := filepath.Join(targetSkillDir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("Failed to read installed SKILL.md: %v", err)
	}
	if !strings.Contains(string(data), "name: okf") {
		t.Errorf("Expected installed SKILL.md to contain 'name: okf'")
	}

	// 2. Test skill export on the examples bundle
	examplesPath := filepath.Join("..", "..", "skills", "okf", "examples")
	exportedFile := filepath.Join(tmpDir, "exported_skill.md")

	err = cli.RunSkill([]string{"export", "--bundle", examplesPath, "--output", exportedFile, "--name", "example-skill"})
	if err != nil {
		t.Fatalf("cli.RunSkill('export') failed: %v", err)
	}

	expData, err := os.ReadFile(exportedFile)
	if err != nil {
		t.Fatalf("Failed to read exported skill file: %v", err)
	}

	expStr := string(expData)
	if !strings.Contains(expStr, "name: example-skill") {
		t.Errorf("Expected exported skill to contain 'name: example-skill', got:\n%s", expStr)
	}
	if !strings.Contains(expStr, "Monthly Recurring Revenue") {
		t.Errorf("Expected exported skill to contain 'Monthly Recurring Revenue'")
	}
	if !strings.Contains(expStr, "Sanctioned MRR Calculation Query") {
		t.Errorf("Expected exported skill to contain attested query")
	}
}

func TestRunExport_Skill(t *testing.T) {
	tmpDir := t.TempDir()
	examplesPath := filepath.Join("..", "..", "skills", "okf", "examples")
	exportedFile := filepath.Join(tmpDir, "unified_export.md")

	err := cli.RunExport([]string{"skill", "--bundle", examplesPath, "--output", exportedFile, "--name", "unified-skill"})
	if err != nil {
		t.Fatalf("cli.RunExport('skill') failed: %v", err)
	}

	data, err := os.ReadFile(exportedFile)
	if err != nil {
		t.Fatalf("Failed to read exported skill file: %v", err)
	}
	if !strings.Contains(string(data), "name: unified-skill") {
		t.Errorf("Expected unified export to contain 'name: unified-skill'")
	}
}
