package evals

import (
	"testing"
)

// runCheckCase is one row shared by TestRunCheck and TestRunCheck_FailsClosed.
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

// TestRunCheck_FailsClosed asserts an unusable check (a pattern that will
// not compile, or a CheckType this harness does not know) never reports a
// pass by default - an eval scenario grades as failed, not skipped, when the
// check itself is broken.
func TestRunCheck_FailsClosed(t *testing.T) {
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

func TestGradeAssertion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		assertion  Assertion
		artifact   string
		wantStatus Status
	}{
		{
			name:       "judge assertion always needs a judge",
			assertion:  Assertion{ID: "1", Kind: KindJudge},
			artifact:   "anything",
			wantStatus: StatusNeedsJudge,
		},
		{
			name:       "machine assertion with no check needs a judge (validate.Check should have caught this)",
			assertion:  Assertion{ID: "1", Kind: KindMachine},
			artifact:   "anything",
			wantStatus: StatusNeedsJudge,
		},
		{
			name: "machine assertion with a passing check passes",
			assertion: Assertion{
				ID:    "1",
				Kind:  KindMachine,
				Check: &Check{Type: CheckContains, Pattern: "ok"},
			},
			artifact:   "it is ok",
			wantStatus: StatusPass,
		},
		{
			name: "machine assertion with a failing check fails",
			assertion: Assertion{
				ID:    "1",
				Kind:  KindMachine,
				Check: &Check{Type: CheckContains, Pattern: "ok"},
			},
			artifact:   "nope",
			wantStatus: StatusFail,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := gradeAssertion(testCase.assertion, testCase.artifact)
			if got.Status != testCase.wantStatus {
				t.Fatalf("status = %v, want %v", got.Status, testCase.wantStatus)
			}
		})
	}
}

// TestGrade_TrapBites is the bite-proof for the grader: the real committed
// "rate-limiter-fake-clock" scenario (integration-tests skill) against one
// artifact that falls straight into its trap (a now func() clock field, plus
// a real time.Sleep, no synctest) and one that avoids it (drives the window
// reset through testing/synctest). The machine assertions must disagree.
func TestGrade_TrapBites(t *testing.T) {
	t.Parallel()

	scenarios, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var scenario Scenario
	found := false
	for _, candidate := range scenarios {
		if candidate.Skill == "integration-tests" && candidate.ID == "rate-limiter-fake-clock" {
			scenario, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("expected the rate-limiter-fake-clock scenario to be loaded")
	}

	const trapped = `
type RateLimiter struct {
	now func() time.Time
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)
	rl.Allow()
	time.Sleep(60 * time.Millisecond)
	if !rl.Allow() {
		t.Fatal("expected reset after window")
	}
}
`
	const clean = `
func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rl := NewRateLimiter(1, 50*time.Millisecond)
		rl.Allow()
		time.Sleep(60 * time.Millisecond)
		synctest.Wait()
		if !rl.Allow() {
			t.Fatal("expected reset after window")
		}
	})
}
`

	trappedResult := Grade(scenario, trapped)
	trappedPassed, trappedTotal := trappedResult.MachineTally()
	if trappedPassed != 0 {
		t.Fatalf(
			"trapped artifact: expected 0/%d machine assertions to pass, got %d",
			trappedTotal,
			trappedPassed,
		)
	}

	cleanResult := Grade(scenario, clean)
	cleanPassed, cleanTotal := cleanResult.MachineTally()
	if cleanPassed != cleanTotal {
		t.Fatalf(
			"clean artifact: expected %d/%d machine assertions to pass, got %d",
			cleanTotal,
			cleanTotal,
			cleanPassed,
		)
	}
}
