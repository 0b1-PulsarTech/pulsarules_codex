package loader

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

const (
	mergedJoin   = " + "
	mergedSuffix = " (merged)"
)

func merge(base, private *projectfile.File, presetFlag string) (*projectfile.File, error) {
	keptBase := privateWinsOverrides(base, private, presetFlag)
	merged := &projectfile.File{
		Path:             mergedName(base, private),
		Sources:          []string{base.Path, private.Path},
		Version:          base.Version,
		Preset:           cmp.Or(private.Preset, base.Preset),
		IncludeGenerated: cmp.Or(private.IncludeGenerated, base.IncludeGenerated),
		SkipDirs:         union(base.SkipDirs, private.SkipDirs),
		Tests:            union(base.Tests, private.Tests),
		Analyzers:        mergeAnalyzers(base.Analyzers, private.Analyzers),
		Overrides:        slices.Concat(keptBase, private.Overrides),
		Delegated:        mergeDelegated(base.Delegated, private.Delegated),
	}
	// Each input passed the text limits alone, so only the list sizes can overflow in a merge.
	if err := checkMergedSizes(merged); err != nil {
		return nil, err
	}
	if err := merged.Validate(configRules()); err != nil {
		return nil, err
	}
	return merged, nil
}

func mergedName(base, private *projectfile.File) string {
	return base.Path + mergedJoin + private.Path + mergedSuffix
}

func checkMergedSizes(merged *projectfile.File) error {
	sizes := []struct {
		field        string
		count, limit int
	}{
		{"skip_dirs", len(merged.SkipDirs), projectfile.MaxSeqItems},
		{"tests", len(merged.Tests), projectfile.MaxSeqItems},
		{"overrides", len(merged.Overrides), projectfile.MaxSeqItems},
		{"delegated", len(merged.Delegated), projectfile.MaxSeqItems},
		{"analyzers", len(merged.Analyzers), projectfile.MaxMapKeys},
	}
	for _, size := range sizes {
		if size.count > size.limit {
			return fmt.Errorf("%s: %s: more than %d entries", merged.Path, size.field, size.limit)
		}
	}
	return nil
}

func union(base, private []string) []string {
	seen := make(map[string]bool, len(base)+len(private))
	var merged []string
	for _, item := range slices.Concat(base, private) {
		if !seen[item] {
			seen[item] = true
			merged = append(merged, item)
		}
	}
	return merged
}

func mergeAnalyzers(
	base, private map[string]projectfile.AnalyzerEntry,
) map[string]projectfile.AnalyzerEntry {
	if len(base)+len(private) == 0 {
		return nil
	}
	merged := make(map[string]projectfile.AnalyzerEntry, len(base)+len(private))
	maps.Copy(merged, base)
	for id, entry := range private {
		below := merged[id]
		merged[id] = projectfile.AnalyzerEntry{
			Enabled:  cmp.Or(entry.Enabled, below.Enabled),
			Severity: cmp.Or(entry.Severity, below.Severity),
			Params:   mergeParams(below.Params, entry.Params),
		}
	}
	return merged
}

func mergeParams(base, private map[string]any) map[string]any {
	if len(base)+len(private) == 0 {
		return nil
	}
	merged := make(map[string]any, len(base)+len(private))
	maps.Copy(merged, base)
	maps.Copy(merged, private)
	return merged
}

func mergeDelegated(base, private []projectfile.DelegatedEntry) []projectfile.DelegatedEntry {
	merged := slices.Clone(base)
	for _, entry := range private {
		at := slices.IndexFunc(merged, func(kept projectfile.DelegatedEntry) bool {
			return kept.ID == entry.ID
		})
		if at < 0 {
			merged = append(merged, entry)
			continue
		}
		merged[at] = entry
	}
	return merged
}
