package analysis

import (
	"os"
	"path/filepath"
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
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged)
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with no repo and no changes, got %d: %+v",
			len(findings),
			findings,
		)
	}
}

func TestSession_NilConfigAndKnowledge_ScopeChanged(t *testing.T) {
	t.Parallel()

	sess := NewSession(nil, "", nil, nil)
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged)
	if len(findings) != 0 {
		t.Fatalf(
			"expected 0 findings with nil config, knowledge and repo, got %d: %+v",
			len(findings),
			findings,
		)
	}
}

func TestSession_ScopeChanged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	emDash := "\u2014"
	content := "package x\n\n// bad note " + emDash + " trips no-em-dash\n"
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
	findings := sess.Analyze(ScopeChanged, nil, FileSetChanged)

	foundStatic := false
	for _, finding := range findings {
		if finding.AnalyzerID == "no-em-dash" {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Fatal("expected a no-em-dash finding over the changed file")
	}
}
