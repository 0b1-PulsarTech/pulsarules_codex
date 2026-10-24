package loader

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
)

const (
	mergedJoin   = " + "
	mergedSuffix = " (merged)"
)

func (stack layers) merge() (*projectfile.File, error) {
	base, private := stack.base, stack.private
	keptBase := stack.privateWins()
	merged := &projectfile.File{
		Path:             mergedName(base, private),
		Sources:          slices.Concat(sourcesOf(base), sourcesOf(private)),
		Version:          base.Version,
		Preset:           cmp.Or(private.Preset, base.Preset),
		IncludeGenerated: cmp.Or(private.IncludeGenerated, base.IncludeGenerated),
		SkipDirs:         ordered.Dedupe(slices.Concat(base.SkipDirs, private.SkipDirs)),
		Tests:            ordered.Dedupe(slices.Concat(base.Tests, private.Tests)),
		Analyzers:        mergeAnalyzers(base.Analyzers, private.Analyzers),
		Overrides:        slices.Concat(keptBase, private.Overrides),
		Delegated:        mergeDelegated(base.Delegated, private.Delegated),
	}
	// Each input passed the text limits alone, so only the list sizes can overflow in a merge.
	if err := checkMergedSizes(merged); err != nil {
		return nil, err
	}
	if err := merged.Validate(stack.rules); err != nil {
		return nil, err
	}
	return merged, nil
}

func (stack layers) combine() (*projectfile.File, error) {
	merged, err := stack.mergeChecked()
	if err != nil {
		return nil, err
	}
	if err = stack.checkOverrides(merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func (stack layers) mergeChecked() (*projectfile.File, error) {
	// Each file alone must build, so a fault of one file is never blamed on the merge.
	for _, layer := range []*projectfile.File{stack.base, stack.private} {
		if _, err := Build(layer, ""); err != nil {
			return nil, &fileError{File: layer.Path, Err: err}
		}
	}
	merged, err := stack.merge()
	if err != nil {
		return nil, &fileError{
			File: firstSource(stack.base), Lead: mergedName(stack.base, stack.private), Err: err,
		}
	}
	return merged, nil
}

func (stack layers) checkOverrides(merged *projectfile.File) error {
	private := stack.private
	// Checked against the merged project state, but numbered as in the private file.
	project := *merged
	project.Overrides = merged.Overrides[:len(merged.Overrides)-len(private.Overrides)]
	cfg, err := Build(&project, stack.preset)
	if err != nil {
		return &fileError{File: faultFile(nil, merged), Err: err}
	}
	if err = private.CheckOverrides(cfg); err != nil {
		return &fileError{File: private.Path, Err: err}
	}
	return nil
}

func mergedName(base, private *projectfile.File) string {
	sources := slices.Concat(sourcesOf(base), sourcesOf(private))
	return strings.Join(sources, mergedJoin) + mergedSuffix
}

func sourcesOf(file *projectfile.File) []string {
	if len(file.Sources) > 0 {
		return file.Sources
	}
	return []string{file.Path}
}

func firstSource(file *projectfile.File) string {
	return sourcesOf(file)[0]
}

func checkMergedSizes(merged *projectfile.File) error {
	sizes := []struct {
		field        string
		count, limit int
	}{
		{"skip_dirs", len(merged.SkipDirs), strictyaml.MaxSeqItems},
		{"tests", len(merged.Tests), strictyaml.MaxSeqItems},
		{"overrides", len(merged.Overrides), strictyaml.MaxSeqItems},
		{"delegated", len(merged.Delegated), strictyaml.MaxSeqItems},
		{"analyzers", len(merged.Analyzers), strictyaml.MaxMapKeys},
	}
	for _, size := range sizes {
		if size.count > size.limit {
			return fmt.Errorf("%s: %s: more than %d entries", merged.Path, size.field, size.limit)
		}
	}
	return nil
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
