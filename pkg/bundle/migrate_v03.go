package bundle

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/abcubed3/okf/pkg/internal/links"
	"gopkg.in/yaml.v3"
)

// FileMigration describes changes applied (or planned under dryRun) to a single bundle file.
type FileMigration struct {
	Path    string
	Actions []string
}

// MigrationReport summarizes an OKF v0.3 in-place bundle migration.
type MigrationReport struct {
	DryRun  bool
	Files   []FileMigration
	Updated int
}

var (
	mdInlineLinkRe   = regexp.MustCompile(`(\[[^\]]*\]\()([^)\s]+)(\))`)
	citationTargetRe = regexp.MustCompile(`^(\[\d+\]\s+)(\S+)(\s*)$`)
)

// MigrateBundleToV03 upgrades an OKF v0.1/v0.2 bundle to OKF v0.3 conformance.
// When dryRun is true, it mutates only in-memory copies to compute the MigrationReport without touching disk.
func MigrateBundleToV03(b *Bundle, dryRun bool) (*MigrationReport, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle cannot be nil")
	}
	report := &MigrationReport{DryRun: dryRun}

	var absRoot, realRoot string
	if strings.TrimSpace(b.Path) != "" {
		var err error
		absRoot, err = filepath.Abs(b.Path)
		if err != nil {
			return nil, err
		}
		realRoot, err = filepath.EvalSymlinks(absRoot)
		if err != nil {
			return nil, err
		}
	}

	writeDoc := func(relSlash string, c *Concept) error {
		if absRoot != "" {
			// Validate symlink-safe write containment even in dryRun so unsafe symlinks are caught early
			targetPath, err := ensureSafeWriteTarget(realRoot, absRoot, relSlash, !dryRun)
			if err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			content, err := FormatConceptMarkdown(c)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, []byte(content), 0o644)
		}
		return nil
	}

	// 1. Migrate root and nested index.md files
	indexKeys := make([]string, 0, len(b.IndexFiles))
	for k := range b.IndexFiles {
		indexKeys = append(indexKeys, k)
	}
	slices.Sort(indexKeys)

	for _, relSlash := range indexKeys {
		orig := b.IndexFiles[relSlash]
		if orig == nil || orig.ParseError != "" {
			continue
		}
		c := copyConcept(orig)
		var actions []string

		if relSlash == "index.md" {
			if c.Frontmatter.Type != "" {
				c.Frontmatter.Type = ""
				actions = append(actions, "removed non-conformant 'type' field from root index.md")
			}
			if c.Frontmatter.OKFVersion != "0.3" {
				prev := c.Frontmatter.OKFVersion
				if prev == "" {
					prev = "unversioned"
				}
				c.Frontmatter.OKFVersion = "0.3"
				c.HasFrontmatter = true
				actions = append(actions, fmt.Sprintf("set okf_version: \"0.3\" (was %s)", prev))
			}
		} else if c.HasFrontmatter || !c.Frontmatter.IsZero() {
			c.HasFrontmatter = false
			c.Frontmatter = Frontmatter{}
			actions = append(actions, "stripped forbidden YAML frontmatter from nested index.md")
		}

		if newBody, bodyRewrites := rewriteBodyBareRootLinks(b.Path, relSlash, c.Body); bodyRewrites > 0 {
			c.Body = newBody
			actions = append(actions, fmt.Sprintf("anchored %d bare root-relative markdown link(s) with leading '/'", bodyRewrites))
		}

		if len(actions) > 0 {
			if err := writeDoc(relSlash, c); err != nil {
				return nil, err
			}
			if !dryRun {
				b.IndexFiles[relSlash] = c
				if relSlash == "index.md" {
					b.OKFVersion = "0.3"
				}
			}
			report.Files = append(report.Files, FileMigration{Path: relSlash, Actions: actions})
			report.Updated++
		}
	}

	// 2. Migrate log.md files (strip forbidden frontmatter & anchor root-relative links)
	logKeys := make([]string, 0, len(b.LogFiles))
	for k := range b.LogFiles {
		logKeys = append(logKeys, k)
	}
	slices.Sort(logKeys)

	for _, relSlash := range logKeys {
		orig := b.LogFiles[relSlash]
		if orig == nil || orig.ParseError != "" {
			continue
		}
		c := copyConcept(orig)
		var actions []string
		if orig.HasFrontmatter || !orig.Frontmatter.IsZero() {
			c.HasFrontmatter = false
			c.Frontmatter = Frontmatter{}
			actions = append(actions, "stripped forbidden YAML frontmatter from log.md")
		}
		if newBody, bodyRewrites := rewriteBodyBareRootLinks(b.Path, relSlash, c.Body); bodyRewrites > 0 {
			c.Body = newBody
			actions = append(actions, fmt.Sprintf("anchored %d bare root-relative markdown link(s) with leading '/'", bodyRewrites))
		}
		if len(actions) > 0 {
			if err := writeDoc(relSlash, c); err != nil {
				return nil, err
			}
			if !dryRun {
				b.LogFiles[relSlash] = c
			}
			report.Files = append(report.Files, FileMigration{Path: relSlash, Actions: actions})
			report.Updated++
		}
	}

	// 3. Migrate concept documents
	conceptIDs := make([]string, 0, len(b.Concepts))
	for id := range b.Concepts {
		conceptIDs = append(conceptIDs, id)
	}
	slices.Sort(conceptIDs)

	for _, id := range conceptIDs {
		orig := b.Concepts[id]
		if orig == nil || orig.ParseError != "" {
			continue
		}
		c := copyConcept(orig)
		var actions []string

		// 3a. Migrate v0.1 timestamp -> generated.at
		if strings.TrimSpace(c.Frontmatter.Timestamp) != "" {
			normTs := normalizeTimestampToRFC3339(c.Frontmatter.Timestamp)
			if c.Frontmatter.Generated == nil {
				c.Frontmatter.Generated = &ActorInfo{
					By: "process:okf-migrate",
					At: normTs,
				}
				actions = append(actions, "migrated legacy 'timestamp' to 'generated'")
			} else if strings.TrimSpace(c.Frontmatter.Generated.At) == "" {
				c.Frontmatter.Generated.At = normTs
				actions = append(actions, "copied legacy 'timestamp' to 'generated.at' and removed 'timestamp'")
			} else {
				actions = append(actions, "removed redundant legacy 'timestamp'")
			}
			c.Frontmatter.Timestamp = ""
		}

		// 3b. Normalize date-only timestamps to RFC3339 UTC
		if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.StaleAfter); changed {
			c.Frontmatter.StaleAfter = norm
			actions = append(actions, "normalized 'stale_after' to RFC3339 UTC")
		}
		if c.Frontmatter.Generated != nil {
			if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.Generated.At); changed {
				c.Frontmatter.Generated.At = norm
				actions = append(actions, "normalized 'generated.at' to RFC3339 UTC")
			}
		}
		for i := range c.Frontmatter.Verified {
			if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.Verified[i].At); changed {
				c.Frontmatter.Verified[i].At = norm
				actions = append(actions, fmt.Sprintf("normalized 'verified[%d].at' to RFC3339 UTC", i))
			}
		}
		if c.Frontmatter.UsageWindow != nil {
			if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.UsageWindow.From); changed {
				c.Frontmatter.UsageWindow.From = norm
				actions = append(actions, "normalized 'usage_window.from' to RFC3339 UTC")
			}
			if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.UsageWindow.To); changed {
				c.Frontmatter.UsageWindow.To = norm
				actions = append(actions, "normalized 'usage_window.to' to RFC3339 UTC")
			}
		}
		for i := range c.Frontmatter.Sources {
			if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.Sources[i].LastModified); changed {
				c.Frontmatter.Sources[i].LastModified = norm
				actions = append(actions, fmt.Sprintf("normalized 'sources[%d].last_modified' to RFC3339 UTC", i))
			}
			if c.Frontmatter.Sources[i].UsageWindow != nil {
				if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.Sources[i].UsageWindow.From); changed {
					c.Frontmatter.Sources[i].UsageWindow.From = norm
					actions = append(actions, fmt.Sprintf("normalized 'sources[%d].usage_window.from' to RFC3339 UTC", i))
				}
				if norm, changed := upgradeDateOnlyTimestamp(c.Frontmatter.Sources[i].UsageWindow.To); changed {
					c.Frontmatter.Sources[i].UsageWindow.To = norm
					actions = append(actions, fmt.Sprintf("normalized 'sources[%d].usage_window.to' to RFC3339 UTC", i))
				}
			}
		}

		// 3c. Migrate v0.1 body # Citations -> frontmatter sources when sources is empty
		if len(c.Citations) > 0 && len(c.Frontmatter.Sources) == 0 {
			for _, cite := range c.Citations {
				if strings.TrimSpace(cite.URI) == "" {
					continue
				}
				c.Frontmatter.Sources = append(c.Frontmatter.Sources, Source{
					ID:       fmt.Sprintf("cite-%d", cite.Number),
					Resource: cite.URI,
					Title:    cite.Title,
				})
			}
			if len(c.Frontmatter.Sources) > 0 {
				actions = append(actions, "migrated legacy '# Citations' entries into frontmatter 'sources'")
			}
		}

		// 3d. Detect legacy 'sources[].uri' keys on disk so standalone uri -> resource upgrades get persisted
		relConceptSlash := filepath.ToSlash(c.Path)
		if relConceptSlash == "" {
			relConceptSlash = c.ID + ".md"
		}
		if absRoot != "" && hasLegacySourceURIOnDisk(absRoot, relConceptSlash) {
			actions = append(actions, "migrated legacy 'sources[].uri' field to 'sources[].resource'")
		}

		// 3e. Rewrite bare root-relative paths to leading "/" paths for v0.3 resolution
		rewriteField := func(fieldName string, val *string) {
			if rewritten, ok := rewriteBareRootPath(b.Path, relConceptSlash, *val); ok {
				*val = rewritten
				actions = append(actions, fmt.Sprintf("anchored bare root-relative path in '%s' to %q", fieldName, rewritten))
			}
		}
		rewriteField("resource", &c.Frontmatter.Resource)
		rewriteField("superseded_by", &c.Frontmatter.SupersededBy)
		rewriteField("computation", &c.Frontmatter.Computation)
		if c.Frontmatter.Executor != nil {
			rewriteField("executor.resource", &c.Frontmatter.Executor.Resource)
		}
		if c.Frontmatter.Attester != nil {
			rewriteField("attester.resource", &c.Frontmatter.Attester.Resource)
		}
		for i := range c.Frontmatter.Sources {
			rewriteField(fmt.Sprintf("sources[%d].resource", i), &c.Frontmatter.Sources[i].Resource)
		}

		if newBody, bodyRewrites := rewriteBodyBareRootLinks(b.Path, relConceptSlash, c.Body); bodyRewrites > 0 {
			c.Body = newBody
			actions = append(actions, fmt.Sprintf("anchored %d bare root-relative markdown link(s) with leading '/'", bodyRewrites))
		}
		for i := range c.Citations {
			if rewritten, ok := rewriteBareRootPath(b.Path, relConceptSlash, c.Citations[i].URI); ok {
				c.Citations[i].URI = rewritten
			}
		}

		if len(actions) > 0 {
			c.HasFrontmatter = true
			if err := writeDoc(relConceptSlash, c); err != nil {
				return nil, err
			}
			if !dryRun {
				b.Concepts[id] = c
			}
			report.Files = append(report.Files, FileMigration{Path: relConceptSlash, Actions: actions})
			report.Updated++
		}
	}

	return report, nil
}

