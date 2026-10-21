package registry

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

func Rules() map[string]string {
	rules := make(map[string]string)
	for _, spec := range specs() {
		addRules(rules, spec.Descriptor)
	}
	return rules
}

func addRules(rules map[string]string, desc pass.Descriptor) {
	if desc.Rule != "" {
		rules[desc.ID] = desc.Rule
	}
	for _, sub := range desc.Subs {
		addRules(rules, sub)
	}
}
