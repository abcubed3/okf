// Package validator checks OKF bundles for schema completeness, broken internal links, and general spec conformance.
package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/abcubed3/okf/pkg/internal/links"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var (
	sharedMarkdown  = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))
	okfVersionRegex = regexp.MustCompile(`^\d+\.\d+$`)
	isoDateRegex    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	footnoteRefRe   = regexp.MustCompile(`\[\^([^\s\]]+)\]`)
)

// IssueSeverity indicates whether the finding is a hard error or a soft warning.
type IssueSeverity string

const (
	// SeverityError indicates a conformance error that causes validation failure.
	SeverityError IssueSeverity = "ERROR"
	// SeverityWarning indicates a warning or recommendation violation that does not fail build linting.
	SeverityWarning IssueSeverity = "WARNING"
)

// Issue represents a validation finding.
type Issue struct {
	// ConceptID is the identifier of the concept where the issue was found.
	ConceptID string
	// Path is the file path to the concept document.
	Path string
	// Severity designates the issue as an Error or a Warning.
	Severity IssueSeverity
	// Message is the detailed description of the validation issue.
	Message string
}

// Options configure the validator.
type Options struct {
	Host       string
	APIKey     string
	HTTPClient *http.Client
}

// remoteBundleEntry caches the result of fetching a remote bundle's concepts,
// distinguishing unreachable/auth failures from confirmed empty or 404 responses.
type remoteBundleEntry struct {
	concepts    map[string]bool
	unreachable bool
	errMsg      string
}

// remoteCache stores fetched concepts per bundle ID to avoid repeated network calls.
type remoteCache map[string]remoteBundleEntry

// bundleRoots caches resolved lexical and symlink-evaluated bundle root paths across a validation pass.
type bundleRoots struct {
	hasPath  bool
	absRoot  string
	absErr   error
	realRoot string
	realErr  error
}

func resolveBundleRoots(b *bundle.Bundle) bundleRoots {
	if b == nil || strings.TrimSpace(b.Path) == "" {
		return bundleRoots{}
	}
	absRoot, absErr := filepath.Abs(b.Path)
	if absErr != nil {
		return bundleRoots{hasPath: true, absErr: absErr}
	}
	realRoot, realErr := filepath.EvalSymlinks(absRoot)
	return bundleRoots{
		hasPath:  true,
		absRoot:  absRoot,
		realRoot: realRoot,
		realErr:  realErr,
	}
}

// ValidateBundle performs OKF conformance checks on a bundle, verifying frontmatter structures,
// reserved index.md / log.md rules, and internal/remote links.
func ValidateBundle(b *bundle.Bundle, opts Options) []Issue {
	var issues []Issue
	rcache := make(remoteCache)
	roots := resolveBundleRoots(b)
	for _, c := range b.Concepts {
		issues = append(issues, validateConceptWithRoots(c, b, opts, rcache, roots)...)
	}
	for relSlash, idx := range b.IndexFiles {
		issues = append(issues, validateIndexFileWithRoots(relSlash, idx, b, opts, rcache, roots)...)
	}
	for relSlash, lg := range b.LogFiles {
		issues = append(issues, validateLogFileWithRoots(relSlash, lg, b, opts, rcache, roots)...)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Message < issues[j].Message
	})

	return issues
}

// ValidateIndexFile validates a reserved index.md file against OKF Spec v0.3 §8 and §12.
func ValidateIndexFile(relSlash string, idx *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	return validateIndexFileWithRoots(relSlash, idx, b, opts, rcache, resolveBundleRoots(b))
}

