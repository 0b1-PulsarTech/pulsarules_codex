package loader

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func privateRules(rules projectfile.Rules, base *projectfile.File) projectfile.Rules {
	if base == nil {
		return rules
	}
	// Clipped once, so every append copies and never writes into the caller's spare capacity.
	ids := slices.Clip(rules.AnalyzerIDs)
	// The private file may tune a command only the versioned file defines.
	for _, entry := range base.Delegated {
		ids = append(ids, entry.ID)
	}
	rules.AnalyzerIDs = ids
	return rules
}
