package bundle

import (
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"time"
)

// MergeStrategy defines how conflicting concepts are reconciled during bundle merges.
type MergeStrategy string

const (
	// MergeUnion reconciles matching concepts by trust tier, freshness, and timestamp, unioning tags.
	MergeUnion MergeStrategy = "union"
	// MergeOurs retains the source bundle's concept when both bundles define the same concept ID.
	MergeOurs MergeStrategy = "ours"
	// MergeTheirs retains the target bundle's concept when both bundles define the same concept ID.
	MergeTheirs MergeStrategy = "theirs"
)

// MergeOptions configures bundle merging behavior, allowing deterministic reference time
// and warning output redirection for tests.
type MergeOptions struct {
	Now        time.Time
	WarnWriter io.Writer
}

// Merge combines two bundles (source and target) into a new bundle based on the strategy.
func Merge(source, target *Bundle, strategy MergeStrategy) (*Bundle, error) {
	return MergeWithOptions(source, target, strategy, MergeOptions{})
}

// MergeWithOptions combines two bundles (source and target) into a new bundle using opts.
func MergeWithOptions(source, target *Bundle, strategy MergeStrategy, opts MergeOptions) (*Bundle, error) {
	if strategy != MergeUnion && strategy != MergeOurs && strategy != MergeTheirs {
		return nil, fmt.Errorf("invalid merge strategy: %q", strategy)
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.WarnWriter == nil {
		opts.WarnWriter = os.Stderr
	}

	merged := NewBundle(source.Path)
	merged.OKFVersion = source.OKFVersion
	if merged.OKFVersion == "" {
		merged.OKFVersion = target.OKFVersion
	}
	merged.Title = source.Title
	if merged.Title == "" {
		merged.Title = target.Title
	}
	merged.Description = source.Description
	if merged.Description == "" {
		merged.Description = target.Description
	}
	merged.License = source.License
	if merged.License == "" {
		merged.License = target.License
	}

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

		reconciled, err := reconcileConcepts(sourceC, targetC, strategy, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile concept %s: %w", id, err)
		}
		merged.Concepts[id] = reconciled
	}

	// 3. Copy and merge IndexFiles
	for relPath, idx := range source.IndexFiles {
		merged.IndexFiles[relPath] = copyConcept(idx)
	}
	for relPath, targetIdx := range target.IndexFiles {
		sourceIdx, exists := merged.IndexFiles[relPath]
		if !exists {
			merged.IndexFiles[relPath] = copyConcept(targetIdx)
			continue
		}
		reconciled, err := reconcileConcepts(sourceIdx, targetIdx, strategy, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile index file %s: %w", relPath, err)
		}
		merged.IndexFiles[relPath] = reconciled
		if relPath == "index.md" && reconciled != nil {
			if reconciled.Frontmatter.OKFVersion != "" {
				merged.OKFVersion = reconciled.Frontmatter.OKFVersion
			}
			if reconciled.Frontmatter.Title != "" {
				merged.Title = reconciled.Frontmatter.Title
			}
			if reconciled.Frontmatter.Desc != "" {
				merged.Description = reconciled.Frontmatter.Desc
			}
			if reconciled.Frontmatter.License != "" {
				merged.License = reconciled.Frontmatter.License
			}
		}
	}

	// 4. Copy and merge LogFiles
	for relPath, lg := range source.LogFiles {
		merged.LogFiles[relPath] = copyConcept(lg)
	}
	for relPath, targetLg := range target.LogFiles {
		sourceLg, exists := merged.LogFiles[relPath]
		if !exists {
			merged.LogFiles[relPath] = copyConcept(targetLg)
			continue
		}
		reconciled, err := reconcileConcepts(sourceLg, targetLg, strategy, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile log file %s: %w", relPath, err)
		}
		merged.LogFiles[relPath] = reconciled
	}

	return merged, nil
}

func cloneExtraMap(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	return maps.Clone(m)
}

func cloneUsageWindow(uw *UsageWindow) *UsageWindow {
	if uw == nil {
		return nil
	}
	return &UsageWindow{
		From:  uw.From,
		To:    uw.To,
		Extra: cloneExtraMap(uw.Extra),
	}
}

