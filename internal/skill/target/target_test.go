package target

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func newTestContext(tb testing.TB, base string, ids []string) Context {
	tb.Helper()
	idx, templates, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("load knowledge: %v", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		tb.Fatalf("new renderer: %v", err)
	}
	return Context{
		Templates:      templates,
		Index:          idx,
		Renderer:       rnd,
		HookInstallers: install.NewRegistry(),
		Base:           base,
		IDs:            ids,
		NoMCP:          true,
		NoHooks:        true,
		SettingsFile:   "settings.json",
	}
}

func TestReport_NoteAndWarn(t *testing.T) {
	t.Parallel()

	var report Report
	report.Note("installed: %s", "x")
	report.Warn("skipped: %s", "y")

	if len(report.Notes) != 1 || report.Notes[0] != "installed: x" {
		t.Fatalf("Notes = %v, want [installed: x]", report.Notes)
	}
	if len(report.Warnings) != 1 || report.Warnings[0] != "skipped: y" {
		t.Fatalf("Warnings = %v, want [skipped: y]", report.Warnings)
	}
}

func seedFile(tb testing.TB, filePath, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", filePath, err)
	}
}

func assertPresent(tb testing.TB, filePath string) {
	tb.Helper()
	if _, err := os.Stat(filePath); err != nil {
		tb.Errorf("expected %q to exist, stat err = %v", filePath, err)
	}
}

func requirePresent(tb testing.TB, filePath string) {
	tb.Helper()
	if _, err := os.Stat(filePath); err != nil {
		tb.Fatalf("expected %q to exist, stat err = %v", filePath, err)
	}
}

func assertMissing(tb testing.TB, filePath string) {
	tb.Helper()
	if _, err := os.Stat(filePath); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("expected %q to be absent, stat err = %v", filePath, err)
	}
}

func assertNote(tb testing.TB, entries []string, substr string) {
	tb.Helper()
	if !strings.Contains(strings.Join(entries, "\n"), substr) {
		tb.Errorf("entries = %v, want one containing %q", entries, substr)
	}
}
