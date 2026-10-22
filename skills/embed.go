// Package skills provides functionality to read and copy embedded skill files.
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed *
var skillFS embed.FS

type Skill string

const (
	OKF Skill = "okf"
)

// ReadSkill reads the embedded skill's SKILL.md file.
func ReadSkill(name Skill) ([]byte, error) {
	if err := validateName(string(name)); err != nil {
		return nil, err
	}

	content, err := fs.ReadFile(skillFS, path.Join(string(name), "SKILL.md"))
	if err != nil {
		return nil, fmt.Errorf("read skill %q: %w", name, err)
	}
	return content, nil
}

// CopySkill copies the embedded skill directory, including references and examples.
func CopySkill(name Skill, destination string) error {
	if err := validateName(string(name)); err != nil {
		return err
	}

	src, err := fs.Sub(skillFS, string(name))
	if err != nil {
		return fmt.Errorf("open embedded skill %q: %w", name, err)
	}

	return fs.WalkDir(src, ".", func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		target := filepath.Join(destination, filepath.FromSlash(file))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := fs.ReadFile(src, file)
		if err != nil {
			return fmt.Errorf("read embedded file %q: %w", file, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("create directory for %q: %w", target, err)
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return fmt.Errorf("write %q: %w", target, err)
		}
		return nil
	})
}

func validateName(name string) error {
	if !fs.ValidPath(name) || strings.Contains(name, "/") {
		return fmt.Errorf("invalid skill name %q", name)
	}
	return nil
}
