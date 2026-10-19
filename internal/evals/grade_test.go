package evals

import (
	"testing"
)

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
			name: "machine assertion with no check " +
				"needs a judge (validate.Check should have caught this)",
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

func TestGrade_RateLimiterScenario(t *testing.T) {
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
