package evals

import (
	"testing"
)

type runCheckCase struct {
	name     string
	check    Check
	artifact string
	want     bool
}

func TestRunCheck(t *testing.T) {
	t.Parallel()

	testCases := []runCheckCase{
		{
			name:     "contains matches substring",
			check:    Check{Type: CheckContains, Pattern: "foo"},
			artifact: "a foo b",
			want:     true,
		},
		{
			name:     "contains misses absent substring",
			check:    Check{Type: CheckContains, Pattern: "foo"},
			artifact: "a bar b",
			want:     false,
		},
		{
			name:     "not_contains passes when absent",
			check:    Check{Type: CheckNotContains, Pattern: "foo"},
			artifact: "a bar b",
			want:     true,
		},
		{
			name:     "not_contains fails when present",
			check:    Check{Type: CheckNotContains, Pattern: "foo"},
			artifact: "a foo b",
			want:     false,
		},
		{
			name:     "regex_match matches pattern",
			check:    Check{Type: CheckRegexMatch, Pattern: `^:\w+: fix:`},
			artifact: ":bug: fix: repair it",
			want:     true,
		},
		{
			name:     "regex_match misses non-matching text",
			check:    Check{Type: CheckRegexMatch, Pattern: `^:\w+: fix:`},
			artifact: "fix: repair it",
			want:     false,
		},
		{
			name:     "regex_absent passes when pattern absent",
			check:    Check{Type: CheckRegexAbsent, Pattern: `Co-Authored-By`},
			artifact: ":bug: fix: repair it",
			want:     true,
		},
		{
			name:     "regex_absent fails when pattern present",
			check:    Check{Type: CheckRegexAbsent, Pattern: `Co-Authored-By`},
			artifact: "Co-Authored-By: someone",
			want:     false,
		},
	}
	runCheckCases(t, testCases)
}

func TestRunCheck_UnusableCheck(t *testing.T) {
	t.Parallel()

	testCases := []runCheckCase{
		{
			name:     "invalid regex fails closed",
			check:    Check{Type: CheckRegexMatch, Pattern: `(unclosed`},
			artifact: "anything",
			want:     false,
		},
		{
			name:     "unknown check type fails closed",
			check:    Check{Type: "made-up", Pattern: "x"},
			artifact: "x",
			want:     false,
		},
	}
	runCheckCases(t, testCases)
}

func runCheckCases(t *testing.T, testCases []runCheckCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := runCheck(testCase.check, testCase.artifact)
			if got != testCase.want {
				t.Fatalf(
					"runCheck(%+v, %q) = %v, want %v",
					testCase.check,
					testCase.artifact,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestValidateCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		check Check
		want  string
	}{
		{"contains passes", Check{Type: CheckContains, Pattern: "("}, ""},
		{"not_contains passes", Check{Type: CheckNotContains, Pattern: "("}, ""},
		{"regex_match passes", Check{Type: CheckRegexMatch, Pattern: "a+"}, ""},
		{"regex_absent passes", Check{Type: CheckRegexAbsent, Pattern: "a+"}, ""},
		{
			"regex_match invalid",
			Check{Type: CheckRegexMatch, Pattern: "[a-"},
			`L has an invalid regex "[a-": error parsing regexp: missing closing ]: ` + "`[a-`",
		},
		{
			"regex_absent invalid",
			Check{Type: CheckRegexAbsent, Pattern: "("},
			`L has an invalid regex "(": error parsing regexp: missing closing ): ` + "`(`",
		},
		{"unknown type", Check{Type: "made-up"}, `L has unknown check type "made-up"`},
		{"empty type", Check{}, `L has unknown check type ""`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := validateCheck("L", testCase.check)
			if testCase.want == "" {
				if len(problems) != 0 {
					t.Fatalf("want no problems, got %v", problems)
				}
				return
			}
			if len(problems) != 1 || problems[0] != testCase.want {
				t.Fatalf("got %q, want [%q]", problems, testCase.want)
			}
		})
	}
}

func TestCheckRows_Complete(t *testing.T) {
	t.Parallel()

	rows := checkRows()
	for _, checkType := range []CheckType{
		CheckContains, CheckNotContains, CheckRegexMatch, CheckRegexAbsent,
	} {
		row, known := rows[checkType]
		if !known || row.validate == nil || row.grade == nil {
			t.Errorf("check type %q lacks a complete row", checkType)
		}
	}
}
