package validate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func newRouterFixture(tb testing.TB, routerYAML string) *knowledge.Index {
	tb.Helper()

	root := tb.TempDir()
	standards := filepath.Join(root, "knowledge", "standards")
	for _, dir := range []string{"rules", "patterns", "workflows"} {
		if err := os.MkdirAll(filepath.Join(standards, dir), 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFixtureFile(tb, filepath.Join(standards, "skills.yaml"), `skills:
  - id: project-router
    name: Project Router
    description: fixture router
  - id: go-style
    name: Go Style
    description: fixture skill
`)
	writeFixtureFile(tb, filepath.Join(standards, "router.yaml"), routerYAML)
	idx, _, err := knowledge.Load(root)
	if err != nil {
		tb.Fatalf("Load fixture: %v", err)
	}
	return idx
}

func TestRouterSkillIDs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		routerYAML string
		want       []string
	}{
		{"empty baseline", "router: {}\n", nil},
		{"known ids", "router:\n  baseline:\n    always: [{skill: go-style}]\n", nil},
		{
			"unknown always id",
			"router:\n  baseline:\n    always: [{skill: ghost}]\n",
			[]string{`router baseline always names unknown skill "ghost"`},
		},
		{
			"unknown commit_last and conditional ids",
			"router:\n  baseline:\n    commit_last: [{skill: ghost}]\n" +
				"    conditional: [{skill: spook}]\n",
			[]string{
				`router baseline commit_last names unknown skill "ghost"`,
				`router baseline conditional names unknown skill "spook"`,
			},
		},
		{
			"unknown dispatch id",
			"router:\n  dispatch:\n    - {signal: s, skills: [go-style, ghost-dispatch]}\n",
			[]string{`router dispatch names unknown skill "ghost-dispatch"`},
		},
		{
			"unknown order id",
			"router:\n  order:\n    - {skills: [ghost-order], note: n}\n",
			[]string{`router order names unknown skill "ghost-order"`},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			idx := newRouterFixture(t, testCase.routerYAML)
			if got := routerSkillIDs(idx); !slices.Equal(got, testCase.want) {
				t.Errorf("routerSkillIDs = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestValidate_RejectsUnknownRouterID(t *testing.T) {
	t.Parallel()

	idx := newRouterFixture(t, "router:\n  baseline:\n    always: [{skill: ghost}]\n")
	result := Validate(idx)
	want := `router baseline always names unknown skill "ghost"`
	if !slices.Contains(result.Errors, want) {
		t.Errorf("Validate errors = %v, want to contain %q", result.Errors, want)
	}
}