func validateIndexFileWithRoots(relSlash string, idx *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache, roots bundleRoots) []Issue {
	if idx == nil {
		return nil
	}
	if rcache == nil {
		rcache = make(remoteCache)
	}
	var issues []Issue
	if idx.ParseError != "" {
		return []Issue{{
			ConceptID: idx.ID,
			Path:      idx.Path,
			Severity:  SeverityError,
			Message:   idx.ParseError,
		}}
	}

	normRel := filepath.ToSlash(relSlash)
	if normRel == "" {
		normRel = filepath.ToSlash(idx.Path)
	}

	if normRel == "index.md" {
		if strings.TrimSpace(idx.Frontmatter.Type) != "" {
			sev := SeverityError
			if links.IsLegacyVersion(idx.Frontmatter.OKFVersion) {
				sev = SeverityWarning
			}
			issues = append(issues, Issue{
				ConceptID: idx.ID,
				Path:      idx.Path,
				Severity:  sev,
				Message:   "bundle-root index.md is a reserved index file and must not declare a 'type' field",
			})
		}
		if ver := strings.TrimSpace(idx.Frontmatter.OKFVersion); ver != "" && !okfVersionRegex.MatchString(ver) {
			issues = append(issues, Issue{
				ConceptID: idx.ID,
				Path:      idx.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("invalid 'okf_version' format %q (expected '<major>.<minor>', e.g. \"0.3\")", ver),
			})
		}
	} else if idx.HasFrontmatter || !idx.Frontmatter.IsZero() {
		issues = append(issues, Issue{
			ConceptID: idx.ID,
			Path:      idx.Path,
			Severity:  SeverityError,
			Message:   "nested index.md must not contain YAML frontmatter",
		})
	}

	issues = append(issues, validateLinksWithRoots(idx, b, opts, rcache, roots)...)
	issues = append(issues, validateCitationsWithRoots(idx, b, opts, rcache, roots)...)
	return issues
}

// ValidateLogFile validates a reserved log.md file against OKF Spec v0.3 §9.
func ValidateLogFile(relSlash string, lg *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	return validateLogFileWithRoots(relSlash, lg, b, opts, rcache, resolveBundleRoots(b))
}

func validateLogFileWithRoots(relSlash string, lg *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache, roots bundleRoots) []Issue {
	if lg == nil {
		return nil
	}
	if rcache == nil {
		rcache = make(remoteCache)
	}
	var issues []Issue
	if lg.ParseError != "" {
		return []Issue{{
			ConceptID: lg.ID,
			Path:      lg.Path,
			Severity:  SeverityError,
			Message:   lg.ParseError,
		}}
	}

	if lg.HasFrontmatter || !lg.Frontmatter.IsZero() {
		issues = append(issues, Issue{
			ConceptID: lg.ID,
			Path:      lg.Path,
			Severity:  SeverityError,
			Message:   "log.md must not contain YAML frontmatter",
		})
	}

	srcBytes := []byte(lg.Body)
	doc := sharedMarkdown.Parser().Parse(text.NewReader(srcBytes))
	var prevDate time.Time
	prevHeading := ""
	hasPrevDate := false

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Level != 2 {
			return ast.WalkContinue, nil
		}
		heading := strings.TrimSpace(extractPlainHeadingText(h, srcBytes))
		if !isoDateRegex.MatchString(heading) {
			issues = append(issues, Issue{
				ConceptID: lg.ID,
				Path:      lg.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("log.md date heading '## %s' should use ISO 8601 YYYY-MM-DD format", heading),
			})
			return ast.WalkSkipChildren, nil
		}
		parsedDate, err := time.Parse("2006-01-02", heading)
		if err != nil {
			issues = append(issues, Issue{
				ConceptID: lg.ID,
				Path:      lg.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("log.md date heading '## %s' is not a valid calendar date", heading),
			})
			return ast.WalkSkipChildren, nil
		}
		if hasPrevDate && parsedDate.After(prevDate) {
			issues = append(issues, Issue{
				ConceptID: lg.ID,
				Path:      lg.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("log.md date heading '## %s' should be in reverse chronological order (appears after '## %s')", heading, prevHeading),
			})
		}
		prevDate = parsedDate
		prevHeading = heading
		hasPrevDate = true
		return ast.WalkSkipChildren, nil
	})

	issues = append(issues, validateLinksWithRoots(lg, b, opts, rcache, roots)...)
	issues = append(issues, validateCitationsWithRoots(lg, b, opts, rcache, roots)...)
	return issues
}

func extractPlainHeadingText(h *ast.Heading, src []byte) string {
	var buf strings.Builder
	for i := 0; i < h.Lines().Len(); i++ {
		seg := h.Lines().At(i)
		buf.Write(seg.Value(src))
	}
	return buf.String()
}

// ValidateConcept performs OKF conformance checks on a single concept against the bundle.
func ValidateConcept(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache) []Issue {
	return validateConceptWithRoots(c, b, opts, rcache, resolveBundleRoots(b))
}

