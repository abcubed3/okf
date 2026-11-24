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

func TestRunLint_MigrateV03AndDryRun(t *testing.T) {
	tmp := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.md", "---\nokf_version: \"0.2\"\ntype: \"Index\"\ntitle: \"Legacy Bundle\"\n---\n# Legacy Bundle\n- [Revenue](metrics/rev.md)\n")
	write("README.md", "# Repo README\n")
	write("references/policy.md", "---\ntype: Reference\ntitle: Policy\ndescription: Revenue policy.\n---\n# Policy\n")
	write("metrics/rev.md", `---
type: Metric
title: Revenue
description: Monthly revenue.
timestamp: 2026-07-01
sources:
  - uri: references/policy.md
    title: Policy Doc
x-custom-team: finance
---
# Revenue
See [Policy](references/policy.md).
`)

	// 1. Dry-run must not mutate files on disk
	if err := RunLint([]string{"--migrate-v03", "--dry-run", tmp}); err != nil {
		t.Fatalf("dry-run migrate failed: %v", err)
	}
	rawIdxBefore, _ := os.ReadFile(filepath.Join(tmp, "index.md"))
	if !strings.Contains(string(rawIdxBefore), `okf_version: "0.2"`) {
		t.Fatalf("dry-run unexpectedly modified index.md:\n%s", string(rawIdxBefore))
	}

	// 2. Real --migrate-v03 --strict must migrate in-place and pass strict v0.3 lint with 0 warnings/errors!
	if err := RunLint([]string{"--migrate-v03", "--strict", tmp}); err != nil {
		t.Fatalf("RunLint --migrate-v03 --strict failed: %v", err)
	}

	rawIdxAfter, _ := os.ReadFile(filepath.Join(tmp, "index.md"))
	if !strings.Contains(string(rawIdxAfter), `okf_version: "0.3"`) || strings.Contains(string(rawIdxAfter), "type:") {
		t.Errorf("expected migrated index.md to have okf_version 0.3 and no type, got:\n%s", string(rawIdxAfter))
	}

	rawRevAfter, _ := os.ReadFile(filepath.Join(tmp, "metrics", "rev.md"))
	revStr := string(rawRevAfter)
	for _, want := range []string{
		"generated:",
		"2026-07-01T00:00:00Z",
		"resource: /references/policy.md",
		"x-custom-team: finance",
		"[Policy](/references/policy.md)",
	} {
		if !strings.Contains(revStr, want) {
			t.Errorf("migrated metrics/rev.md missing %q:\n%s", want, revStr)
		}
	}
	if strings.Contains(revStr, "timestamp:") || strings.Contains(revStr, "uri:") {
		t.Errorf("migrated metrics/rev.md retained legacy timestamp/uri:\n%s", revStr)
	}
}
