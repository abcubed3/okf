package bundle

import (
	"fmt"
	"io"
	"sort"
)

type ConceptDiffType string

const (
	DiffAdded    ConceptDiffType = "ADDED"
	DiffDeleted  ConceptDiffType = "DELETED"
	DiffModified ConceptDiffType = "MODIFIED"
)

type FieldDiff struct {
	OldValue interface{}
	NewValue interface{}
}

type ConceptDiff struct {
	ID        string
	Type      ConceptDiffType
	FmDiffs   map[string]FieldDiff
	BodyDrift bool
}

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

	return diff, nil
}

func compareFrontmatter(oldFm, newFm Frontmatter) map[string]FieldDiff {
	diffs := make(map[string]FieldDiff)

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

	// Compare Extra map
	for k, newV := range newFm.Extra {
		oldV, exists := oldFm.Extra[k]
		if !exists {
			diffs["extra."+k] = FieldDiff{OldValue: nil, NewValue: newV}
		} else if fmt.Sprintf("%v", oldV) != fmt.Sprintf("%v", newV) {
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
	for i := range a {
		if a[i] != b[i] {
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