func validateConceptWithRoots(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache, roots bundleRoots) []Issue {
	var issues []Issue
	if rcache == nil {
		rcache = make(remoteCache)
	}

	if c.ParseError != "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   c.ParseError,
		})
		return issues
	}

	// 1. Hard Rule: Must have non-empty type
	if strings.TrimSpace(c.Frontmatter.Type) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   "missing or empty 'type' field in frontmatter",
		})
	}

	// 2. Soft Rules: Recommended fields
	if strings.TrimSpace(c.Frontmatter.Title) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   "missing recommended 'title' field",
		})
	}
	if strings.TrimSpace(c.Frontmatter.Desc) == "" {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   "missing recommended 'description' field",
		})
	}

	issues = append(issues, validateFrontmatterPathsWithRoots(c, b, roots)...)

	// 3. Link validation
	linkIssues := validateLinksWithRoots(c, b, opts, rcache, roots)
	issues = append(issues, linkIssues...)

	// 4. Citation validation
	citationIssues := validateCitationsWithRoots(c, b, opts, rcache, roots)
	issues = append(issues, citationIssues...)

	// 5. OKF v0.2 / v0.3 Trust Signals, Provenance, Lifecycle & Attestation validation
	trustIssues := validateTrustSignals(c, b)
	issues = append(issues, trustIssues...)

	return issues
}

func validateTimestampFormat(c *bundle.Concept, fieldName, rawVal string) (time.Time, bool, []Issue) {
	trimmed := strings.TrimSpace(rawVal)
	if trimmed == "" {
		return time.Time{}, false, nil
	}
	if bundle.IsStrictRFC3339(trimmed) {
		t, _ := bundle.ParseOKFTimestamp(trimmed)
		return t, true, nil
	}
	if t, ok := bundle.ParseOKFTimestamp(trimmed); ok {
		return t, true, []Issue{{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("field '%s' value '%s' is missing time and UTC offset (expected RFC3339 datetime, e.g. 2026-06-30T14:00:00Z)", fieldName, trimmed),
		}}
	}
	return time.Time{}, false, []Issue{{
		ConceptID: c.ID,
		Path:      c.Path,
		Severity:  SeverityWarning,
		Message:   fmt.Sprintf("invalid '%s' date format '%s' (expected RFC3339 datetime with UTC offset)", fieldName, trimmed),
	}}
}

