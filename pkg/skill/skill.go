// Package skill provides capabilities for compiling OKF bundles into AI Agent Skills (SKILL.md)
// and installing the official OKF Agent Skill into developer workspaces.
package skill

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/skills"
	"gopkg.in/yaml.v3"
)

// ExportOptions controls how an OKF bundle is compiled into an Agent Skill.
type ExportOptions struct {
	// Name is the skill identifier (e.g., "data-warehouse", "finance-metrics").
	Name string
	// Description is the frontmatter description explaining when the agent should activate this skill.
	Description string
	// Tags filters concepts to only those containing at least one matching tag.
	Tags []string
	// Types filters concepts to only those matching one of the given types.
	Types []string
	// IncludeAttestations specifies whether attested computation queries should be highlighted.
	IncludeAttestations bool
}

// SkillFrontmatter defines the YAML frontmatter for a standard Agent Skill.
type SkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ExportBundleToSkill compiles an OKF bundle into an Agent Skill (SKILL.md) content string.
func ExportBundleToSkill(b *bundle.Bundle, opts ExportOptions) (string, error) {
	if b == nil {
		return "", fmt.Errorf("bundle cannot be nil")
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "okf-domain-knowledge"
	}
	// Sanitize name for skill frontmatter (lowercase, alphanumeric, dashes)
	name = strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	desc := strings.TrimSpace(opts.Description)
	if desc == "" {
		desc = fmt.Sprintf("Domain knowledge and verified schemas compiled from the %s OKF bundle. Use when querying database tables, business metrics, or attested calculation logic in this domain.", name)
	}

	filtered := filterConcepts(b.Concepts, opts.Tags, opts.Types)
	if len(filtered) == 0 {
		return "", fmt.Errorf("no concepts matched the specified filters (tags: %v, types: %v)", opts.Tags, opts.Types)
	}

	// Sort concepts deterministically by ID
	conceptIDs := make([]string, 0, len(filtered))
	for id := range filtered {
		conceptIDs = append(conceptIDs, id)
	}
	sort.Strings(conceptIDs)

	var sb strings.Builder

	// 1. YAML Frontmatter
	fm := SkillFrontmatter{
		Name:        name,
		Description: desc,
	}
	fmBytes, err := yaml.Marshal(&fm)
	if err != nil {
		return "", fmt.Errorf("failed to marshal skill frontmatter: %w", err)
	}
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n\n")

	// 2. Title & Overview
	sb.WriteString(fmt.Sprintf("# %s — Agent Skill\n\n", strings.Title(strings.ReplaceAll(name, "-", " "))))
	sb.WriteString("This skill provides verified domain knowledge, catalog schemas, and sanctioned calculation rules compiled directly from the organizational Open Knowledge Format (OKF) bundle.\n\n")

	// 3. Trust & Freshness Directives
	sb.WriteString("## 🛡️ Critical Agent Directives\n\n")
	sb.WriteString("1. **Strict Schema Fidelity:** Only query tables, fields, and API endpoints defined in this skill. Do NOT assume unlisted columns exist.\n")
	sb.WriteString("2. **Sanctioned Computations:** When computing metrics listed in this skill, always use the verbatim SQL queries from the **Attested Computations** section below. Do NOT invent ad-hoc aggregations.\n")
	sb.WriteString("3. **Staleness Awareness:** Check concept freshness timestamps before executing business-critical queries.\n\n")

	// 4. Attested Computations (if present and enabled)
	var attestations []*bundle.Concept
	for _, id := range conceptIDs {
		c := filtered[id]
		if strings.EqualFold(c.Frontmatter.Type, "Attestation") || (c.Frontmatter.Attestation != nil && c.Frontmatter.Attestation.Query != "") {
			attestations = append(attestations, c)
		}
	}

	if opts.IncludeAttestations && len(attestations) > 0 {
		sb.WriteString("## 📐 Sanctioned Computations & Attested Queries\n\n")
		sb.WriteString("The following queries have been formally audited and certified by domain stewards:\n\n")
		for _, a := range attestations {
			title := a.Frontmatter.Title
			if title == "" {
				title = a.ID
			}
			sb.WriteString(fmt.Sprintf("### %s\n", title))
			if a.Frontmatter.Desc != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", a.Frontmatter.Desc))
			}
			if a.Frontmatter.Attestation != nil {
				if a.Frontmatter.Attestation.Executor != "" {
					sb.WriteString(fmt.Sprintf("- **Target Runtime / Engine:** `%s`\n", a.Frontmatter.Attestation.Executor))
				}
				if a.Frontmatter.Attestation.Attester != "" {
					sb.WriteString(fmt.Sprintf("- **Attester:** `%s`\n", a.Frontmatter.Attestation.Attester))
				}
			}
			if len(a.Frontmatter.Verified) > 0 {
				sb.WriteString(fmt.Sprintf("- **Verification Tier:** `%s` (by %s)\n", a.Frontmatter.Verified[0].Tier(), a.Frontmatter.Verified[0].By))
			}
			if a.Frontmatter.Attestation != nil && a.Frontmatter.Attestation.Query != "" {
				sb.WriteString("\n```sql\n")
				query := strings.TrimSpace(a.Frontmatter.Attestation.Query)
				sb.WriteString(query)
				sb.WriteString("\n```\n\n")
			}
		}
	}

	// 5. Concepts Catalog Grouped by Type
	typeGroups := make(map[string][]*bundle.Concept)
	for _, id := range conceptIDs {
		c := filtered[id]
		cType := c.Frontmatter.Type
		if cType == "" {
			cType = "General"
		}
		typeGroups[cType] = append(typeGroups[cType], c)
	}

	var sortedTypes []string
	for t := range typeGroups {
		sortedTypes = append(sortedTypes, t)
	}
	sort.Strings(sortedTypes)

	sb.WriteString("## 📚 Domain Concepts & Schema Catalog\n\n")
	for _, t := range sortedTypes {
		sb.WriteString(fmt.Sprintf("### Type: %s\n\n", t))
		for _, c := range typeGroups[t] {
			title := c.Frontmatter.Title
			if title == "" {
				title = c.ID
			}
			sb.WriteString(fmt.Sprintf("#### %s (`%s`)\n", title, c.ID))
			if c.Frontmatter.Desc != "" {
				sb.WriteString(fmt.Sprintf("- **Description:** %s\n", c.Frontmatter.Desc))
			}
			if c.Frontmatter.Resource != "" {
				sb.WriteString(fmt.Sprintf("- **Resource:** `%s`\n", c.Frontmatter.Resource))
			}
			if len(c.Frontmatter.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("- **Tags:** `%s`\n", strings.Join(c.Frontmatter.Tags, "`, `")))
			}
			if c.Frontmatter.Status != "" {
				sb.WriteString(fmt.Sprintf("- **Lifecycle Status:** `%s`\n", c.Frontmatter.Status))
			}
			if c.Frontmatter.StaleAfter != "" {
				sb.WriteString(fmt.Sprintf("- **Stale After:** `%s`\n", c.Frontmatter.StaleAfter))
			}
			if len(c.Frontmatter.Verified) > 0 {
				sb.WriteString(fmt.Sprintf("- **Trust Status:** Verified (%s by %s)\n", c.Frontmatter.Verified[0].Tier(), c.Frontmatter.Verified[0].By))
			}

			// Include snippet or summary from concept body if helpful
			cleanBody := strings.TrimSpace(c.Body)
			if cleanBody != "" {
				sb.WriteString("\n")
				sb.WriteString(cleanBody)
				sb.WriteString("\n")
			}
			sb.WriteString("\n---\n\n")
		}
	}

	// 6. Footer metadata
	sb.WriteString(fmt.Sprintf("\n*Generated by OKF Agent Skill Engine on %s*\n", time.Now().UTC().Format(time.RFC3339)))

	return sb.String(), nil
}

