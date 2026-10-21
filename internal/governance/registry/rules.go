package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func ConfigRules() projectfile.Rules {
	return projectfile.Rules{
		AnalyzerIDs:      configurableIDs(),
		ReservedIDs:      append(specIDs(), config.AnalyzerID),
		Params:           specParams(),
		DelegatedFormats: command.Formats(),
	}
}

func Rules() map[string]string {
	rules := make(map[string]string)
	for _, spec := range specs() {
		if spec.Rule != "" {
			rules[spec.ID] = spec.Rule
		}
	}
	return rules
}