// validateTrustSignals verifies OKF v0.2/v0.3 provenance, trust tiers, freshness, lifecycle, and attestation metadata.
func validateTrustSignals(c *bundle.Concept, b *bundle.Bundle) []Issue {
	var issues []Issue

	// 1. Lifecycle status & superseded_by check (§5.4)
	supersededBy := strings.TrimSpace(c.Frontmatter.SupersededBy)
	st := bundle.ConceptLifecycleStatus(strings.ToLower(strings.TrimSpace(string(c.Frontmatter.Status))))
	if c.Frontmatter.Status != "" {
		switch st {
		case bundle.ConceptLifecycleDraft, bundle.ConceptLifecycleStable:
			// Valid standard statuses
		case bundle.ConceptLifecycleDeprecated:
			msg := "concept is marked as deprecated"
			if supersededBy != "" {
				msg = fmt.Sprintf("concept is marked as deprecated (superseded by '%s')", supersededBy)
			}
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   msg,
			})
		default:
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("unrecognized lifecycle status '%s' (expected %s, %s, or %s)", c.Frontmatter.Status, bundle.ConceptLifecycleDraft, bundle.ConceptLifecycleStable, bundle.ConceptLifecycleDeprecated),
			})
		}
	}
	if supersededBy != "" {
		if st != bundle.ConceptLifecycleDeprecated {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   "'superseded_by' is set on a non-deprecated concept (expected status: 'deprecated')",
			})
		}
		if !links.IsSchemeURI(supersededBy) {
			bundleRoot := ""
			okfVer := ""
			maxHops := 8
			if b != nil {
				bundleRoot = b.Path
				okfVer = b.OKFVersion
				if len(b.Concepts)+1 > maxHops {
					maxHops = len(b.Concepts) + 1
				}
			}
			if resolved, escaped := links.ResolveBundlePath(bundleRoot, c.Path, supersededBy, okfVer); !escaped {
				targetID := strings.TrimSuffix(resolved, ".md")
				if targetID == c.ID || resolved == filepath.ToSlash(c.Path) {
					issues = append(issues, Issue{
						ConceptID: c.ID,
						Path:      c.Path,
						Severity:  SeverityWarning,
						Message:   "'superseded_by' cannot reference the concept itself",
					})
				} else if b != nil {
					visited := map[string]bool{c.ID: true}
					nextID := targetID
					for range maxHops {
						if visited[nextID] {
							issues = append(issues, Issue{
								ConceptID: c.ID,
								Path:      c.Path,
								Severity:  SeverityWarning,
								Message:   fmt.Sprintf("'superseded_by' chain forms a cycle involving '%s'", nextID),
							})
							break
						}
						visited[nextID] = true
						nextC, exists := b.GetConcept(nextID)
						if !exists || nextC == nil {
							break
						}
						nextSup := strings.TrimSpace(nextC.Frontmatter.SupersededBy)
						if nextSup == "" || links.IsSchemeURI(nextSup) {
							break
						}
						nextResolved, nextEsc := links.ResolveBundlePath(bundleRoot, nextC.Path, nextSup, okfVer)
						if nextEsc || nextResolved == "" {
							break
						}
						nextID = strings.TrimSuffix(nextResolved, ".md")
					}
				}
			}
		}
	}

	// 2. Freshness (stale_after) check (§5.5)
	if c.Frontmatter.StaleAfter != "" {
		staleTime, ok, tsIssues := validateTimestampFormat(c, "stale_after", c.Frontmatter.StaleAfter)
		issues = append(issues, tsIssues...)
		if ok && !time.Now().Before(staleTime) {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("concept is stale (stale_after date '%s' has passed)", strings.TrimSpace(c.Frontmatter.StaleAfter)),
			})
		}
	}

	// 3. Provenance sources & usage_window check (§5.1)
	sourceIDs := make(map[string]bool)
	for i, src := range c.Frontmatter.Sources {
		if strings.TrimSpace(src.Resource) == "" {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityError,
				Message:   fmt.Sprintf("sources[%d] is missing required 'resource' field", i),
			})
		}
		if id := strings.TrimSpace(src.ID); id != "" {
			if sourceIDs[id] {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityWarning,
					Message:   fmt.Sprintf("duplicate 'sources[].id' value %q", id),
				})
			}
			sourceIDs[id] = true
		}
		if src.LastModified != "" {
			_, _, tsIssues := validateTimestampFormat(c, fmt.Sprintf("sources[%d].last_modified", i), src.LastModified)
			issues = append(issues, tsIssues...)
		}
		if src.UsageWindow != nil {
			_, _, fromIssues := validateTimestampFormat(c, fmt.Sprintf("sources[%d].usage_window.from", i), src.UsageWindow.From)
			_, _, toIssues := validateTimestampFormat(c, fmt.Sprintf("sources[%d].usage_window.to", i), src.UsageWindow.To)
			issues = append(issues, fromIssues...)
			issues = append(issues, toIssues...)
		}
	}
	if c.Frontmatter.UsageWindow != nil {
		_, _, fromIssues := validateTimestampFormat(c, "usage_window.from", c.Frontmatter.UsageWindow.From)
		_, _, toIssues := validateTimestampFormat(c, "usage_window.to", c.Frontmatter.UsageWindow.To)
		issues = append(issues, fromIssues...)
		issues = append(issues, toIssues...)
	}
	for _, label := range extractFootnoteLabels(c.Body) {
		if !sourceIDs[label] {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("footnote '[^%s]' does not match any 'sources[].id' entry", label),
			})
		}
	}

	// 4. Generated timestamp and actor check (§5.2)
	if c.Frontmatter.Generated != nil {
		if strings.TrimSpace(c.Frontmatter.Generated.By) == "" {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   "generated frontmatter block is missing 'by' actor identifier",
			})
		}
		if strings.TrimSpace(c.Frontmatter.Generated.At) == "" {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   "generated frontmatter block is missing 'at' timestamp",
			})
		} else {
			_, _, tsIssues := validateTimestampFormat(c, "generated.at", c.Frontmatter.Generated.At)
			issues = append(issues, tsIssues...)
		}
	}

	// 5. Verified records & current-only trust tier check (§5.2, §5.3)
	for i, v := range c.Frontmatter.Verified {
		if strings.TrimSpace(v.By) == "" {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("verified[%d] record is missing 'by' actor identifier", i),
			})
		}
		if strings.TrimSpace(v.At) == "" {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("verified[%d] record is missing 'at' timestamp", i),
			})
		} else {
			_, _, tsIssues := validateTimestampFormat(c, fmt.Sprintf("verified[%d].at", i), v.At)
			issues = append(issues, tsIssues...)
		}
	}
	if c.HasOutdatedVerifications() && c.Frontmatter.Generated != nil {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("concept has verification record(s) older than 'generated.at' (%s); active trust tier is '%s' until re-verified", strings.TrimSpace(c.Frontmatter.Generated.At), c.TrustTier()),
		})
	}

	// 6. Attestation / Attested Computation contract check (§10)
	cType := strings.ToLower(strings.TrimSpace(c.Frontmatter.Type))
	hasLegacyAttestation := c.Frontmatter.Attestation != nil &&
		(strings.TrimSpace(c.Frontmatter.Attestation.Query) != "" || strings.TrimSpace(c.Frontmatter.Attestation.Executor) != "")
	if cType == "attestation" {
		// Silent backward compatibility for draft type: Attestation when populated
		if !hasLegacyAttestation {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityError,
				Message:   "attestation concept requires an 'attestation' frontmatter block with 'query' or 'executor'",
			})
		}
	} else if cType == "attested computation" {
		for i, p := range c.Frontmatter.Parameters {
			if strings.TrimSpace(p.Name) == "" {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityError,
					Message:   fmt.Sprintf("parameters[%d] is missing required 'name' field", i),
				})
			}
			if strings.TrimSpace(p.Type) == "" {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityError,
					Message:   fmt.Sprintf("parameters[%d] is missing required 'type' field", i),
				})
			}
		}

		if strings.TrimSpace(c.Frontmatter.Runtime) == "" && !hasLegacyAttestation {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityError,
				Message:   "attested computation concept requires a 'runtime' frontmatter field",
			})
		} else if !hasLegacyAttestation {
			hasInline := c.InlineComputation() != ""
			hasFile := strings.TrimSpace(c.Frontmatter.Computation) != ""
			if !hasInline && !hasFile {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityWarning,
					Message:   "attested computation has neither an inline '# Computation' code block nor a 'computation' file path",
				})
			} else if hasInline && hasFile {
				issues = append(issues, Issue{
					ConceptID: c.ID,
					Path:      c.Path,
					Severity:  SeverityWarning,
					Message:   "attested computation specifies both an inline '# Computation' code block and a 'computation' file path; prefer one",
				})
			}
		}
	}

	return issues
}

