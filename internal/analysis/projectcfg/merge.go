package projectcfg

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func merge(base, private *config.ProjectFile, presetFlag string) (*config.ProjectFile, error) {
	keptBase := privateWinsOverrides(base, private, presetFlag)
	merged := &config.ProjectFile{
		Path:             mergedName(base, private),
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
	if err := merged.Validate(Rules()); err != nil {
		return nil, err
	}
	return merged, nil
}

func mergedName(base, private *config.ProjectFile) string {
	return base.Path + mergedJoin + private.Path + mergedSuffix
}

func checkMergedSizes(merged *config.ProjectFile) error {
	sizes := []struct {
		field        string
		count, limit int
	}{
		{"skip_dirs", len(merged.SkipDirs), config.MaxSeqItems},
		{"tests", len(merged.Tests), config.MaxSeqItems},
		{"overrides", len(merged.Overrides), config.MaxSeqItems},
		{"delegated", len(merged.Delegated), config.MaxSeqItems},
		{"analyzers", len(merged.Analyzers), config.MaxMapKeys},
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
	base, private map[string]config.AnalyzerEntry,
) map[string]config.AnalyzerEntry {
	if len(base)+len(private) == 0 {
		return nil
	}
	merged := make(map[string]config.AnalyzerEntry, len(base)+len(private))
	maps.Copy(merged, base)
	for id, entry := range private {
		below := merged[id]
		merged[id] = config.AnalyzerEntry{
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

func mergeDelegated(base, private []config.DelegatedEntry) []config.DelegatedEntry {
	merged := slices.Clone(base)
	for _, entry := range private {
		at := slices.IndexFunc(merged, func(kept config.DelegatedEntry) bool {
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
