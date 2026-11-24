package bundle

import (
	"fmt"
	"io"
	"reflect"
	"sort"
)

// ConceptDiffType classifies how a concept or reserved file changed between two bundles.
type ConceptDiffType string

const (
	// DiffAdded indicates a concept or reserved file present only in the target bundle.
	DiffAdded ConceptDiffType = "ADDED"
	// DiffDeleted indicates a concept or reserved file removed in the target bundle.
	DiffDeleted ConceptDiffType = "DELETED"
	// DiffModified indicates a concept or reserved file whose frontmatter or body changed.
	DiffModified ConceptDiffType = "MODIFIED"
)

// FieldDiff records the old and new values of a modified frontmatter attribute.
type FieldDiff struct {
	OldValue any
	NewValue any
}

// ConceptDiff summarizes frontmatter and body drift for a single concept or reserved file.
type ConceptDiff struct {
	ID        string
	Type      ConceptDiffType
	FmDiffs   map[string]FieldDiff
	BodyDrift bool
}

// BundleDiff aggregates all structural and metadata changes between two bundles.
type BundleDiff struct {
	SourcePath string
	TargetPath string
	Changes    []ConceptDiff
}

// Diff compares two bundles (representing old/source state and new/target state)
// and returns structural additions, deletions, and modifications.
func Diff(oldB, newB *Bundle) (*BundleDiff, error) {
	diff := &BundleDiff{
		SourcePath: oldB.Path,
		TargetPath: newB.Path,
		Changes:    []ConceptDiff{},
	}

	// 1. Check for additions and modifications
	for id, newC := range newB.Concepts {
		oldC, exists := oldB.GetConcept(id)
		if !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   id,
				Type: DiffAdded,
			})
			continue
		}

		bodyDrift := newC.Body != oldC.Body
		fmDiffs := compareFrontmatter(oldC.Frontmatter, newC.Frontmatter)

		if bodyDrift || len(fmDiffs) > 0 {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:        id,
				Type:      DiffModified,
				FmDiffs:   fmDiffs,
				BodyDrift: bodyDrift,
			})
		}
	}

	// 2. Check for deletions
	for id := range oldB.Concepts {
		if _, exists := newB.GetConcept(id); !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   id,
				Type: DiffDeleted,
			})
		}
	}

	// 3. Check IndexFiles for additions, modifications, and deletions
	for relPath, newIdx := range newB.IndexFiles {
		oldIdx, exists := oldB.IndexFiles[relPath]
		if !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   relPath,
				Type: DiffAdded,
			})
			continue
		}
		bodyDrift := newIdx.Body != oldIdx.Body
		fmDiffs := compareFrontmatter(oldIdx.Frontmatter, newIdx.Frontmatter)
		if bodyDrift || len(fmDiffs) > 0 {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:        relPath,
				Type:      DiffModified,
				FmDiffs:   fmDiffs,
				BodyDrift: bodyDrift,
			})
		}
	}
	for relPath := range oldB.IndexFiles {
		if _, exists := newB.IndexFiles[relPath]; !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   relPath,
				Type: DiffDeleted,
			})
		}
	}

	// 4. Check LogFiles for additions, modifications, and deletions
	for relPath, newLg := range newB.LogFiles {
		oldLg, exists := oldB.LogFiles[relPath]
		if !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   relPath,
				Type: DiffAdded,
			})
			continue
		}
		bodyDrift := newLg.Body != oldLg.Body
		fmDiffs := compareFrontmatter(oldLg.Frontmatter, newLg.Frontmatter)
		if bodyDrift || len(fmDiffs) > 0 {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:        relPath,
				Type:      DiffModified,
				FmDiffs:   fmDiffs,
				BodyDrift: bodyDrift,
			})
		}
	}
	for relPath := range oldB.LogFiles {
		if _, exists := newB.LogFiles[relPath]; !exists {
			diff.Changes = append(diff.Changes, ConceptDiff{
				ID:   relPath,
				Type: DiffDeleted,
			})
		}
	}

	return diff, nil
}