func normalizeTimestampToRFC3339(raw string) string {
	s := strings.TrimSpace(raw)
	if IsStrictRFC3339(s) {
		return s
	}
	if t, ok := ParseOKFTimestamp(s); ok {
		return t.UTC().Format("2006-01-02T15:04:05Z")
	}
	return s
}

func upgradeDateOnlyTimestamp(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" || IsStrictRFC3339(s) {
		return raw, false
	}
	if t, ok := ParseOKFTimestamp(s); ok {
		return t.UTC().Format("2006-01-02T15:04:05Z"), true
	}
	return raw, false
}

func rewriteBareRootPath(bundleRoot, containingRelSlash, raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || links.IsSchemeURI(trimmed) || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../") {
		return raw, false
	}
	if strings.TrimSpace(bundleRoot) == "" || path.Dir(containingRelSlash) == "." {
		return raw, false
	}
	parts := strings.SplitN(trimmed, "#", 2)
	base := strings.ReplaceAll(parts[0], "\\", "/")
	if base == "" {
		return raw, false
	}

	// Only rewrite if it fails relative to containing file's dir AND resolves to an existing file at bundle root
	v03Rel, esc03 := links.ResolveBundlePath(bundleRoot, containingRelSlash, base, "0.3")
	v02Rel, esc02 := links.ResolveBundlePath(bundleRoot, containingRelSlash, base, "0.2")
	if !esc03 && !esc02 && v03Rel != v02Rel {
		rewritten := "/" + v02Rel
		if len(parts) == 2 {
			rewritten += "#" + parts[1]
		}
		return rewritten, true
	}
	return raw, false
}

