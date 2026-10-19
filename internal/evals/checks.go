package evals

import (
	"fmt"
	"regexp"
	"strings"
)

type checkRow struct {
	validate func(check Check) error
	grade    func(check Check, artifact string) bool
}

func checkRows() map[CheckType]checkRow {
	literal := func(Check) error { return nil }
	pattern := func(check Check) error {
		if _, err := regexp.Compile(check.Pattern); err != nil {
			return fmt.Errorf("an invalid regex %q: %w", check.Pattern, err)
		}
		return nil
	}
	return map[CheckType]checkRow{
		CheckContains: {validate: literal, grade: func(check Check, artifact string) bool {
			return strings.Contains(artifact, check.Pattern)
		}},
		CheckNotContains: {validate: literal, grade: func(check Check, artifact string) bool {
			return !strings.Contains(artifact, check.Pattern)
		}},
		CheckRegexMatch: {validate: pattern, grade: func(check Check, artifact string) bool {
			return regexMatches(check.Pattern, artifact)
		}},
		CheckRegexAbsent: {validate: pattern, grade: func(check Check, artifact string) bool {
			return !regexMatches(check.Pattern, artifact)
		}},
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

func runCheck(check Check, artifact string) bool {
	row, known := checkRows()[check.Type]
	return known && row.grade(check, artifact)
}

func validateCheck(alabel string, check Check) []string {
	row, known := checkRows()[check.Type]
	if !known {
		return []string{fmt.Sprintf("%s has unknown check type %q", alabel, check.Type)}
	}
	if err := row.validate(check); err != nil {
		return []string{fmt.Sprintf("%s has %v", alabel, err)}
	}
	return nil
}
