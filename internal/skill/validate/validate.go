package validate

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

type Result struct {
	Errors []string
}

func (result Result) OK() bool { return len(result.Errors) == 0 }

type Check func(*knowledge.Index) []string

func Validate(idx *knowledge.Index, extra ...Check) Result {
	pipeline := []Check{
		ruleSummaries,
		ruleDependencies,
		patternDependencies,
		patternComposes,
		referencesResolve,
		skillCompositions,
		skillBodies,
		skillSidecars,
		skillNormativeSections,
		profileOverrides,
		routerPresent,
		routerSkillIDs,
		emojiThemeShortcodes,
		canonicalSections,
	}
	var result Result
	for _, step := range pipeline {
		result.Errors = append(result.Errors, step(idx)...)
	}
	// Extra carries checks from packages validate must not import (registry, evals).
	for _, step := range extra {
		result.Errors = append(result.Errors, step(idx)...)
	}
	return result
}
