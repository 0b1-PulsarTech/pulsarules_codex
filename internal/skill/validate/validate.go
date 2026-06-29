package validate

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

type Result struct {
	Errors []string
}

func (result Result) OK() bool { return len(result.Errors) == 0 }

type Check func(*knowledge.Index) []string

func Validate(idx *knowledge.Index, extra ...Check) Result {
	pipeline := []Check{
		ruleDependencies,
		patternDependencies,
		patternComposes,
		referencesResolve,
		skillCompositions,
		skillBodies,
		skillSidecars,
		profileOverrides,
		routerPresent,
		emojiThemeShortcodes,
	}
	var result Result
	for _, step := range pipeline {
		result.Errors = append(result.Errors, step(idx)...)
	}
	// Extra lets a caller pass a check needing unexported state (e.g. render.LintSections).
	for _, step := range extra {
		result.Errors = append(result.Errors, step(idx)...)
	}
	return result
}