func extractFootnoteLabels(body string) []string {
	if body == "" {
		return nil
	}
	seen := make(map[string]bool)
	var labels []string

	fenceChar := byte(0)
	fenceLen := 0
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if fc, fl := parseCodeFence(trimmed); fc != 0 {
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
		if strings.HasPrefix(trimmed, "[^") && strings.Contains(trimmed, "]:") {
			continue
		}
		// Strip CommonMark inline backtick code spans before scanning [^label] references
		cleanLine := stripInlineCodeSpans(line)
		for _, m := range footnoteRefRe.FindAllStringSubmatch(cleanLine, -1) {
			if len(m) == 2 && !seen[m[1]] {
				seen[m[1]] = true
				labels = append(labels, m[1])
			}
		}
	}
	return labels
}

func parseCodeFence(trimmed string) (byte, int) {
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

func stripInlineCodeSpans(line string) string {
	if !strings.Contains(line, "`") {
		return line
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
		closeIdx := -1
		j := i
		for j < len(line) {
			if line[j] != '`' {
				j++
				continue
			}
			cStart := j
			for j < len(line) && line[j] == '`' {
				j++
			}
			if j-cStart == tickLen {
				closeIdx = cStart
				break
			}
		}
		if closeIdx == -1 {
			continue
		}
		out.WriteString(line[plainStart:tickStart])
		out.WriteByte(' ')
		i = closeIdx + tickLen
		plainStart = i
	}
	out.WriteString(line[plainStart:])
	return out.String()
}

// validateLinksWithRoots scans the concept body for broken internal markdown links using a robust Markdown AST parser.
func validateLinksWithRoots(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache, roots bundleRoots) []Issue {
	var issues []Issue

	// Extract links from AST, ignoring code blocks
	targets := extractLinksFromAST(c.Body)

	for _, target := range targets {
		// Ignore external links
		if links.IsExternal(target) {
			continue
		}

		// Ignore links with anchors only in the same file (e.g. #section-name)
		if strings.HasPrefix(target, "#") {
			continue
		}

		// Strip anchor/fragment part of target link (e.g., ../tables/users.md#columns -> ../tables/users.md)
		if idx := strings.Index(target, "#"); idx != -1 {
			target = target[:idx]
		}

		// If target is empty after stripping anchor, skip
		if target == "" {
			continue
		}

		// Handle remote hub:// links
		if strings.HasPrefix(target, "hub://") {
			issues = append(issues, validateHubTarget(c, target, opts, rcache)...)
			continue
		}

		// Resolve logical relative or absolute path inside bundle (with legacy v0.1/v0.2 fallback)
		bundleRoot := ""
		okfVer := ""
		if b != nil {
			bundleRoot = b.Path
			okfVer = b.OKFVersion
		}
		resolvedRelPath, escaped := links.ResolveBundlePath(bundleRoot, c.Path, target, okfVer)

		// Normalize: check if resolved path goes beyond the root
		if escaped {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("link '%s' references a path outside the bundle", target),
			})
			continue
		}

		// Convert to Concept ID by removing the extension (.md)
		targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")

		// Check if the concept or reserved/bundle file exists in the bundle
		if !bundleTargetExistsWithRoots(b, resolvedRelPath, roots) {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("broken link: target concept '%s' (resolved from '%s') does not exist", targetConceptID, target),
			})
		}
	}

	return issues
}

