package loader

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func privateRules(rules projectfile.Rules, base *projectfile.File) projectfile.Rules {
	if base == nil {
		return rules
	}
	// The private file may tune a command only the versioned file defines.
	for _, entry := range base.Delegated {
		rules.AnalyzerIDs = append(slices.Clip(rules.AnalyzerIDs), entry.ID)
	}
	return rules
}