func compareFrontmatter(oldFm, newFm Frontmatter) map[string]FieldDiff {
	diffs := make(map[string]FieldDiff)

	if oldFm.OKFVersion != newFm.OKFVersion {
		diffs["okf_version"] = FieldDiff{OldValue: oldFm.OKFVersion, NewValue: newFm.OKFVersion}
	}
	if oldFm.License != newFm.License {
		diffs["license"] = FieldDiff{OldValue: oldFm.License, NewValue: newFm.License}
	}
	if oldFm.Type != newFm.Type {
		diffs["type"] = FieldDiff{OldValue: oldFm.Type, NewValue: newFm.Type}
	}
	if oldFm.Title != newFm.Title {
		diffs["title"] = FieldDiff{OldValue: oldFm.Title, NewValue: newFm.Title}
	}
	if oldFm.Desc != newFm.Desc {
		diffs["description"] = FieldDiff{OldValue: oldFm.Desc, NewValue: newFm.Desc}
	}
	if oldFm.Resource != newFm.Resource {
		diffs["resource"] = FieldDiff{OldValue: oldFm.Resource, NewValue: newFm.Resource}
	}
	if oldFm.Timestamp != newFm.Timestamp {
		diffs["timestamp"] = FieldDiff{OldValue: oldFm.Timestamp, NewValue: newFm.Timestamp}
	}

	if !slicesEqual(oldFm.Tags, newFm.Tags) {
		diffs["tags"] = FieldDiff{OldValue: oldFm.Tags, NewValue: newFm.Tags}
	}

	if oldFm.Status != newFm.Status {
		diffs["status"] = FieldDiff{OldValue: oldFm.Status, NewValue: newFm.Status}
	}
	if oldFm.SupersededBy != newFm.SupersededBy {
		diffs["superseded_by"] = FieldDiff{OldValue: oldFm.SupersededBy, NewValue: newFm.SupersededBy}
	}
	if oldFm.StaleAfter != newFm.StaleAfter {
		diffs["stale_after"] = FieldDiff{OldValue: oldFm.StaleAfter, NewValue: newFm.StaleAfter}
	}
	if !reflect.DeepEqual(oldFm.Sources, newFm.Sources) {
		diffs["sources"] = FieldDiff{OldValue: oldFm.Sources, NewValue: newFm.Sources}
	}
	if !reflect.DeepEqual(oldFm.UsageWindow, newFm.UsageWindow) {
		diffs["usage_window"] = FieldDiff{OldValue: oldFm.UsageWindow, NewValue: newFm.UsageWindow}
	}
	if !reflect.DeepEqual(oldFm.Generated, newFm.Generated) {
		diffs["generated"] = FieldDiff{OldValue: oldFm.Generated, NewValue: newFm.Generated}
	}
	if !reflect.DeepEqual(oldFm.Verified, newFm.Verified) {
		diffs["verified"] = FieldDiff{OldValue: oldFm.Verified, NewValue: newFm.Verified}
	}
	if oldFm.Runtime != newFm.Runtime {
		diffs["runtime"] = FieldDiff{OldValue: oldFm.Runtime, NewValue: newFm.Runtime}
	}
	if !reflect.DeepEqual(oldFm.Parameters, newFm.Parameters) {
		diffs["parameters"] = FieldDiff{OldValue: oldFm.Parameters, NewValue: newFm.Parameters}
	}
	if oldFm.Computation != newFm.Computation {
		diffs["computation"] = FieldDiff{OldValue: oldFm.Computation, NewValue: newFm.Computation}
	}
	if !reflect.DeepEqual(oldFm.Executor, newFm.Executor) {
		diffs["executor"] = FieldDiff{OldValue: oldFm.Executor, NewValue: newFm.Executor}
	}
	if !reflect.DeepEqual(oldFm.Attester, newFm.Attester) {
		diffs["attester"] = FieldDiff{OldValue: oldFm.Attester, NewValue: newFm.Attester}
	}
	if !reflect.DeepEqual(oldFm.Attestation, newFm.Attestation) {
		diffs["attestation"] = FieldDiff{OldValue: oldFm.Attestation, NewValue: newFm.Attestation}
	}

	// Compare Extra map
	for k, newV := range newFm.Extra {
		oldV, exists := oldFm.Extra[k]
		if !exists {
			diffs["extra."+k] = FieldDiff{OldValue: nil, NewValue: newV}
		} else if !reflect.DeepEqual(oldV, newV) {
			diffs["extra."+k] = FieldDiff{OldValue: oldV, NewValue: newV}
		}
	}
	for k, oldV := range oldFm.Extra {
		if _, exists := newFm.Extra[k]; !exists {
			diffs["extra."+k] = FieldDiff{OldValue: oldV, NewValue: nil}
		}
	}

	return diffs
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac, bc := append([]string{}, a...), append([]string{}, b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

// PrettyPrint prints a formatted human-readable bundle drift report to the writer.
func (bd *BundleDiff) PrettyPrint(w io.Writer) {
	if len(bd.Changes) == 0 {
		fmt.Fprintln(w, "No differences found between bundles. Perfect alignment! 🎉")
		return
	}

	fmt.Fprintln(w, "Bundle Drift Report:")
	fmt.Fprintf(w, "  Source (Old): %s\n", bd.SourcePath)
	fmt.Fprintf(w, "  Target (New): %s\n\n", bd.TargetPath)

	// Sort changes by ID for deterministic output
	sort.Slice(bd.Changes, func(i, j int) bool {
		return bd.Changes[i].ID < bd.Changes[j].ID
	})

	for _, c := range bd.Changes {
		switch c.Type {
		case DiffAdded:
			fmt.Fprintf(w, "[+] ADDED    %s\n", c.ID)
		case DiffDeleted:
			fmt.Fprintf(w, "[-] DELETED  %s\n", c.ID)
		case DiffModified:
			fmt.Fprintf(w, "[~] MODIFIED %s\n", c.ID)
			if c.BodyDrift {
				fmt.Fprintln(w, "    ~ Body text drifted")
			}
			if len(c.FmDiffs) > 0 {
				fmt.Fprintln(w, "    ~ Metadata frontmatter drifted:")
				// Sort keys for deterministic output
				var keys []string
				for k := range c.FmDiffs {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fd := c.FmDiffs[k]
					fmt.Fprintf(w, "        - %s:\n", k)
					fmt.Fprintf(w, "            Old: %v\n", fd.OldValue)
					fmt.Fprintf(w, "            New: %v\n", fd.NewValue)
				}
			}
		}
	}
}
