package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestCanonicalSections_EmbeddedIsClean(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := canonicalSections(idx); len(problems) != 0 {
		t.Errorf("embedded index should lint clean, got %v", problems)
	}
}

func TestCanonicalSections_Fixture(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want string
	}{
		{"canonical define", `{{define "must"}}x{{end}}`, ""},
		{
			"non-canonical define", `{{define "musty"}}x{{end}}`,
			`rules "fixture-rule" defines non-canonical section "musty"`,
		},
		{
			"malformed template", `{{define bad`,
			`rules "fixture-rule": parse body:`,
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
			rules := filepath.Join(standards, "rules")
			writeFixtureFile(t, filepath.Join(rules, "fixture-rule.md"),
				"---\nid: fixture-rule\nname: Fixture Rule\ndescription: d\n---\n\n"+
					"# Fixture Rule\n\n"+testCase.body+"\n")
			writeFixtureFile(t, filepath.Join(root, "knowledge", "standards", "skills.yaml"),
				"skills:\n  - id: s\n    name: S\n    description: d\n")
			idx, _, err := knowledge.Load(root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			problems := canonicalSections(idx)
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
