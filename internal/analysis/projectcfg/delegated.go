package projectcfg

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/config"

func privateRules(base *config.ProjectFile) config.Rules {
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