func copyConcept(c *Concept) *Concept {
	if c == nil {
		return nil
	}
	newC := &Concept{
		ID:             c.ID,
		Path:           c.Path,
		Body:           c.Body,
		HasFrontmatter: c.HasFrontmatter,
		ParseError:     c.ParseError,
	}

	fm := Frontmatter{
		OKFVersion:   c.Frontmatter.OKFVersion,
		License:      c.Frontmatter.License,
		Type:         c.Frontmatter.Type,
		Title:        c.Frontmatter.Title,
		Desc:         c.Frontmatter.Desc,
		Resource:     c.Frontmatter.Resource,
		Timestamp:    c.Frontmatter.Timestamp,
		Status:       c.Frontmatter.Status,
		SupersededBy: c.Frontmatter.SupersededBy,
		StaleAfter:   c.Frontmatter.StaleAfter,
		UsageWindow:  cloneUsageWindow(c.Frontmatter.UsageWindow),
		Runtime:      c.Frontmatter.Runtime,
		Computation:  c.Frontmatter.Computation,
	}

	if c.Frontmatter.Tags != nil {
		fm.Tags = make([]string, len(c.Frontmatter.Tags))
		copy(fm.Tags, c.Frontmatter.Tags)
	}
	if c.Frontmatter.Sources != nil {
		fm.Sources = make([]Source, len(c.Frontmatter.Sources))
		for i, src := range c.Frontmatter.Sources {
			srcCopy := src
			srcCopy.UsageWindow = cloneUsageWindow(src.UsageWindow)
			srcCopy.Extra = cloneExtraMap(src.Extra)
			fm.Sources[i] = srcCopy
		}
	}
	if c.Frontmatter.Parameters != nil {
		fm.Parameters = make([]Parameter, len(c.Frontmatter.Parameters))
		for i, p := range c.Frontmatter.Parameters {
			pCopy := p
			pCopy.Extra = cloneExtraMap(p.Extra)
			fm.Parameters[i] = pCopy
		}
	}
	if c.Frontmatter.Generated != nil {
		g := *c.Frontmatter.Generated
		g.Extra = cloneExtraMap(c.Frontmatter.Generated.Extra)
		fm.Generated = &g
	}
	if c.Frontmatter.Verified != nil {
		fm.Verified = make(ActorInfoList, len(c.Frontmatter.Verified))
		for i, v := range c.Frontmatter.Verified {
			vCopy := v
			vCopy.Extra = cloneExtraMap(v.Extra)
			fm.Verified[i] = vCopy
		}
	}
	if c.Frontmatter.Executor != nil {
		ex := Executor{
			Resource: c.Frontmatter.Executor.Resource,
			Extra:    cloneExtraMap(c.Frontmatter.Executor.Extra),
		}
		if c.Frontmatter.Executor.Receipt != nil {
			ex.Receipt = make([]string, len(c.Frontmatter.Executor.Receipt))
			copy(ex.Receipt, c.Frontmatter.Executor.Receipt)
		}
		fm.Executor = &ex
	}
	if c.Frontmatter.Attester != nil {
		at := *c.Frontmatter.Attester
		at.Extra = cloneExtraMap(c.Frontmatter.Attester.Extra)
		fm.Attester = &at
	}
	if c.Frontmatter.Attestation != nil {
		att := *c.Frontmatter.Attestation
		fm.Attestation = &att
	}
	fm.Extra = cloneExtraMap(c.Frontmatter.Extra)
	newC.Frontmatter = fm

	if c.Citations != nil {
		newC.Citations = make([]Citation, len(c.Citations))
		copy(newC.Citations, c.Citations)
	}

	return newC
}

