package governance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func writeFileAtDir(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestSession_SkipDirs_ConfiguredUnion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	emDash := "\u2014"
	body := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
	writeFileAtDir(t, dir, "vendor/lib.go", body)
	writeFileAtDir(t, dir, "thirdparty/gen.go", body)
	writeFileAt(t, dir, "main.go", body)
	runGitAt(t, dir, "add", "-A")
	commitAllowEmpty(t, dir, ":sparkles: feat: Add fixtures")

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()
	cfg.SkipDirs = []string{"thirdparty"}

	findings := NewSession(repo, "", nil, cfg).Analyze(ScopeChanged, nil, FileSetAll).Findings

	seen := map[string]bool{}
	for _, finding := range findings {
		if finding.AnalyzerID == "typographic-markers" {
			seen[finding.File] = true
		}
	}
	if !seen["main.go"] {
		t.Errorf("expected main.go to be reported, findings=%+v", findings)
	}
	if seen["thirdparty/gen.go"] {
		t.Error("expected thirdparty/gen.go to be skipped by the configured entry")
	}
	if seen["vendor/lib.go"] {
		t.Error("expected vendor/lib.go to stay skipped by the built-in default")
	}
}

func TestSession_SkipDirs_SharedByLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)

	emDash := "\u2014"
	body := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
	writeFileAtDir(t, dir, "thirdparty/gen.go", body)
	runGitAt(t, dir, "add", "-A")

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	cfg := config.Defaults()
	cfg.ApplyPreset()
	cfg.SkipDirs = []string{"thirdparty"}

	findings := NewSession(repo, "", nil, cfg).Analyze(ScopeChanged, nil, FileSetChanged).Findings
	for _, finding := range findings {
		if finding.AnalyzerID == "typographic-markers" {
			t.Fatalf("expected the staged thirdparty file to stay skipped, got %+v", finding)
		}
	}
}
