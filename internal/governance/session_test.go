package governance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestSession_NilRepo_ScopeChanged(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()

	sess := NewSession(nil, idx, testCatalog(t), cfg)
	findings := sess.Analyze(Request{Scope: ScopeChanged}).Findings
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with no repo and no changes, got %d: %+v",
			len(findings),
			findings,
		)
	}
}

func TestSession_WithCommitMsg(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	sess := NewSession(repo, idx, testCatalog(t), cfg)
	findings := sess.Analyze(Request{Scope: ScopeFull, CommitMsg: "bad commit message"}).Findings

	found := false
	for _, finding := range findings {
		if finding.AnalyzerID == "commit-emoji-required" ||
			finding.AnalyzerID == "commit-type-required" {
			found = true
			if finding.RuleSummary == "" {
				t.Error("expected RuleSummary to be populated for commit finding")
			}
		}
	}
	if !found {
		t.Error("expected at least one commit-lint finding")
	}
}

func TestSession_NilConfigAndKnowledge_ScopeChanged(t *testing.T) {
	t.Parallel()

	sess := NewSession(nil, nil, testCatalog(t), nil)
	findings := sess.Analyze(Request{Scope: ScopeChanged}).Findings
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with nil config, knowledge and repo, got %d: %+v",
			len(findings),
			findings,
		)
	}
}

func TestSession_ScopeCommit(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	sess := NewSession(nil, idx, testCatalog(t), nil)
	findings := sess.Analyze(Request{Scope: ScopeCommit, CommitMsg: "bad commit message"}).Findings
	if len(findings) == 0 {
		t.Fatal("expected findings for invalid commit message")
	}
}

func TestSession_ScopeChanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	emDash := "\u2014"
	content := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
	if err := os.WriteFile(
		filepath.Join(dir, "violation.go"), []byte(content), 0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()

	sess := NewSession(repo, nil, testCatalog(t), cfg)
	findings := sess.Analyze(Request{Scope: ScopeChanged}).Findings

	foundStatic := false
	for _, finding := range findings {
		if finding.AnalyzerID == "golangci-lint" {
			t.Fatalf(
				"ScopeChanged must not invoke external-tool delegation, got analyzer %q",
				finding.AnalyzerID,
			)
		}
		if finding.AnalyzerID == "typographic-markers" {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Fatal("expected a typographic-markers finding over the changed file")
	}
}

func TestSession_FileSetAll_CommittedDefect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	emDash := "\u2014"
	content := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
	writeFileAt(t, dir, "violation.go", content)
	runGitAt(t, dir, "add", "violation.go")
	commitAllowEmpty(t, dir, ":sparkles: feat: Add violation.go")

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()

	changedFindings := NewSession(repo, nil, testCatalog(t), cfg).
		Analyze(Request{Scope: ScopeChanged}).Findings
	for _, finding := range changedFindings {
		if finding.AnalyzerID == "typographic-markers" {
			t.Fatalf(
				"FileSetChanged on a clean tree must not see committed defects, got %+v",
				finding,
			)
		}
	}

	allFindings := NewSession(repo, nil, testCatalog(t), cfg).
		Analyze(Request{Scope: ScopeChanged, Files: FileSetAll}).Findings
	foundAll := false
	for _, finding := range allFindings {
		if finding.AnalyzerID == "typographic-markers" && finding.File == "violation.go" {
			foundAll = true
		}
	}
	if !foundAll {
		t.Fatalf("expected FileSetAll to report the committed defect, got %+v", allFindings)
	}
}

func TestSession_ScopeCommit_StaticSources(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	content := "package x\n\n// bad note \u2014 trips typographic-markers\n"
	if err := os.WriteFile(
		filepath.Join(dir, "violation.go"), []byte(content), 0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGitAt(t, dir, "add", "violation.go")

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	sess := NewSession(repo, nil, testCatalog(t), nil)
	findings := sess.Analyze(Request{Scope: ScopeCommit}).Findings

	for _, finding := range findings {
		if finding.AnalyzerID == "typographic-markers" {
			return
		}
	}
	t.Fatalf("expected a typographic-markers finding over the changed file, got %+v", findings)
}

func TestSession_ScopeCommit_EmptyStatus(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	content := "package x\n\n// bad note \u2014 would trip typographic-markers\n"
	if err := os.WriteFile(
		filepath.Join(dir, "violation.go"), []byte(content), 0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	sess := NewSession(repo, nil, testCatalog(t), nil)
	findings := sess.Analyze(Request{Scope: ScopeCommit, Status: &vcs.Status{}}).Findings
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings with an explicit empty status, got %+v", findings)
	}
}

func TestSession_PackageLevelVarState(t *testing.T) {
	t.Parallel()

	const source = "package p\n\n" +
		"var helperCount = 0\n\n" +
		"var sharedLimit = 0\n\n" +
		"func Track() int {\n\thelperCount++\n\tsharedLimit++\n\treturn sharedLimit\n}\n\n" +
		"func Reset() {\n\tsharedLimit = 0\n}\n"

	testCases := []struct {
		name               string
		disableSingleUse   bool
		wantSingleUseLines []int
		wantPackageVarLine []int
	}{
		{
			name:               "single-use-declaration enabled claims the one-function var",
			wantSingleUseLines: []int{3},
			wantPackageVarLine: []int{5},
		},
		{
			name: "single-use-declaration disabled " +
				"leaves both vars to package-level-var",
			disableSingleUse:   true,
			wantSingleUseLines: nil,
			wantPackageVarLine: []int{3, 5},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repoDir := t.TempDir()
			if err := os.WriteFile(
				filepath.Join(repoDir, "p.go"),
				[]byte(source),
				0o600,
			); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			cfg := config.Defaults()
			cfg.ApplyPreset()
			if testCase.disableSingleUse {
				cfg.Analyzers["single-use-declaration"] = pass.AnalyzerConfig{Enabled: false}
			}
			result := NewSession(&goldenRepo{root: repoDir}, nil, testCatalog(t), cfg).
				Analyze(Request{Scope: ScopeChanged, Files: FileSetAll})

			linesByAnalyzer := make(map[string][]int)
			for _, finding := range result.Findings {
				linesByAnalyzer[finding.AnalyzerID] = append(
					linesByAnalyzer[finding.AnalyzerID], finding.Line,
				)
			}
			got := linesByAnalyzer["single-use-declaration"]
			if !slices.Equal(got, testCase.wantSingleUseLines) {
				t.Errorf(
					"single-use-declaration lines = %v, want %v",
					got,
					testCase.wantSingleUseLines,
				)
			}
			got = linesByAnalyzer["package-level-var"]
			if !slices.Equal(got, testCase.wantPackageVarLine) {
				t.Errorf("package-level-var lines = %v, want %v", got, testCase.wantPackageVarLine)
			}
		})
	}
}

func writeTestClassifierRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		writeFileAtDir(t, dir, "src/"+name+".ts", "export {};\n")
	}
	writeFileAtDir(t, dir, "src/big.ts", strings.Repeat("export {};\n", 300))
	return dir
}

type classifierRun struct {
	globs   []string
	fileSet FileSet
	status  *vcs.Status
}

func analyzeTestClassifierRepo(
	t *testing.T, dir string, run classifierRun,
) ([]finding.Finding, []pass.FileChange) {
	t.Helper()

	cfg := config.Defaults()
	cfg.ApplyPreset()
	cfg.Analyzers["package-file-count"] = pass.AnalyzerConfig{
		Enabled: true, Params: map[string]any{"extensions": []string{".ts"}},
	}
	cfg.TestGlobs = run.globs
	sess := NewSession(&goldenRepo{root: dir}, nil, testCatalog(t), cfg)
	req := Request{Scope: ScopeChanged, Status: run.status, Files: run.fileSet}
	return sess.Analyze(req).Findings, sess.load(req).ChangedFiles
}

type testGlobCase struct {
	name         string
	globs        []string
	fileSet      FileSet
	status       *vcs.Status
	wantTest     bool
	wantFileSize bool
	wantCount    bool
}

func testGlobCases() []testGlobCase {
	bigStatus := vcs.Status{Changes: []vcs.Change{{Path: "src/big.ts", Extension: ".ts"}}}
	return []testGlobCase{
		{
			name:         "no glob leaves big.ts a source",
			fileSet:      FileSetAll,
			wantFileSize: true,
			wantCount:    true,
		},
		{
			name:     "glob makes big.ts a test",
			globs:    []string{"src/big.ts"},
			fileSet:  FileSetAll,
			wantTest: true,
		},
		{
			name:         "status path without glob leaves big.ts a source",
			fileSet:      FileSetChanged,
			status:       &bigStatus,
			wantFileSize: true,
			wantCount:    true,
		},
		{
			name:     "status path with glob makes big.ts a test",
			globs:    []string{"src/big.ts"},
			fileSet:  FileSetChanged,
			status:   &bigStatus,
			wantTest: true,
		},
	}
}

func TestSession_TestGlobsReachEveryTestConsumer(t *testing.T) {
	t.Parallel()

	for _, testCase := range testGlobCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			run := classifierRun{
				globs: testCase.globs, fileSet: testCase.fileSet, status: testCase.status,
			}
			findings, changed := analyzeTestClassifierRepo(t, writeTestClassifierRepo(t), run)
			gotTest := slices.ContainsFunc(changed, func(file pass.FileChange) bool {
				return file.Path == "src/big.ts" && file.IsTest
			})
			gotSize, gotCount := false, false
			for _, finding := range findings {
				gotSize = gotSize ||
					finding.AnalyzerID == "file-size" && finding.File == "src/big.ts"
				gotCount = gotCount || finding.AnalyzerID == "package-file-count"
			}
			if gotTest != testCase.wantTest || gotSize != testCase.wantFileSize ||
				gotCount != testCase.wantCount {
				t.Fatalf("IsTest, file-size, package-file-count = %v, %v, %v; want %v, %v, %v",
					gotTest, gotSize, gotCount,
					testCase.wantTest, testCase.wantFileSize, testCase.wantCount)
			}
		})
	}
}

func TestSession_InvalidTestGlob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFileAtDir(t, dir, "web/test/a.mjs", "export {};\n")
	findings, changed := analyzeTestClassifierRepo(
		t, dir, classifierRun{globs: []string{"a/[b"}, fileSet: FileSetAll},
	)

	var configErrors []finding.Finding
	for _, finding := range findings {
		if finding.AnalyzerID == "config" {
			configErrors = append(configErrors, finding)
		}
	}
	if len(configErrors) != 1 {
		t.Fatalf("config findings = %+v, want exactly one", configErrors)
	}
	got := configErrors[0]
	if got.Severity != finding.SeverityError || got.File != ".pulsarules.yaml" ||
		!strings.Contains(got.Message, `"a/[b"`) {
		t.Fatalf("config finding = %+v", got)
	}
	if len(changed) != 1 || !changed[0].IsTest {
		t.Fatalf("defaults must still classify web/test/a.mjs as a test, got %+v", changed)
	}
}
