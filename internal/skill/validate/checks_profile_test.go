package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestProfileOverrides(t *testing.T) {
	t.Parallel()

	idx := newProfileOverridesFixture(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "override of an unknown skill reported",
			idx: &knowledge.Index{Profiles: []knowledge.Profile{
				{ID: "ghost-profile", Overrides: map[string]knowledge.SkillOverride{
					"ghost-skill": {ComposeRules: []string{"whatever"}},
				}},
			}},
			wantProblem: `profile "ghost-profile" overrides unknown skill "ghost-skill"`,
		},
		{
			name: "override composing a rule with an empty body reported",
			idx:  idx,
			wantProblem: `profile "empty-ref" override of skill "base-skill" ` +
				`composes missing/empty rule "hollow-rule"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := profileOverrides(testCase.idx)
			if len(problems) == 0 ||
				!strings.Contains(strings.Join(problems, "\n"), testCase.wantProblem) {
				t.Fatalf(
					"expected a problem containing %q, got %v", testCase.wantProblem, problems,
				)
			}
		})
	}

	t.Run("override composing a rule with a real body is silent", func(t *testing.T) {
		t.Parallel()

		problems := profileOverrides(idx)
		for _, problem := range problems {
			if strings.Contains(problem, `"good"`) {
				t.Errorf("profile %q override should resolve cleanly, got %v", "good", problems)
			}
		}
	})
}

func newProfileOverridesFixture(tb testing.TB) *knowledge.Index {
	tb.Helper()

	root := tb.TempDir()
	standards := filepath.Join(root, "knowledge", "standards")
	for _, dir := range []string{"rules", "patterns", "workflows"} {
		if err := os.MkdirAll(filepath.Join(standards, dir), 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeFixtureFile(tb, filepath.Join(standards, "rules", "solid-rule.md"), `---
id: solid-rule
name: Solid Rule
description: fixture rule carrying a real body
---

# Solid Rule

{{define "must"}}
Do the fixture thing.
{{end}}
`)
	writeFixtureFile(tb, filepath.Join(standards, "rules", "hollow-rule.md"), `---
id: hollow-rule
name: Hollow Rule
description: fixture rule carrying an empty body
---`)
	writeFixtureFile(tb, filepath.Join(standards, "skills.yaml"), `skills:
  - id: base-skill
    name: Base Skill
    description: fixture skill a profile overrides
    compose_rules: [solid-rule]
`)
	writeFixtureFile(tb, filepath.Join(standards, "profiles.yaml"), `profiles:
  - id: good
    axis: fixture-axis
    description: overrides base-skill to a rule with a real body
    overrides:
      base-skill:
        compose_rules: [solid-rule]
  - id: empty-ref
    axis: fixture-axis
    description: overrides base-skill to a rule with an empty body
    overrides:
      base-skill:
        compose_rules: [hollow-rule]
`)

	idx, _, err := knowledge.Load(root)
	if err != nil {
		tb.Fatalf("Load fixture: %v", err)
	}
	return idx
}

func TestOrientationProse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "pointer only leaves nothing",
			body: "The rules below are the composed commits rule.",
			want: "",
		},
		{
			name: "orientation survives, pointer does not",
			body: "Wire an app together.\n\nThe steps are the composed bootstrap rule.",
			want: "Wire an app together.",
		},
		{
			name: "blank lines are dropped",
			body: "First line.\n\n\nSecond line.",
			want: "First line.Second line.",
		},
		{name: "empty body", body: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := orientationProse(testCase.body); got != testCase.want {
				t.Errorf("orientationProse = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestSkillSidecars_Orientation(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := skillSidecars(idx); len(problems) != 0 {
		t.Errorf("every sidecar should carry orientation, got %d problems: %v",
			len(problems), problems)
	}
}

func TestMinOrientationChars_ShortestSidecar(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	shortest, shortestID := 0, ""
	for _, skill := range idx.Skills {
		if skill.ID == knowledge.RouterSkillID {
			continue
		}
		chars := len(orientationProse(idx.Body(knowledge.KindSkill, skill.ID)))
		if shortestID == "" || chars < shortest {
			shortest, shortestID = chars, skill.ID
		}
	}
	if shortest < minOrientationChars {
		t.Errorf("skill %q carries %d chars of orientation, below the %d bar",
			shortestID, shortest, minOrientationChars)
	}
	t.Logf("shortest orientation: %q at %d chars (bar %d)",
		shortestID, shortest, minOrientationChars)
}

func TestSkillSidecars_MessageNamesTheSkill(t *testing.T) {
	t.Parallel()

	problems := skillSidecars(&knowledge.Index{
		Skills: []knowledge.Skill{{ID: "lonely", Name: "Lonely"}},
	})
	if len(problems) != 1 || !strings.Contains(problems[0], "lonely") {
		t.Errorf("a sidecar-less skill should be reported by name, got %v", problems)
	}
}
