package loader

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func (stack layers) privateWins() []projectfile.OverrideEntry {
	base := stack.base
	// The private file is clone-local and never published, so what it switches off stays off.
	isOff := stack.privateDisables()
	kept := make([]projectfile.OverrideEntry, 0, len(base.Overrides))
	for _, entry := range base.Overrides {
		trimmed := maps.Clone(entry.Analyzers)
		maps.DeleteFunc(trimmed, func(id string, _ projectfile.AnalyzerEntry) bool {
			project, _, _ := strings.Cut(id, "/")
			return isOff(project)
		})
		if len(trimmed) == 0 && len(entry.Analyzers) > 0 {
			continue
		}
		entry.Analyzers = trimmed
		kept = append(kept, entry)
	}
	return kept
}

func (stack layers) privateDisables() func(id string) bool {
	base, private := stack.base, stack.private
	// Only a choice made in the private file drops a team override: the flag alone never does.
	chosen := presetDisabled(private.Preset)
	effective := presetDisabled(cmp.Or(stack.preset, private.Preset))
	return func(id string) bool {
		if entry := private.Analyzers[id]; entry.Enabled != nil {
			return !*entry.Enabled
		}
		if baseOn := base.Analyzers[id].Enabled; baseOn != nil && *baseOn {
			return false
		}
		return chosen(id) && effective(id)
	}
}

func presetDisabled(preset string) func(id string) bool {
	if preset == "" {
		return func(string) bool { return false }
	}
	analyzers := withPreset(nil, preset).Analyzers
	return func(id string) bool {
		entry, listed := analyzers[id]
		return listed && !entry.Enabled
	}
}

func (stack layers) underlay() (*projectfile.File, error) {
	merged, err := stack.mergeChecked()
	if err != nil {
		return nil, err
	}
	// The user layer is personal: what it switches off never invalidates an override above it.
	merged.PersonalOff = stack.lifted(merged)
	return merged, nil
}

func (stack layers) lifted(merged *projectfile.File) []string {
	// What the team and private files run, the user layer switches off, and an override enables.
	team := *stack.private
	withUser := *merged
	team.Overrides, withUser.Overrides = nil, nil
	teamOnly := bareConfig(&team, stack.preset)
	stacked := bareConfig(&withUser, stack.preset)
	return slices.DeleteFunc(stack.enabledByOverrides(), func(id string) bool {
		return !teamOnly.Enabled(id) || stacked.Enabled(id)
	})
}

func bareConfig(file *projectfile.File, presetFlag string) *config.GovernanceConfig {
	cfg := withPreset(file, presetFlag)
	file.Apply(cfg)
	return cfg
}

func (stack layers) enabledByOverrides() []string {
	var ids []string
	for _, override := range stack.private.Overrides {
		for id, entry := range override.Analyzers {
			if entry.Enabled != nil && *entry.Enabled {
				project, _, _ := strings.Cut(id, "/")
				ids = append(ids, project)
			}
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