// extractLinksFromAST parses the markdown body using Goldmark and walks the AST to find all link destinations.
// Links inside inline code or fenced code blocks are automatically bypassed by the parser.
func extractLinksFromAST(body string) []string {
	var targets []string
	reader := text.NewReader([]byte(body))
	doc := sharedMarkdown.Parser().Parse(reader)

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if link, ok := n.(*ast.Link); ok {
				targets = append(targets, string(link.Destination))
			}
		}
		return ast.WalkContinue, nil
	})

	return targets
}

func validateHubTarget(c *bundle.Concept, target string, opts Options, rcache remoteCache) []Issue {
	var issues []Issue

	bundleID, conceptID, parseErr := links.ParseHubURI(target)
	if parseErr != nil {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("malformed hub:// URI '%s': %v", target, parseErr),
		})
		return issues
	}

	if opts.Host == "" {
		// Offline mode or no host configured, we cannot validate remote existence
		return issues
	}

	entry, ok := rcache[bundleID]
	if !ok {
		entry = fetchRemoteBundle(opts, bundleID)
		rcache[bundleID] = entry
	}

	if entry.unreachable {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("unable to verify remote link '%s' (%s)", target, entry.errMsg),
		})
		return issues
	}

	if conceptID != "" && !entry.concepts[conceptID] {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("broken remote link: concept '%s' not found in remote bundle '%s'", conceptID, bundleID),
		})
	} else if conceptID == "" && len(entry.concepts) == 0 {
		issues = append(issues, Issue{
			ConceptID: c.ID,
			Path:      c.Path,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("remote bundle '%s' could not be resolved or is empty", bundleID),
		})
	}

	return issues
}

