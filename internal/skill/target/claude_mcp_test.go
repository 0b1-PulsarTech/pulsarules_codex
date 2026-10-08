package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestUnwireClaudeMCP_NoNoteWhenNothingChanged(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seedMCP string
	}{
		{name: "absent .mcp.json"},
		{name: "present .mcp.json with no gopls entry", seedMCP: `{"mcpServers": {"other": {}}}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			repoDir := t.TempDir()
			if testCase.seedMCP != "" {
				path := filepath.Join(repoDir, ".mcp.json")
				if err := os.WriteFile(path, []byte(testCase.seedMCP), 0o600); err != nil {
					t.Fatalf("seed .mcp.json: %v", err)
				}
			}

			var report Report
			if err := unwireClaudeMCP(repoDir, &report); err != nil {
				t.Fatalf("unwireClaudeMCP: %v", err)
			}
			if len(report.Notes) != 0 {
				t.Errorf("Notes = %v, want none (nothing was unwired)", report.Notes)
			}
		})
	}
}

func TestUnwireClaudeMCP_NotesWhenGoplsRemoved(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	seed := `{"mcpServers": {"gopls": {"command": "gopls", "args": ["mcp"]}}}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed .mcp.json: %v", err)
	}

	var report Report
	if err := unwireClaudeMCP(repoDir, &report); err != nil {
		t.Fatalf("unwireClaudeMCP: %v", err)
	}
	if !strings.Contains(strings.Join(report.Notes, "\n"), "unwired MCP servers") {
		t.Errorf("Notes = %v, want an 'unwired MCP servers' entry", report.Notes)
	}
}

func TestGenerateGoplsSkill_ForeignBackup(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}

	skillsDir := t.TempDir()
	goplsDir := filepath.Join(skillsDir, "gopls-navigation")
	const mine = "# my own gopls notes, with no ownership marker\n"
	seedFile(t, filepath.Join(goplsDir, "SKILL.md"), mine)

	var report Report
	if err = generateGoplsSkill(templates, skillsDir, "gopls reference", &report); err != nil {
		t.Fatalf("generateGoplsSkill: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("no backup warning reported for a foreign gopls-navigation skill")
	}
	assertNote(t, report.Warnings, "gopls-navigation")
	if !dirHoldsContent(t, goplsDir, mine) {
		t.Error("the user's own SKILL.md content is gone from the directory")
	}
}

func TestGenerateGoplsSkill_FailureAfterBackup(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("the 4096-byte PATH_MAX that forces the failure is linux's")
	}
	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}

	const name = "gopls-navigation"
	slotTail := ".gitignore" + marker.BackupSuffix
	skillsDir := padPath(t, t.TempDir(), 4096-1-len(slotTail)-len(name)-1)
	goplsDir := filepath.Join(skillsDir, name)
	skillPath := filepath.Join(goplsDir, "SKILL.md")
	seedFile(t, skillPath, "# my own gopls notes\n")
	seedFile(t, filepath.Join(goplsDir, ".gitignore"), "mine\n")

	short := strings.NewReplacer(skillsDir, "<dir>")
	var report Report
	err = generateGoplsSkill(templates, skillsDir, "gopls reference", &report)
	if !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Fatalf("generateGoplsSkill = %v, want ENAMETOOLONG on the .gitignore backup slot",
			short.Replace(fmt.Sprint(err)))
	}
	want := marker.BackupMessage(skillPath, skillPath+marker.BackupSuffix)
	if len(report.Warnings) != 1 || report.Warnings[0] != want {
		t.Errorf("Warnings = %q, want only the %s backup",
			short.Replace(strings.Join(report.Warnings, "; ")), filepath.Base(skillPath))
	}
}

func padPath(tb testing.TB, base string, length int) string {
	tb.Helper()
	for remaining := length - len(base); remaining > 0; {
		width := min(remaining-1, 200)
		if remaining-width-1 == 1 {
			width--
		}
		base = filepath.Join(base, strings.Repeat("d", width))
		remaining -= width + 1
	}
	if len(base) != length {
		tb.Fatalf("padded path is %d bytes, want %d", len(base), length)
	}
	return base
}

func dirHoldsContent(tb testing.TB, dir, content string) bool {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		body, readErr := os.ReadFile(filepath.Join(dir, entry.Name())) //nolint:gosec // fixture.
		if readErr == nil && string(body) == content {
			return true
		}
	}
	return false
}