func rewriteBodyBareRootLinks(bundleRoot, containingRelSlash, body string) (string, int) {
	if strings.TrimSpace(bundleRoot) == "" || path.Dir(containingRelSlash) == "." || body == "" {
		return body, 0
	}
	lines := strings.Split(body, "\n")
	fenceChar := byte(0)
	fenceLen := 0
	inCitations := false
	rewrites := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fc, fl := parseFenceDelimiter(trimmed); fc != 0 {
			if fenceChar == 0 {
				fenceChar = fc
				fenceLen = fl
				continue
			}
			if fc == fenceChar && fl >= fenceLen {
				fenceChar = 0
				fenceLen = 0
				continue
			}
		}
		if fenceChar != 0 || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			inCitations = strings.EqualFold(heading, "citations")
		} else if inCitations {
			if sub := citationTargetRe.FindStringSubmatch(trimmed); len(sub) == 4 {
				if rewritten, ok := rewriteBareRootPath(bundleRoot, containingRelSlash, sub[2]); ok {
					leadingWS := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					lines[i] = leadingWS + sub[1] + rewritten + sub[3]
					rewrites++
					continue
				}
			}
		}

		lines[i] = rewriteLinksOutsideCodeSpans(line, func(segment string) string {
			return mdInlineLinkRe.ReplaceAllStringFunc(segment, func(m string) string {
				sub := mdInlineLinkRe.FindStringSubmatch(m)
				if len(sub) != 4 {
					return m
				}
				if rewritten, ok := rewriteBareRootPath(bundleRoot, containingRelSlash, sub[2]); ok {
					rewrites++
					return sub[1] + rewritten + sub[3]
				}
				return m
			})
		})
	}
	if rewrites == 0 {
		return body, 0
	}
	return strings.Join(lines, "\n"), rewrites
}

