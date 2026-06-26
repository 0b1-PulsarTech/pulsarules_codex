package validate

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

// Result is the outcome of Validate.
type Result struct {
	Errors []string
}

// OK reports whether validation found no problems.
func (r Result) OK() bool { return len(r.Errors) == 0 }

// Check reports the problems it found, or nothing when the index satisfies it.
type Check func(*knowledge.Index) []string

// validationPipeline is the built-in set of checks Validate always runs.
var validationPipeline = []Check{
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

// Validate runs the built-in pipeline plus every extra check, in order, and
// folds their problems into one result. A caller with a check that needs
// unexported state elsewhere (e.g. render.LintSections) passes it via extra
// instead of duplicating it inside this package.
func Validate(idx *knowledge.Index, extra ...Check) Result {
	var result Result
	for _, step := range validationPipeline {
		result.Errors = append(result.Errors, step(idx)...)
	}
	for _, step := range extra {
		result.Errors = append(result.Errors, step(idx)...)
	}
	return result
}