func TestClaudeInstall_GoplsLanguages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		languages langdetect.Set
		wantSkill bool
		wantWarn  bool
	}{
		{name: "dart project gets no gopls skill", languages: langdetect.NewSet(langdetect.Dart)},
		{
			name:      "go project gets the skill when gopls is on PATH",
			languages: langdetect.NewSet(langdetect.Go),
			wantSkill: mcpwire.GoplsOnPath(),
			wantWarn:  !mcpwire.GoplsOnPath(),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.Languages = testCase.languages
			ctx.NoMCP = false
			report, err := (claudeTarget{}).Install(ctx)
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			skillPath := filepath.Join(base, ".claude", "skills", "gopls-navigation", "SKILL.md")
			if _, statErr := os.Stat(skillPath); (statErr == nil) != testCase.wantSkill {
				t.Errorf("gopls skill present = %v, want %v", statErr == nil, testCase.wantSkill)
			}
			warned := strings.Contains(strings.Join(report.Warnings, "\n"), noGoplsWarning)
			if warned != testCase.wantWarn {
				t.Errorf("no-gopls warning = %v, want %v", warned, testCase.wantWarn)
			}
			mcpPath := filepath.Join(base, ".mcp.json")
			if _, statErr := os.Stat(mcpPath); statErr != nil && testCase.wantSkill {
				t.Errorf("go project with gopls must wire .mcp.json: %v", statErr)
			}
		})
	}
}

func fakeGoplsOnPath(tb testing.TB) {
	tb.Helper()
	dir := tb.TempDir()
	script := "#!/bin/sh\necho fake gopls instructions\n"
	if err := os.WriteFile(filepath.Join(dir, "gopls"), []byte(script), 0o700); err != nil {
		tb.Fatalf("write fake gopls: %v", err)
	}
	tb.Setenv("PATH", dir)
}

//nolint:paralleltest // fakeGoplsOnPath needs t.Setenv, which bars t.Parallel.
func TestGoplsSkill_Branches(t *testing.T) {
	testCases := []struct {
		name          string
		languages     langdetect.Set
		hasGopls      bool
		wantGenerated bool
		wantWarn      bool
	}{
		{name: "no go detected", languages: langdetect.NewSet(langdetect.Dart), hasGopls: true},
		{
			name:          "go with gopls on PATH",
			languages:     langdetect.NewSet(langdetect.Go),
			hasGopls:      true,
			wantGenerated: true,
		},
		{
			name:      "go without gopls on PATH",
			languages: langdetect.NewSet(langdetect.Go),
			wantWarn:  true,
		},
	}
	for _, testCase := range testCases {
		//nolint:paralleltest // shares the process PATH.
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.hasGopls {
				fakeGoplsOnPath(t)
			} else {
				t.Setenv("PATH", "")
			}
			ctx := newTestContext(t, t.TempDir(), []string{"go-style"})
			ctx.Languages = testCase.languages

			var report Report
			generated, err := goplsSkill(ctx, t.TempDir(), &report)
			if err != nil {
				t.Fatalf("goplsSkill: %v", err)
			}
			if generated != testCase.wantGenerated {
				t.Errorf("generated = %v, want %v", generated, testCase.wantGenerated)
			}
			warned := strings.Contains(strings.Join(report.Warnings, "\n"), noGoplsWarning)
			if warned != testCase.wantWarn {
				t.Errorf("warned = %v, want %v (%v)", warned, testCase.wantWarn, report.Warnings)
			}
		})
	}
}

//nolint:paralleltest // fakeGoplsOnPath needs t.Setenv, which bars t.Parallel.
func TestWireClaudeMCP_NotesWiredServers(t *testing.T) {
	fakeGoplsOnPath(t)
	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.Languages = langdetect.NewSet(langdetect.Go)

	var report Report
	if err := wireClaudeMCP(ctx, filepath.Join(base, ".claude", "skills"), &report); err != nil {
		t.Fatalf("wireClaudeMCP: %v", err)
	}
	want := "wired MCP servers (gopls) into " + filepath.Join(base, ".mcp.json")
	if len(report.Notes) != 1 || report.Notes[0] != want {
		t.Errorf("Notes = %q, want [%q]", report.Notes, want)
	}
}
