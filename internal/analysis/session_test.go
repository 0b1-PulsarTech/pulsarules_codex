package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestSession_NoChanges(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()

	// A nil repo trips every analyzer's own guard (empty CommitMsg, nil
	// Sources, no StagedRenames), so this must return zero findings.
	// ScopeChanged, not ScopeFull, keeps it deterministic - ScopeFull also
	// runs the golangci-lint delegation analyzer against the ambient
	// toolchain, leaking environment-dependent findings.
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

func TestSession_NilConfig(t *testing.T) {
	t.Parallel()

	// NewSession substitutes config.Defaults() for a nil cfg, but a nil repo
	// still means every analyzer's guard trips, so this comes back empty
	// too. ScopeChanged keeps the result deterministic (see NoChanges above
	// for why ScopeFull's delegation analyzers would make this flaky).
	sess := NewSession(nil, "", nil, nil)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged).Findings
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with nil config and no repo, got %d: %+v",
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

func TestSession_NilKnowledge(t *testing.T) {
	t.Parallel()

	// A nil index only disables RuleSummary injection; it never manufactures a
	// finding on its own, so this is still empty with no repo and no
	// changes. ScopeChanged keeps the result deterministic (see NoChanges
	// above for why ScopeFull's delegation analyzers would make this flaky).
	sess := NewSession(nil, "", nil, nil)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged).Findings
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with nil knowledge index, got %d: %+v",
			len(findings),
			findings,
		)
	}
}

func TestSession_ScopeChanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	// An em-dash (U+2014, written via escape so this test file itself
	// carries none) trips the typographic-markers analyzer cheaply.
	emDash := "\u2014"
	content := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
	if err := os.WriteFile(filepath.Join(dir, "violation.go"), []byte(content), 0o600); err != nil {
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
	for _, f := range findings {
		if f.AnalyzerID == "golangci-lint" {
			t.Fatalf(
				"ScopeChanged must not invoke external-tool delegation, got analyzer %q",
				f.AnalyzerID,
			)
		}
		if f.AnalyzerID == "typographic-markers" {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Fatal("expected a typographic-markers finding over the changed file")
	}
}

// TestSession_FileSetAll_CleanTree proves the regression that matters: a
// defect committed to HEAD is invisible on a clean tree (nothing staged,
// nothing modified) under the default FileSetChanged discovery, since
// WorktreeStatus reports no changes - but FileSetAll walks the source tree
// regardless of git status and finds it.
func TestSession_FileSetAll_CleanTree(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	// An em-dash (U+2014, written via escape so this test file itself
	// carries none) trips the typographic-markers analyzer cheaply.
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
