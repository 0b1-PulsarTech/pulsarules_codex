package loader

import (
	"cmp"
	"maps"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func privateWinsOverrides(
	base, private *projectfile.File, presetFlag string,
) []projectfile.OverrideEntry {
	// The private file is clone-local and never published, so what it switches off stays off.
	isOff := privateDisables(base, private, presetFlag)
	kept := make([]projectfile.OverrideEntry, 0, len(base.Overrides))
	for _, entry := range base.Overrides {
		trimmed := maps.Clone(entry.Analyzers)
		maps.DeleteFunc(trimmed, func(id string, _ projectfile.OverrideRules) bool {
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

func privateDisables(
	base, private *projectfile.File, presetFlag string,
) func(id string) bool {
	// Only a choice made in the private file drops a team override: the flag alone never does.
	chosen := presetDisabled(private.Preset)
	effective := presetDisabled(cmp.Or(presetFlag, private.Preset))
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