func reconcileConcepts(c1, c2 *Concept, strategy MergeStrategy, opts MergeOptions) (*Concept, error) {
	switch strategy {
	case MergeOurs:
		return copyConcept(c1), nil
	case MergeTheirs:
		return copyConcept(c2), nil
	case MergeUnion:
		c1Newer := isConceptNewer(c1, c2)
		tier1 := c1.TrustTier()
		tier2 := c2.TrustTier()
		stale1 := c1.IsStale(opts.Now)
		stale2 := c2.IsStale(opts.Now)

		nonStaleHuman1 := tier1 == TrustTierHumanReviewed && !stale1
		nonStaleHuman2 := tier2 == TrustTierHumanReviewed && !stale2
		staleHuman1 := tier1 == TrustTierHumanReviewed && stale1
		staleHuman2 := tier2 == TrustTierHumanReviewed && stale2

		var base, override *Concept
		strictlyHigherTrust := false

		switch {
		case nonStaleHuman1 && !nonStaleHuman2:
			base, override = c1, c2
			strictlyHigherTrust = true
			if !c1Newer {
				fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q kept non-stale human-reviewed source over newer %s target\n", c1.ID, tier2)
			}
		case nonStaleHuman2 && !nonStaleHuman1:
			base, override = c2, c1
			strictlyHigherTrust = true
			if c1Newer {
				fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q kept non-stale human-reviewed target over newer %s source\n", c2.ID, tier1)
			}
		case staleHuman1 || staleHuman2:
			// Past stale_after: newer revision wins and replaces expired human sign-off.
			if c1Newer {
				base, override = c1, c2
				if staleHuman2 {
					fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q replaced stale human-reviewed target with newer %s source\n", c1.ID, tier1)
				}
			} else {
				base, override = c2, c1
				if staleHuman1 {
					fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q replaced stale human-reviewed source with newer %s target\n", c2.ID, tier2)
				}
			}
		case tier1 == TrustTierMachineConfirmed && tier2 == TrustTierUnverified && !stale1:
			base, override = c1, c2
			strictlyHigherTrust = true
			if !c1Newer {
				fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q kept machine-confirmed source over newer unverified target\n", c1.ID)
			}
		case tier2 == TrustTierMachineConfirmed && tier1 == TrustTierUnverified && !stale2:
			base, override = c2, c1
			strictlyHigherTrust = true
			if c1Newer {
				fmt.Fprintf(opts.WarnWriter, "merge warning: concept %q kept machine-confirmed target over newer unverified source\n", c2.ID)
			}
		default:
			if c1Newer {
				base, override = c1, c2
			} else {
				base, override = c2, c1
			}
		}

		res := copyConcept(base)

		// 1. Backfill only non-authoritative basic metadata if empty on base
		if res.Frontmatter.Type == "" {
			res.Frontmatter.Type = override.Frontmatter.Type
		}
		if res.Frontmatter.Title == "" {
			res.Frontmatter.Title = override.Frontmatter.Title
		}
		if res.Frontmatter.Desc == "" {
			res.Frontmatter.Desc = override.Frontmatter.Desc
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

		// 3. Only backfill Extra keys when trust tiers are equal (never from lower-trust override)
		if !strictlyHigherTrust && len(override.Frontmatter.Extra) > 0 {
			if res.Frontmatter.Extra == nil {
				res.Frontmatter.Extra = make(map[string]any)
			}
			for k, v := range override.Frontmatter.Extra {
				if _, exists := res.Frontmatter.Extra[k]; !exists {
					res.Frontmatter.Extra[k] = v
				}
			}
		}

		// Citations remain derived from the winning revision's Body (already copied from base).
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported merge strategy: %q", strategy)
	}
}

// isConceptNewer compares the most recent recorded activity timestamp across Timestamp,
// generated.at, and verified[].at. On equal or unparseable timestamps, ties favor c2 (target).
func isConceptNewer(c1, c2 *Concept) bool {
	s1 := effectiveTimestamp(c1)
	s2 := effectiveTimestamp(c2)

	t1, ok1 := ParseOKFTimestamp(s1)
	t2, ok2 := ParseOKFTimestamp(s2)

	if ok1 && ok2 {
		return t1.After(t2)
	}
	if !ok1 && ok2 {
		return false
	}
	if ok1 && !ok2 {
		return true
	}
	return s1 > s2
}

func effectiveTimestamp(c *Concept) string {
	if c == nil {
		return ""
	}

	var candidates []string
	if c.Frontmatter.Timestamp != "" {
		candidates = append(candidates, c.Frontmatter.Timestamp)
	}
	if c.Frontmatter.Generated != nil && c.Frontmatter.Generated.At != "" {
		candidates = append(candidates, c.Frontmatter.Generated.At)
	}
	for _, v := range c.Frontmatter.Verified {
		if v.At != "" {
			candidates = append(candidates, v.At)
		}
	}

	var latest time.Time
	hasParsed := false
	latestRaw := ""
	fallbackRaw := ""

	for _, raw := range candidates {
		if fallbackRaw == "" {
			fallbackRaw = raw
		}
		if t, ok := ParseOKFTimestamp(raw); ok {
			if !hasParsed || t.After(latest) {
				latest = t
				latestRaw = raw
				hasParsed = true
			}
		}
	}
	if hasParsed {
		return latestRaw
	}
	return fallbackRaw
}
