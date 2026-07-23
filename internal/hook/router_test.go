package hook

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
			router := newFileRouter(fakeRouterIndex())
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
		router *fileRouter
	}{
		{"nil index", newFileRouter(nil)},
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
