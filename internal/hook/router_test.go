package hook

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// fakeRouterIndex builds a minimal knowledge.Index shaped like the real
// skills.yaml/router.yaml: a Go baseline, an extension-tagged skill, and a
// dispatch row for test work - enough to exercise Router.SkillsForFile
// without depending on the embedded knowledge base's actual content.
func fakeRouterIndex() *knowledge.Index {
	return &knowledge.Index{
		Skills: []knowledge.Skill{
			{ID: "go-style", Triggers: []string{"writing or editing any .go file"}},
			{ID: "errors-logging", Triggers: []string{"returning / wrapping / mapping an error"}},
			{ID: "code-placement", Triggers: []string{"deciding where a file or package belongs"}},
			{
				ID:       "code-minimalism",
				Triggers: []string{"writing or editing any function/method body"},
			},
			{
				ID:       "database-persistence",
				Triggers: []string{"writing a .sql query or generating migrations"},
			},
			{ID: "integration-tests", Triggers: []string{"writing unit tests for business logic"}},
		},
		Router: knowledge.RouterSpec{
			Baseline: knowledge.RouterBaseline{
				Always: []knowledge.RouterBaselineEntry{
					{Skill: "go-style"},
					{Skill: "errors-logging"},
					{Skill: "code-placement"},
					{Skill: "code-minimalism"},
				},
			},
			Dispatch: []knowledge.RouterDispatchRow{
				{Signal: "Any test work", Skills: []string{"integration-tests"}},
			},
		},
	}
}

func goBaseline() []string {
	return []string{"go-style", "errors-logging", "code-placement", "code-minimalism"}
}

func TestRouterSkillsForFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filePath string
		want     []string
	}{
		{"go file routes the Go baseline", "pkg/foo.go", goBaseline()},
		{
			"go test file adds integration-tests",
			"pkg/foo_test.go",
			append(goBaseline(), "integration-tests"),
		},
		{"sql file routes database-persistence", "query.sql", []string{"database-persistence"}},
		{"unmatched extension routes nothing", "README.md", nil},
		{"extensionless file routes nothing", "Makefile", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			router := NewRouter(fakeRouterIndex())
			got := router.SkillsForFile(testCase.filePath)
			if !sameSet(got, testCase.want) {
				t.Fatalf("SkillsForFile(%q) = %v, want %v", testCase.filePath, got, testCase.want)
			}
		})
	}
}

func TestRouterSkillsForFile_NilIndex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		router *Router
	}{
		{"nil index", NewRouter(nil)},
		{"nil router", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.router.SkillsForFile("pkg/foo.go"); got != nil {
				t.Fatalf("SkillsForFile = %v, want nil", got)
			}
		})
	}
}

func TestFilterInstalledDropsUninstalledSkills(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(skillsDir, "go-style"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillsDir, "go-style", "SKILL.md"),
		[]byte("stub"),
		0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := filterInstalled([]string{"go-style", "not-installed"}, skillsDir)
	want := []string{"go-style"}
	if !slices.Equal(got, want) {
		t.Fatalf("filterInstalled = %v, want %v", got, want)
	}
}

// sameSet reports whether got and want carry the same elements, ignoring
// order (Router.SkillsForFile's result order follows internal iteration, not
// a contract callers should depend on).
func sameSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	gotSorted := slices.Clone(got)
	wantSorted := slices.Clone(want)
	slices.Sort(gotSorted)
	slices.Sort(wantSorted)
	return slices.Equal(gotSorted, wantSorted)
}
