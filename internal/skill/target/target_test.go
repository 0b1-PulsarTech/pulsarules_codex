package target

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// newTestContext builds a Context from the embedded knowledge snapshot with MCP
// and hook wiring gated off, so a Strategy's skill-rendering path is exercised
// deterministically without depending on gopls or copying a binary.
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
	report.note("installed: %s", "x")
	report.warn("skipped: %s", "y")

	if len(report.Notes) != 1 || report.Notes[0] != "installed: x" {
		t.Fatalf("Notes = %v, want [installed: x]", report.Notes)
	}
	if len(report.Warnings) != 1 || report.Warnings[0] != "skipped: y" {
		t.Fatalf("Warnings = %v, want [skipped: y]", report.Warnings)
	}
}
