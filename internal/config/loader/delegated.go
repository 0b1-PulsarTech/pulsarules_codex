package loader

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"

func privateRules(base *projectfile.File) projectfile.Rules {
	rules := Rules()
	if base == nil {
		return rules
	}
	// The private file may tune a command only the versioned file defines.
	for _, entry := range base.Delegated {
		rules.AnalyzerIDs = append(rules.AnalyzerIDs, entry.ID)
	}
	return rules
}
