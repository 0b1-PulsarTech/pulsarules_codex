package evals

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// ValidateCheck reports every problem in the committed eval scenarios: a
// scenario whose skill does not resolve, one with no trap or assertions, a
// duplicate scenario id, or an assertion with unknown Kind or a KindMachine
// assertion with no Check. Matches validate.Check's shape so callers pass it
// via the extra seam instead of validate importing this package.
func ValidateCheck(idx *knowledge.Index) []string {
	scenarios, err := Load()
	if err != nil {
		return []string{fmt.Sprintf("evals: %v", err)}
	}

	var problems []string
	seenIDs := make(map[string]bool, len(scenarios))
	for _, scenario := range scenarios {
		problems = append(problems, validateScenario(idx, scenario, seenIDs)...)
	}
	return problems
}

func validateScenario(idx *knowledge.Index, scenario Scenario, seenIDs map[string]bool) []string {
	var problems []string
	label := fmt.Sprintf("eval scenario %q (skill %q)", scenario.ID, scenario.Skill)

	if !idx.SkillExists(scenario.Skill) {
		problems = append(
			problems,
			fmt.Sprintf("%s declares unknown skill %q", label, scenario.Skill),
		)
	}

	key := scenario.Skill + "/" + scenario.ID
	if seenIDs[key] {
		problems = append(
			problems,
			fmt.Sprintf("%s is a duplicate scenario id for this skill", label),
		)
	}
	seenIDs[key] = true

	if strings.TrimSpace(scenario.Prompt) == "" {
		problems = append(problems, fmt.Sprintf("%s has no prompt", label))
	}
	if strings.TrimSpace(scenario.Trap) == "" {
		problems = append(problems, fmt.Sprintf("%s has no trap", label))
	}
	if len(scenario.Assertions) == 0 {
		problems = append(problems, fmt.Sprintf("%s declares no assertions", label))
	}

	for _, assertion := range scenario.Assertions {
		problems = append(problems, validateAssertion(label, assertion)...)
	}
	return problems
}

func validateAssertion(scenarioLabel string, assertion Assertion) []string {
	var problems []string
	alabel := fmt.Sprintf("%s assertion %q", scenarioLabel, assertion.ID)

	if strings.TrimSpace(assertion.Text) == "" {
		problems = append(problems, fmt.Sprintf("%s has no text", alabel))
	}

	switch assertion.Kind {
	case KindMachine:
		if assertion.Check == nil {
			problems = append(
				problems,
				fmt.Sprintf("%s is machine-graded but declares no check", alabel),
			)
		}
	case KindJudge:
		if assertion.Check != nil {
			problems = append(
				problems,
				fmt.Sprintf("%s is judge-graded but declares a check", alabel),
			)
		}
	default:
		problems = append(
			problems,
			fmt.Sprintf(
				"%s has unknown kind %q (want %q or %q)",
				alabel,
				assertion.Kind,
				KindMachine,
				KindJudge,
			),
		)
	}
	return problems
}
