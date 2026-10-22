package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type composition struct {
	refs    []string
	resolve func(id string) bool
	missing func(ref string) string
}

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

func ruleSummaries(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Rules))
	for _, rule := range idx.Rules {
		// rulesummary.Attach copies this summary onto findings; the CLI prints its first sentence.
		if idx.Summary(knowledge.KindRule, rule.ID) == "" {
			problems = append(problems, fmt.Sprintf("rule %q has no blockquote summary", rule.ID))
		}
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
				resolve: hasBody(idx, knowledge.KindRule),
				missing: emptyBody(skill.ID, "rule"),
			},
			{
				refs:    skill.ComposePatterns,
				resolve: hasBody(idx, knowledge.KindPattern),
				missing: emptyBody(skill.ID, "pattern"),
			},
			{
				refs:    skill.ComposeWorkflows,
				resolve: hasBody(idx, knowledge.KindWorkflow),
				missing: emptyBody(skill.ID, "workflow"),
			},
		}
		problems = append(problems, runCompositions(comps)...)
	}
	return problems
}
