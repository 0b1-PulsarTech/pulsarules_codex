package evals

import (
	"testing"
)

func TestLoad_EmbeddedScenarios(t *testing.T) {
	t.Parallel()

	scenarios, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(scenarios) == 0 {
		t.Fatal("expected at least one scenario")
	}

	wantSkills := map[string]bool{
		"code-minimalism":   false,
		"integration-tests": false,
		"commits":           false,
		"go-style":          false,
	}
	for idx, scenario := range scenarios {
		if scenario.Skill == "" {
			t.Fatalf("scenario %q has no skill", scenario.ID)
		}
		if _, known := wantSkills[scenario.Skill]; known {
			wantSkills[scenario.Skill] = true
		}
		if idx > 0 && lessScenario(scenarios[idx], scenarios[idx-1]) {
			t.Fatalf("scenarios not sorted: %q before %q", scenarios[idx-1].ID, scenarios[idx].ID)
		}
	}
	for skill, found := range wantSkills {
		if !found {
			t.Errorf("expected at least one scenario for skill %q", skill)
		}
	}
}

func lessScenario(left, right Scenario) bool {
	if left.Skill != right.Skill {
		return left.Skill < right.Skill
	}
	return left.ID < right.ID
}
