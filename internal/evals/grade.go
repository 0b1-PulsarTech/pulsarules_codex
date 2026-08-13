package evals

import (
	"regexp"
	"strings"
)

type Status string

const (
	StatusPass       Status = "pass"
	StatusFail       Status = "fail"
	StatusNeedsJudge Status = "needs_judge"
)

type AssertionResult struct {
	ID     string
	Text   string
	Kind   Kind
	Status Status
}

type ScenarioResult struct {
	ScenarioID string
	Skill      string
	Results    []AssertionResult
}

func (outcome ScenarioResult) MachineTally() (passed, total int) {
	for _, result := range outcome.Results {
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

func regexMatches(pattern, artifact string) bool {
	// Compiled per call: eval data is small and grading is not a hot path.
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(artifact)
}
