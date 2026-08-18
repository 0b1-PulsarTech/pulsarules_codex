package evals

import (
	"regexp"
	"strings"
)

// Status is the outcome Grade assigns one assertion against an artifact.
type Status string

const (
	// StatusPass is a KindMachine assertion whose Check matched the artifact.
	StatusPass Status = "pass"
	// StatusFail is a KindMachine assertion whose Check did not match the artifact.
	StatusFail Status = "fail"
	// StatusNeedsJudge is a KindJudge assertion; Grade never scores these itself.
	StatusNeedsJudge Status = "needs_judge"
)

// AssertionResult is one assertion's grade against a single artifact.
type AssertionResult struct {
	ID     string
	Text   string
	Kind   Kind
	Status Status
}

// ScenarioResult is every assertion's grade for one scenario against one
// produced artifact (a with-skill or without-skill run, graded separately).
type ScenarioResult struct {
	ScenarioID string
	Skill      string
	Results    []AssertionResult
}

// MachineTally reports how many KindMachine assertions passed out of how many
// were graded; KindJudge assertions are excluded since Grade never scores them.
func (r ScenarioResult) MachineTally() (passed, total int) {
	for _, result := range r.Results {
		if result.Kind != KindMachine {
			continue
		}
		total++
		if result.Status == StatusPass {
			passed++
		}
	}
	return passed, total
}

// Grade scores every assertion in scenario against artifact: a KindMachine
// assertion runs its Check and reports StatusPass or StatusFail; a KindJudge
// assertion reports StatusNeedsJudge untouched, since Grade has no model to
// read the artifact with - a human or LLM judge scores those separately.
func Grade(scenario Scenario, artifact string) ScenarioResult {
	results := make([]AssertionResult, 0, len(scenario.Assertions))
	for _, assertion := range scenario.Assertions {
		results = append(results, gradeAssertion(assertion, artifact))
	}
	return ScenarioResult{ScenarioID: scenario.ID, Skill: scenario.Skill, Results: results}
}

func gradeAssertion(assertion Assertion, artifact string) AssertionResult {
	result := AssertionResult{ID: assertion.ID, Text: assertion.Text, Kind: assertion.Kind}

	if assertion.Kind != KindMachine || assertion.Check == nil {
		result.Status = StatusNeedsJudge
		return result
	}

	result.Status = StatusFail
	if runCheck(*assertion.Check, artifact) {
		result.Status = StatusPass
	}
	return result
}

func runCheck(check Check, artifact string) bool {
	switch check.Type {
	case CheckContains:
		return strings.Contains(artifact, check.Pattern)
	case CheckNotContains:
		return !strings.Contains(artifact, check.Pattern)
	case CheckRegexMatch:
		return regexMatches(check.Pattern, artifact)
	case CheckRegexAbsent:
		return !regexMatches(check.Pattern, artifact)
	default:
		return false
	}
}

// regexMatches compiles pattern fresh per call: eval data is small and
// grading is not a hot path, so caching compiled patterns is not worth the
// added state for a check the harness runs at most a few hundred times.
func regexMatches(pattern, artifact string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(artifact)
}
