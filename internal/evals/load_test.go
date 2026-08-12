package evals

import (
	"testing"
)

// TestLoad_Embedded asserts the committed data/*.json files decode into a
// non-empty, sorted scenario set covering the four skills this harness
// targets, each carrying the skill it belongs to.
func TestLoad_Embedded(t *testing.T) {
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
	for i, scenario := range scenarios {
		if scenario.Skill == "" {
			t.Fatalf("scenario %q has no skill", scenario.ID)
		}
		if _, known := wantSkills[scenario.Skill]; known {
			wantSkills[scenario.Skill] = true
		}
		if i > 0 && lessScenario(scenarios[i], scenarios[i-1]) {
			t.Fatalf("scenarios not sorted: %q before %q", scenarios[i-1].ID, scenarios[i].ID)
		}
	}
	for skill, found := range wantSkills {
		if !found {
			t.Errorf("expected at least one scenario for skill %q", skill)
		}
	}
}

// lessScenario reports whether b should sort before a, the inverse of the
// order Load guarantees; used only to detect an ordering regression.
func lessScenario(a, b Scenario) bool {
	if a.Skill != b.Skill {
		return a.Skill < b.Skill
	}
	return a.ID < b.ID
}
