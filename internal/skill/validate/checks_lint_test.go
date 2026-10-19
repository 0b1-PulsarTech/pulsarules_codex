package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestNonCanonicalSections_EmbeddedIsClean(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := nonCanonicalSections(idx); len(problems) != 0 {
		t.Errorf("embedded index should lint clean, got %v", problems)
	}
}

func TestNonCanonicalSections_Fixture(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		kind knowledge.Kind
		body string
		want string
	}{
		{"canonical define", knowledge.KindRule, `{{define "must"}}x{{end}}`, ""},
		{
			"non-canonical define", knowledge.KindRule, `{{define "musty"}}x{{end}}`,
			`rules "fixture" defines non-canonical section "musty"`,
		},
		{
			"malformed template", knowledge.KindRule, `{{define bad`,
			`rules "fixture": parse body:`,
		},
		{
			"non-canonical define in a pattern",
			knowledge.KindPattern,
			`{{define "musty"}}x{{end}}`,
			`patterns "fixture" defines non-canonical section "musty"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			standards := filepath.Join(root, "knowledge", "standards")
			for _, dir := range []string{"rules", "patterns", "workflows"} {
				if err := os.MkdirAll(filepath.Join(standards, dir), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", dir, err)
				}
			}
			writeFixtureFile(t,
				filepath.Join(standards, string(testCase.kind), "fixture.md"),
				"---\nid: fixture\nname: Fixture\ndescription: d\n---\n\n"+
					"# Fixture\n\n"+testCase.body+"\n")
			writeFixtureFile(t, filepath.Join(standards, "skills.yaml"),
				"skills:\n  - id: s\n    name: S\n    description: d\n")
			idx, _, err := knowledge.Load(root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			var problems []string
			for _, problem := range Validate(idx).Errors {
				if strings.Contains(problem, string(testCase.kind)+` "fixture"`) {
					problems = append(problems, problem)
				}
			}
			if testCase.want == "" {
				if len(problems) != 0 {
					t.Errorf("problems = %v, want none", problems)
				}
				return
			}
			if len(problems) != 1 || !strings.Contains(problems[0], testCase.want) {
				t.Errorf("problems = %v, want one containing %q", problems, testCase.want)
			}
		})
	}
}
