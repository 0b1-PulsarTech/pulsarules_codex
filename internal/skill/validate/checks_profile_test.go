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

func writeFixtureFile(tb testing.TB, path, content string) {
	tb.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatalf("write %s: %v", path, err)
	}
}
