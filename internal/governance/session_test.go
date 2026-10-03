package governance

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

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

	sess := NewSession(nil, "", idx, cfg)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged).Findings
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

	sess := NewSession(repo, "bad commit message", idx, cfg)
	findings := sess.Analyze(ScopeFull, nil, FileSetChanged).Findings

	found := false
	for _, f := range findings {
		if f.AnalyzerID == "commit-emoji-required" || f.AnalyzerID == "commit-type-required" {
			found = true
			if f.RuleSummary == "" {
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

	sess := NewSession(nil, "", nil, nil)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged).Findings
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

	sess := NewSession(nil, "bad commit message", idx, nil)
	findings := sess.Analyze(ScopeCommit, nil, FileSetChanged).Findings
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

	sess := NewSession(repo, "", nil, cfg)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged).Findings

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

	changedFindings := NewSession(repo, "", nil, cfg).
		Analyze(ScopeChanged, nil, FileSetChanged).Findings
	for _, f := range changedFindings {
		if f.AnalyzerID == "typographic-markers" {
			t.Fatalf(
				"FileSetChanged on a clean tree must not see committed defects, got %+v",
				f,
			)
		}
	}

	allFindings := NewSession(repo, "", nil, cfg).
		Analyze(ScopeChanged, nil, FileSetAll).Findings
	foundAll := false
	for _, f := range allFindings {
		if f.AnalyzerID == "typographic-markers" && f.File == "violation.go" {
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

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	sess := NewSession(repo, "", nil, nil)
	findings := sess.Analyze(ScopeCommit, nil, FileSetChanged).Findings

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

	sess := NewSession(repo, "", nil, nil)
	findings := sess.Analyze(ScopeCommit, &vcs.Status{}, FileSetChanged).Findings
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
				cfg.Analyzers["single-use-declaration"] = config.AnalyzerConfig{Enabled: false}
			}
			result := NewSession(&goldenRepo{root: repoDir}, "", nil, cfg).
				Analyze(ScopeChanged, nil, FileSetAll)

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