func parseFenceDelimiter(trimmed string) (byte, int) {
	if len(trimmed) < 3 {
		return 0, 0
	}
	ch := trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, 0
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0
	}
	return ch, n
}

func rewriteLinksOutsideCodeSpans(line string, fn func(string) string) string {
	if !strings.Contains(line, "`") {
		return fn(line)
	}
	var out strings.Builder
	i := 0
	plainStart := 0
	for i < len(line) {
		if line[i] != '`' {
			i++
			continue
		}
		tickStart := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		tickLen := i - tickStart
		closeIdx := findClosingTicks(line, i, tickLen)
		if closeIdx == -1 {
			// Unmatched backticks are literal text in CommonMark
			continue
		}
		out.WriteString(fn(line[plainStart:tickStart]))
		out.WriteString(line[tickStart : closeIdx+tickLen])
		i = closeIdx + tickLen
		plainStart = i
	}
	out.WriteString(fn(line[plainStart:]))
	return out.String()
}

func findClosingTicks(s string, from, tickLen int) int {
	i := from
	for i < len(s) {
		if s[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(s) && s[i] == '`' {
			i++
		}
		if i-start == tickLen {
			return start
		}
	}
	return -1
}

func hasLegacySourceURIOnDisk(absRoot, relSlash string) bool {
	targetPath := filepath.Join(absRoot, filepath.FromSlash(relSlash))
	rel, err := filepath.Rel(absRoot, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	linfo, err := os.Lstat(targetPath)
	if err != nil || linfo.Mode()&os.ModeSymlink != 0 || linfo.IsDir() {
		return false
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 || strings.TrimRight(strings.TrimPrefix(lines[0], "\uFEFF"), " \t\r") != "---" {
		return false
	}
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			endIdx = i
			break
		}
	}
	if endIdx <= 1 {
		return false
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:endIdx], "\n")), &doc); err != nil || len(doc.Content) == 0 {
		return false
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "sources" {
			continue
		}
		val := root.Content[i+1]
		if val.Kind == yaml.MappingNode && mappingHasKey(val, "uri") {
			return true
		}
		if val.Kind == yaml.SequenceNode {
			for _, elem := range val.Content {
				if elem.Kind == yaml.MappingNode && mappingHasKey(elem, "uri") {
					return true
				}
			}
		}
	}
	return false
}

func mappingHasKey(m *yaml.Node, key string) bool {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return true
		}
	}
	return false
}
