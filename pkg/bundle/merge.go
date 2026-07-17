package bundle

import (
	"fmt"
	"sort"
	"time"
)

type MergeStrategy string

const (
	MergeUnion  MergeStrategy = "union"
	MergeOurs   MergeStrategy = "ours"
	MergeTheirs MergeStrategy = "theirs"
)

// Merge combines two bundles (source and target) into a new bundle based on the strategy.
func Merge(source, target *Bundle, strategy MergeStrategy) (*Bundle, error) {
	if strategy != MergeUnion && strategy != MergeOurs && strategy != MergeTheirs {
		return nil, fmt.Errorf("invalid merge strategy: %q", strategy)
	}

	merged := NewBundle(source.Path)

	// 1. Copy all source concepts (deep copy)
	for id, c := range source.Concepts {
		merged.Concepts[id] = copyConcept(c)
	}

	// 2. Merge target concepts
	for id, targetC := range target.Concepts {
		sourceC, exists := merged.Concepts[id]
		if !exists {
			merged.Concepts[id] = copyConcept(targetC)
			continue
		}

		reconciled, err := reconcileConcepts(sourceC, targetC, strategy)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile concept %s: %w", id, err)
		}
		merged.Concepts[id] = reconciled
	}

	return merged, nil
}

func copyConcept(c *Concept) *Concept {
	if c == nil {
		return nil
	}
	newC := &Concept{
		ID:         c.ID,
		Path:       c.Path,
		Body:       c.Body,
		ParseError: c.ParseError,
	}

	newC.Frontmatter = Frontmatter{
		Type:      c.Frontmatter.Type,
		Title:     c.Frontmatter.Title,
		Desc:      c.Frontmatter.Desc,
		Resource:  c.Frontmatter.Resource,
		Timestamp: c.Frontmatter.Timestamp,
	}

	if c.Frontmatter.Tags != nil {
		newC.Frontmatter.Tags = make([]string, len(c.Frontmatter.Tags))
		copy(newC.Frontmatter.Tags, c.Frontmatter.Tags)
	}

	if c.Frontmatter.Extra != nil {
		newC.Frontmatter.Extra = make(map[string]interface{})
		for k, v := range c.Frontmatter.Extra {
			newC.Frontmatter.Extra[k] = v
		}
	}

	if c.Citations != nil {
		newC.Citations = make([]Citation, len(c.Citations))
		copy(newC.Citations, c.Citations)
	}

	return newC
}

func reconcileConcepts(c1, c2 *Concept, strategy MergeStrategy) (*Concept, error) {
	switch strategy {
	case MergeOurs:
		return copyConcept(c1), nil
	case MergeTheirs:
		return copyConcept(c2), nil
	case MergeUnion:
		// Determine which concept is newer based on timestamp
		t1, err1 := time.Parse(time.RFC3339, c1.Frontmatter.Timestamp)
		t2, err2 := time.Parse(time.RFC3339, c2.Frontmatter.Timestamp)
		c1IsNewer := true

		if err1 == nil && err2 == nil {
			c1IsNewer = t1.After(t2)
		} else if err1 != nil && err2 == nil {
			c1IsNewer = false
		} else if err1 == nil && err2 != nil {
			c1IsNewer = true
		} else {
			c1IsNewer = c1.Frontmatter.Timestamp >= c2.Frontmatter.Timestamp
		}

		var base, override *Concept
		if c1IsNewer {
			base = c1
			override = c2
		} else {
			base = c2
			override = c1
		}

		res := copyConcept(base)

		// 1. Merge basic metadata if base is empty but override has it
		if res.Frontmatter.Type == "" {
			res.Frontmatter.Type = override.Frontmatter.Type
		}
		if res.Frontmatter.Title == "" {
			res.Frontmatter.Title = override.Frontmatter.Title
		}
		if res.Frontmatter.Desc == "" {
			res.Frontmatter.Desc = override.Frontmatter.Desc
		}
		if res.Frontmatter.Resource == "" {
			res.Frontmatter.Resource = override.Frontmatter.Resource
		}

		// 2. Merge tags (deduplicated & sorted)
		tagMap := make(map[string]bool)
		for _, t := range c1.Frontmatter.Tags {
			if t != "" {
				tagMap[t] = true
			}
		}
		for _, t := range c2.Frontmatter.Tags {
			if t != "" {
				tagMap[t] = true
			}
		}
		var mergedTags []string
		for t := range tagMap {
			mergedTags = append(mergedTags, t)
		}
		sort.Strings(mergedTags)
		res.Frontmatter.Tags = mergedTags

		// 3. Merge Extra maps
		if res.Frontmatter.Extra == nil {
			res.Frontmatter.Extra = make(map[string]interface{})
		}
		for k, v := range override.Frontmatter.Extra {
			if _, exists := res.Frontmatter.Extra[k]; !exists {
				res.Frontmatter.Extra[k] = v
			}
		}

		// 4. Merge Citations (deduplicated by URI)
		citationMap := make(map[string]Citation)
		var mergedCitations []Citation
		for _, cit := range c1.Citations {
			citationMap[cit.URI] = cit
		}
		for _, cit := range c2.Citations {
			citationMap[cit.URI] = cit
		}
		for _, cit := range citationMap {
			mergedCitations = append(mergedCitations, cit)
		}
		sort.Slice(mergedCitations, func(i, j int) bool {
			return mergedCitations[i].Number < mergedCitations[j].Number
		})
		res.Citations = mergedCitations

		return res, nil
	default:
		return nil, fmt.Errorf("unknown merge strategy %q", strategy)
	}
}