// filterConcepts returns only concepts matching the specified tags and types.
func filterConcepts(concepts map[string]*bundle.Concept, tags, types []string) map[string]*bundle.Concept {
	tagMap := make(map[string]bool)
	for _, t := range tags {
		tagMap[strings.ToLower(strings.TrimSpace(t))] = true
	}

	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[strings.ToLower(strings.TrimSpace(t))] = true
	}

	filtered := make(map[string]*bundle.Concept)
	for id, c := range concepts {
		// Filter by type if types specified
		if len(typeMap) > 0 {
			if !typeMap[strings.ToLower(c.Frontmatter.Type)] {
				continue
			}
		}

		// Filter by tag if tags specified
		if len(tagMap) > 0 {
			matchedTag := false
			for _, tag := range c.Frontmatter.Tags {
				if tagMap[strings.ToLower(tag)] {
					matchedTag = true
					break
				}
			}
			if !matchedTag {
				continue
			}
		}

		filtered[id] = c
	}
	return filtered
}

// InstallOptions defines destination and platform options for installing the official OKF skill.
type InstallOptions struct {
	// TargetDir is the explicit root or skill directory to install to.
	TargetDir string
	// Platform specifies the target agent harness ("antigravity", "claude", "cursor", "workspace").
	Platform string
}

// ResolveSkillInstallPath determines the destination directory for the skill based on platform.
func ResolveSkillInstallPath(opts InstallOptions) string {
	if opts.TargetDir != "" {
		return opts.TargetDir
	}

	platform := strings.ToLower(strings.TrimSpace(opts.Platform))
	switch platform {
	case "claude", "claude-code":
		return filepath.Join(".claude", "skills", "okf")
	case "cursor":
		return filepath.Join(".cursor", "skills", "okf")
	case "antigravity", "agy", "agents", "":
		return filepath.Join(".agents", "skills", "okf")
	default:
		return filepath.Join(".agents", "skills", "okf")
	}
}

// InstallOfficialSkill writes the official OKF agent skill into the target directory.
func InstallOfficialSkill(opts InstallOptions) (string, error) {
	destDir := ResolveSkillInstallPath(opts)
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %s: %w", destDir, err)
	}

	err = skills.CopySkill(skills.OKF, absDest)

	return absDest, err
}