func fetchRemoteBundle(opts Options, bundleID string) remoteBundleEntry {
	apiURL := fmt.Sprintf("%s/api/concepts?bundle=%s", strings.TrimRight(opts.Host, "/"), url.QueryEscape(bundleID))
	req, reqErr := http.NewRequest("GET", apiURL, nil)
	if reqErr != nil {
		return remoteBundleEntry{unreachable: true, errMsg: reqErr.Error()}
	}
	if opts.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return remoteBundleEntry{unreachable: true, errMsg: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return remoteBundleEntry{concepts: make(map[string]bool)}
	}
	if resp.StatusCode != http.StatusOK {
		return remoteBundleEntry{unreachable: true, errMsg: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	var concepts []struct {
		ID string `json:"id"`
	}
	var wrapper struct {
		Concepts *[]struct {
			ID string `json:"id"`
		} `json:"concepts"`
	}
	dec := json.NewDecoder(resp.Body)
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return remoteBundleEntry{unreachable: true, errMsg: err.Error()}
	}
	conceptSet := make(map[string]bool)
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Concepts != nil {
		for _, remC := range *wrapper.Concepts {
			conceptSet[remC.ID] = true
		}
	} else if err := json.Unmarshal(raw, &concepts); err == nil {
		for _, remC := range concepts {
			conceptSet[remC.ID] = true
		}
	} else {
		return remoteBundleEntry{unreachable: true, errMsg: "invalid JSON response"}
	}
	return remoteBundleEntry{concepts: conceptSet}
}

// validateCitationsWithRoots verifies that any local/internal or hub:// citation links resolve to existing concepts.
func validateCitationsWithRoots(c *bundle.Concept, b *bundle.Bundle, opts Options, rcache remoteCache, roots bundleRoots) []Issue {
	var issues []Issue

	for _, cite := range c.Citations {
		target := cite.URI

		// Ignore external links
		if links.IsExternal(target) {
			continue
		}

		// Ignore links with anchors only in the same file (e.g. #section-name)
		if strings.HasPrefix(target, "#") {
			continue
		}

		if strings.HasPrefix(target, "hub://") {
			issues = append(issues, validateHubTarget(c, target, opts, rcache)...)
			continue
		}

		// Strip anchor/fragment part
		if idx := strings.Index(target, "#"); idx != -1 {
			target = target[:idx]
		}

		if target == "" {
			continue
		}

		// Resolve path (with legacy v0.1/v0.2 fallback)
		bundleRoot := ""
		okfVer := ""
		if b != nil {
			bundleRoot = b.Path
			okfVer = b.OKFVersion
		}
		resolvedRelPath, escaped := links.ResolveBundlePath(bundleRoot, c.Path, target, okfVer)

		// Check if resolved path goes beyond root
		if escaped {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("citation link '%s' references a path outside the bundle", cite.URI),
			})
			continue
		}

		// Convert to Concept ID by removing extension
		targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")

		// Check existence
		if !bundleTargetExistsWithRoots(b, resolvedRelPath, roots) {
			issues = append(issues, Issue{
				ConceptID: c.ID,
				Path:      c.Path,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("broken citation link: target concept '%s' (resolved from '%s') does not exist", targetConceptID, cite.URI),
			})
		}
	}

	return issues
}

// bundleTargetExistsWithRoots checks whether a bundle-relative path resolves to a parsed concept,
// a recorded index.md file, an existing file inside the bundle (such as index.md, log.md,
// or a reference file), or a directory containing an index.md file, without escaping the bundle root.
func bundleTargetExistsWithRoots(b *bundle.Bundle, resolvedRelPath string, roots bundleRoots) bool {
	if b == nil {
		return false
	}
	if resolvedRelPath == "." || resolvedRelPath == "" {
		resolvedRelPath = "index.md"
	}

	targetConceptID := strings.TrimSuffix(resolvedRelPath, ".md")
	if _, exists := b.GetConcept(targetConceptID); exists {
		return true
	}

	if b.IndexFiles != nil {
		if _, exists := b.IndexFiles[resolvedRelPath]; exists {
			return true
		}
		if _, exists := b.IndexFiles[targetConceptID+".md"]; exists {
			return true
		}
		if _, exists := b.IndexFiles[path.Join(resolvedRelPath, "index.md")]; exists {
			return true
		}
	}

	if b.LogFiles != nil {
		if _, exists := b.LogFiles[resolvedRelPath]; exists {
			return true
		}
	}

	if !roots.hasPath || roots.absErr != nil || roots.realErr != nil {
		return false
	}

	bundleRoot := roots.absRoot
	targetPath := filepath.Join(bundleRoot, filepath.FromSlash(resolvedRelPath))
	rel, err := filepath.Rel(bundleRoot, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		targetPath = filepath.Join(targetPath, "index.md")
		info, err = os.Stat(targetPath)
		if err != nil || info.IsDir() {
			return false
		}
	}

	realTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return false
	}
	realRel, err := filepath.Rel(roots.realRoot, realTarget)
	return err == nil && realRel != ".." && !strings.HasPrefix(realRel, ".."+string(os.PathSeparator))
}
