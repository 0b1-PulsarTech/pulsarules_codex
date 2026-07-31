package movepurity

import (
	"errors"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var errFakeDiff = errors.New("fake diff error")

func TestAnalyzer_Identity(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer(nil)
	if a.ID() != "commit-move-purity" {
		t.Fatalf("ID() = %q, want commit-move-purity", a.ID())
	}
	if a.Stage() != core.StageStatic {
		t.Fatalf("Stage() = %d, want StageStatic", a.Stage())
	}
	if a.Category() != core.CatCommit {
		t.Fatalf("Category() = %d, want CatCommit", a.Category())
	}
	if needs := a.Needs(); needs.NeedsAST || needs.NeedsGitHistory {
		t.Fatalf("Needs() = %+v, want no requirements", needs)
	}
}

type movePurityCase struct {
	name          string
	ctx           *core.AnalysisContext
	wantFindings  int
	wantSubstring string
}

var movePurityCases = []movePurityCase{
	{
		name:         "no staged renames",
		ctx:          &core.AnalysisContext{},
		wantFindings: 0,
	},
	{
		name: "pure rename alone",
		ctx: &core.AnalysisContext{
			StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
			ChangedFiles: []core.FileChange{
				{Path: "new.go", Staged: true},
			},
		},
		wantFindings: 0,
	},
	{
		name: "below-threshold rename is not a pure move",
		ctx: &core.AnalysisContext{
			StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 75}},
			ChangedFiles: []core.FileChange{
				{Path: "new.go", Staged: true},
			},
		},
		wantFindings:  1,
		wantSubstring: "not a pure move: 75% similar",
	},
	{
		name: "pure rename alongside an unrelated staged edit",
		ctx: &core.AnalysisContext{
			StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
			ChangedFiles: []core.FileChange{
				{Path: "new.go", Staged: true},
				{Path: "unrelated.go", Staged: true},
			},
		},
		wantFindings:  1,
		wantSubstring: "mixes 1 rename(s) with 1 edit(s)",
	},
	{
		name: "unstaged changes elsewhere do not count as mixed",
		ctx: &core.AnalysisContext{
			StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
			ChangedFiles: []core.FileChange{
				{Path: "new.go", Staged: true},
				{Path: "untouched.go", Staged: false},
			},
		},
		wantFindings: 0,
	},
}

func TestAnalyzer_Analyze(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer(nil)
	for _, testCase := range movePurityCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := a.Analyze(testCase.ctx)
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
			if findings[0].Severity != core.SeverityWarning {
				t.Fatalf("Severity = %d, want SeverityWarning by default", findings[0].Severity)
			}
		})
	}
}

func TestAnalyzer_Analyze_ParamsOverride(t *testing.T) {
	t.Parallel()

	ctx := &core.AnalysisContext{
		StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 92}},
		Config: &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
			"commit-move-purity": {
				Params: map[string]any{"min_similarity": 95, "severity": "error"},
			},
		}},
	}

	a := NewAnalyzer(nil)
	findings := a.Analyze(ctx)
	if len(findings) != 1 {
		t.Fatalf(
			"expected one finding once min_similarity is raised above the score, got %+v",
			findings,
		)
	}
	if findings[0].Severity != core.SeverityError {
		t.Fatalf("Severity = %d, want SeverityError when configured", findings[0].Severity)
	}
}

// fakeDiffReader hand-rolls diffReader (two methods, so per the
// integration-tests threshold this stays a struct rather than a generated
// mockgen mock): diffs keys a rename pair as "old->new" and a standalone
// path as itself.
type fakeDiffReader struct {
	diffs map[string]string
	err   error
}

func (f fakeDiffReader) StagedDiff(path string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.diffs[path], nil
}

func (f fakeDiffReader) StagedRenameDiff(oldPath, newPath string, _ int) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.diffs[oldPath+"->"+newPath], nil
}

func TestAnalyzer_Analyze_ImportOnlyEditDoesNotCountAsMixed(t *testing.T) {
	t.Parallel()

	diffs := fakeDiffReader{diffs: map[string]string{
		"importer.go": `-	"repo/old/thing"` + "\n" + `+	"repo/new/thing"` + "\n",
	}}
	a := NewAnalyzer(diffs)

	ctx := &core.AnalysisContext{
		StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
		ChangedFiles: []core.FileChange{
			{Path: "new.go", Staged: true},
			{Path: "importer.go", Staged: true},
		},
	}
	if findings := a.Analyze(ctx); len(findings) != 0 {
		t.Fatalf("Analyze() = %+v, want no findings for a pure import-path edit", findings)
	}
}

func TestAnalyzer_Analyze_RealEditAlongsideRenameStillCounts(t *testing.T) {
	t.Parallel()

	diffs := fakeDiffReader{diffs: map[string]string{
		"other.go": "+var x = 1\n",
	}}
	a := NewAnalyzer(diffs)

	ctx := &core.AnalysisContext{
		StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
		ChangedFiles: []core.FileChange{
			{Path: "new.go", Staged: true},
			{Path: "other.go", Staged: true},
		},
	}
	findings := a.Analyze(ctx)
	if len(findings) != 1 {
		t.Fatalf("Analyze() = %+v, want one mixed-changeset finding", findings)
	}
	if !strings.Contains(findings[0].Message, "mixes") {
		t.Fatalf("Analyze()[0].Message = %q, want it to mention the mix", findings[0].Message)
	}
}

func TestAnalyzer_Analyze_RenameOwnFileGainsAnEdit(t *testing.T) {
	t.Parallel()

	diffs := fakeDiffReader{diffs: map[string]string{
		"old.go->new.go": "+var extra = 1\n",
	}}
	a := NewAnalyzer(diffs)

	ctx := &core.AnalysisContext{
		StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
		ChangedFiles: []core.FileChange{
			{Path: "new.go", Staged: true},
		},
	}
	findings := a.Analyze(ctx)
	if len(findings) != 1 {
		t.Fatalf("Analyze() = %+v, want one carries-an-edit finding", findings)
	}
	if !strings.Contains(findings[0].Message, "carries an edit beyond the move") {
		t.Fatalf(
			"Analyze()[0].Message = %q, want it to mention the extra edit",
			findings[0].Message,
		)
	}
}

func TestAnalyzer_Analyze_DiffFetchErrorCountsAsEdit(t *testing.T) {
	t.Parallel()

	diffs := fakeDiffReader{err: errFakeDiff}
	a := NewAnalyzer(diffs)

	ctx := &core.AnalysisContext{
		StagedRenames: []core.Rename{{OldPath: "old.go", NewPath: "new.go", Score: 100}},
		ChangedFiles: []core.FileChange{
			{Path: "new.go", Staged: true},
			{Path: "other.go", Staged: true},
		},
	}
	findings := a.Analyze(ctx)
	if len(findings) != 2 {
		t.Fatalf(
			"Analyze() = %+v, want both the carries-an-edit and mixed-changeset findings on a diff-fetch error",
			findings,
		)
	}
}
