package hook

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func fakeRouterIndex(t *testing.T) *knowledge.Index {
	t.Helper()
	idx, err := knowledge.NewIndex([]knowledge.Skill{
		{
			ID:        "go-style",
			Languages: []string{"go"},
			Triggers:  []string{"writing or editing any .go file"},
		},
		{
			ID:        "errors-logging",
			Languages: []string{"go"},
			Triggers:  []string{"returning / wrapping / mapping an error"},
		},
		{
			ID:        "code-placement",
			Languages: []string{"go"},
			Triggers:  []string{"deciding where a file or package belongs"},
		},
		{ID: "dart-style", Languages: []string{"dart"}},
		{ID: "dart-errors", Languages: []string{"dart"}},
		{ID: "dart-testing", Languages: []string{"dart"}},
		{ID: "ts-errors", Languages: []string{"typescript", "javascript", "svelte"}},
		{
			ID:       "code-minimalism",
			Triggers: []string{"writing or editing any function/method body"},
		},
		{
			ID:       "database-persistence",
			Triggers: []string{"writing a .sql query or generating migrations"},
		},
		{ID: "integration-tests", Triggers: []string{"writing unit tests for business logic"}},
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	idx.Router = knowledge.RouterSpec{
		Baseline: knowledge.RouterBaseline{
			Always: []knowledge.RouterBaselineEntry{
				{Skill: "go-style"},
				{Skill: "errors-logging"},
				{Skill: "code-placement"},
				{Skill: "code-minimalism"},
				{Skill: "dart-style"},
				{Skill: "dart-errors"},
				{Skill: "ts-errors"},
			},
		},
		Dispatch: []knowledge.RouterDispatchRow{
			{Signal: "Any test work", Skills: []string{"integration-tests"}},
			{Signal: "Writing or editing a _test.dart file", Skills: []string{"dart-testing"}},
		},
	}
	return idx
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
		{
			"dart file routes the Dart baseline", "lib/a.dart",
			[]string{"code-minimalism", "dart-style", "dart-errors"},
		},
		{"svelte file routes ts-errors", "src/A.svelte", []string{"code-minimalism", "ts-errors"}},
		{"sql file routes database-persistence", "query.sql", []string{"database-persistence"}},
		{"unmatched extension routes nothing", "README.md", nil},
		{"extensionless file routes nothing", "Makefile", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			router := newFileRouter(fakeRouterIndex(t))
			got := router.SkillsForFile(testCase.filePath)
			if !sameSet(got, testCase.want) {
				t.Fatalf("SkillsForFile(%q) = %v, want %v", testCase.filePath, got, testCase.want)
			}
		})
	}
}

func TestRouterRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		router   *fileRouter
		filePath string
		want     bool
	}{
		{"go file", newFileRouter(fakeRouterIndex(t)), "pkg/foo.go", true},
		{"sql file", newFileRouter(fakeRouterIndex(t)), "query.sql", true},
		{"ts file has no trigger", newFileRouter(fakeRouterIndex(t)), "src/a.ts", false},
		{"dart file has no trigger", newFileRouter(fakeRouterIndex(t)), "lib/a.dart", false},
		{"markdown", newFileRouter(fakeRouterIndex(t)), "README.md", false},
		{"extensionless file", newFileRouter(fakeRouterIndex(t)), "Makefile", false},
		{"go file with no baseline", newFileRouter(&knowledge.Index{}), "pkg/foo.go", false},
		{"nil index", newFileRouter(nil), "pkg/foo.go", false},
		{"nil router", nil, "pkg/foo.go", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.router.Routes(testCase.filePath); got != testCase.want {
				t.Fatalf("Routes(%q) = %v, want %v", testCase.filePath, got, testCase.want)
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
