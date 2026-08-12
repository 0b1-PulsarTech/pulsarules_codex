package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestSkillNormativeSections_Embedded asserts the committed knowledge base -
// whose contract, after the body-stripping change this check guards against,
// lives entirely in composed rules/patterns - still renders a normative
// section for every skill. This is the regression guard: it proves the check
// does not fire on the current, correct contract.
func TestSkillNormativeSections_Embedded(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := skillNormativeSections(idx); len(problems) != 0 {
		t.Errorf("embedded skills should all render a normative section, got %v", problems)
	}
}

// TestSkillNormativeSections_Fixture builds a minimal on-disk knowledge base
// with four skills - one composing a rule with a "must" section, one
// composing only a "recipe"-only pattern, one composing nothing, and
// project-router composing nothing - and asserts only the "must"-carrying
// skill and the explicitly exempt router pass.
func TestSkillNormativeSections_Fixture(t *testing.T) {
	t.Parallel()

	idx := newNormativeFixture(t)
	problems := skillNormativeSections(idx)

	testCases := []struct {
		name     string
		skillID  string
		wantFail bool
	}{
		{"composes a rule with a must section", "has-must", false},
		{"composes only a recipe-only pattern", "recipe-only", true},
		{"composes nothing", "composes-nothing", true},
		{"project-router is exempt though it composes nothing", "project-router", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := reportsSkill(problems, testCase.skillID)
			if got != testCase.wantFail {
				t.Errorf("skill %q reported = %v, want %v (problems: %v)",
					testCase.skillID, got, testCase.wantFail, problems)
			}
		})
	}
}

// TestSkillNormativeSections_ProfileOverride builds a fixture where a skill's
// base composition passes (it composes a "must"-carrying rule) but two
// profiles override that composition: one down to a "when"-only rule (no
// obligation left to render), one to a different "must"-carrying rule (still
// normative). It asserts the degrading override is reported by profile and
// skill, and the still-normative override is silent - proving the check
// follows a profile's override, not just a skill's base composition.
func TestSkillNormativeSections_ProfileOverride(t *testing.T) {
	t.Parallel()

	idx := newProfileNormativeFixture(t)
	problems := skillNormativeSections(idx)
	joined := strings.Join(problems, "\n")

	testCases := []struct {
		name     string
		want     string
		wantFail bool
	}{
		{
			"base composition still has a must section",
			`skill "overridable" renders no normative section`,
			false,
		},
		{
			"degrading profile override is reported",
			`profile "degrade" overrides skill "overridable" to render no normative section`,
			true,
		},
		{
			"non-degrading profile override is silent",
			`profile "keep" overrides skill "overridable" to render no normative section`,
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := strings.Contains(joined, testCase.want)
			if got != testCase.wantFail {
				t.Errorf("problem containing %q reported = %v, want %v (problems: %v)",
					testCase.want, got, testCase.wantFail, problems)
			}
		})
	}
}

// newProfileNormativeFixture builds a minimal on-disk knowledge base with one
// skill ("overridable") whose base composition renders a "must" section, a
// second "must"-carrying rule, a "when"-only rule with no obligation, and two
// profiles: "degrade" overrides the skill to the "when"-only rule, "keep"
// overrides it to the other "must"-carrying rule.
func newProfileNormativeFixture(t testing.TB) *knowledge.Index {
	t.Helper()

	root := t.TempDir()
	standards := filepath.Join(root, "knowledge", "standards")
	for _, dir := range []string{"rules", "patterns", "workflows"} {
		if err := os.MkdirAll(filepath.Join(standards, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeFixtureFile(t, filepath.Join(standards, "rules", "must-rule.md"), `---
id: must-rule
name: Must Rule
description: fixture rule carrying a must section
---

# Must Rule

{{define "must"}}
Do the fixture thing.
{{end}}
`)
	writeFixtureFile(t, filepath.Join(standards, "rules", "other-must-rule.md"), `---
id: other-must-rule
name: Other Must Rule
description: a second fixture rule carrying a must section
---

# Other Must Rule

{{define "must"}}
Do the other fixture thing.
{{end}}
`)
	writeFixtureFile(t, filepath.Join(standards, "rules", "when-only-rule.md"), `---
id: when-only-rule
name: When Only Rule
description: fixture rule carrying no obligation, only a when section
---

# When Only Rule

{{define "when"}}
Whenever the fixture applies.
{{end}}
`)
	writeFixtureFile(t, filepath.Join(standards, "skills.yaml"), `skills:
  - id: overridable
    name: Overridable
    description: fixture skill whose composition a profile can override
    compose_rules: [must-rule]
  - id: project-router
    name: Project Router
    description: fixture router
`)
	writeFixtureFile(t, filepath.Join(standards, "profiles.yaml"), `profiles:
  - id: degrade
    axis: fixture-axis
    description: overrides overridable down to a rule with no obligation
    overrides:
      overridable:
        compose_rules: [when-only-rule]
  - id: keep
    axis: fixture-axis
    description: overrides overridable to a different still-normative rule
    overrides:
      overridable:
        compose_rules: [other-must-rule]
`)

	idx, _, err := knowledge.Load(root)
	if err != nil {
		t.Fatalf("Load fixture: %v", err)
	}
	return idx
}

// reportsSkill reports whether problems names skillID.
func reportsSkill(problems []string, skillID string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, `"`+skillID+`"`) {
			return true
		}
	}
	return false
}

// newNormativeFixture builds a minimal on-disk knowledge base under a fresh
// t.TempDir() and loads it: one rule carrying a "must" section, one pattern
// carrying only a "recipe" section, and four skills exercising every
// normative-content outcome skillNormativeSections must distinguish.
func newNormativeFixture(t testing.TB) *knowledge.Index {
	t.Helper()

	root := t.TempDir()
	standards := filepath.Join(root, "knowledge", "standards")
	for _, dir := range []string{"rules", "patterns", "workflows"} {
		if err := os.MkdirAll(filepath.Join(standards, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeFixtureFile(t, filepath.Join(standards, "rules", "must-rule.md"), `---
id: must-rule
name: Must Rule
description: fixture rule carrying a must section
---

# Must Rule

{{define "must"}}
Do the fixture thing.
{{end}}
`)
	writeFixtureFile(t, filepath.Join(standards, "patterns", "recipe-pattern.md"), `---
id: recipe-pattern
name: Recipe Pattern
description: fixture pattern carrying only a recipe section
---

# Recipe Pattern

{{define "recipe"}}
Follow the fixture recipe.
{{end}}
`)
	writeFixtureFile(t, filepath.Join(standards, "skills.yaml"), `skills:
  - id: has-must
    name: Has Must
    description: fixture skill
    compose_rules: [must-rule]
  - id: recipe-only
    name: Recipe Only
    description: fixture skill
    compose_patterns: [recipe-pattern]
  - id: composes-nothing
    name: Composes Nothing
    description: fixture skill
  - id: project-router
    name: Project Router
    description: fixture router
`)

	idx, _, err := knowledge.Load(root)
	if err != nil {
		t.Fatalf("Load fixture: %v", err)
	}
	return idx
}

func writeFixtureFile(t testing.TB, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
