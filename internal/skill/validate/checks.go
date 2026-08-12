package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// composition is one set of references an item composes: the references, the
// resolver that tests each, and the reporter that describes a missing one. Each
// composition is a small strategy carried through the shared check engine, so
// every reference check (rule deps, pattern composes, skill compositions, skill
// bodies) collapses to "build the compositions, run them".
type composition struct {
	refs    []string
	resolve func(id string) bool
	missing func(ref string) string
}

// runCompositions is the generic engine shared by every reference check: for
// each composition, report every reference its resolver rejects.
func runCompositions(comps []composition) []string {
	var problems []string
	for _, comp := range comps {
		for _, ref := range comp.refs {
			if !comp.resolve(ref) {
				problems = append(problems, comp.missing(ref))
			}
		}
	}
	return problems
}

// ruleSummaries reports every rule whose body carries no "> " blockquote
// summary directly under its H1. StageRuleInjection surfaces this summary in
// findings, so a rule authored without one silently prints nothing there.
func ruleSummaries(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Rules))
	for _, rule := range idx.Rules {
		if idx.Summary("rules", rule.ID) == "" {
			problems = append(problems, fmt.Sprintf("rule %q has no blockquote summary", rule.ID))
		}
	}
	return problems
}

func ruleDependencies(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Rules))
	for _, rule := range idx.Rules {
		comps := []composition{{
			refs:    rule.Dependencies,
			resolve: idx.RuleExists,
			missing: func(ref string) string {
				return fmt.Sprintf("rule %q depends on unknown rule %q", rule.ID, ref)
			},
		}}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}

func patternDependencies(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Patterns))
	for _, pattern := range idx.Patterns {
		comps := []composition{{
			refs:    pattern.Dependencies,
			resolve: idx.RuleExists,
			missing: func(ref string) string {
				return fmt.Sprintf("pattern %q depends on unknown rule %q", pattern.ID, ref)
			},
		}}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}

func patternComposes(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Patterns))
	for _, pattern := range idx.Patterns {
		comps := []composition{{
			refs:    pattern.Composes,
			resolve: idx.PatternExists,
			missing: func(ref string) string {
				return fmt.Sprintf("pattern %q composes unknown pattern %q", pattern.ID, ref)
			},
		}}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}

func skillCompositions(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		comps := []composition{
			{
				refs:    skill.ComposeRules,
				resolve: idx.RuleExists,
				missing: missingRef(skill.ID, "rule"),
			},
			{
				refs:    skill.ComposePatterns,
				resolve: idx.PatternExists,
				missing: missingRef(skill.ID, "pattern"),
			},
			{
				refs:    skill.ComposeWorkflows,
				resolve: idx.WorkflowExists,
				missing: missingRef(skill.ID, "workflow"),
			},
			{
				refs:    skill.ComposeSkills,
				resolve: idx.SkillExists,
				missing: missingRef(skill.ID, "skill"),
			},
		}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}

func skillBodies(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		comps := []composition{
			{
				refs:    skill.ComposeRules,
				resolve: hasBody(idx, "rules"),
				missing: emptyBody(skill.ID, "rule"),
			},
			{
				refs:    skill.ComposePatterns,
				resolve: hasBody(idx, "patterns"),
				missing: emptyBody(skill.ID, "pattern"),
			},
			{
				refs:    skill.ComposeWorkflows,
				resolve: hasBody(idx, "workflows"),
				missing: emptyBody(skill.ID, "workflow"),
			},
		}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}
