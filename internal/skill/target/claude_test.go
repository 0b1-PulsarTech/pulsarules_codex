package target

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

func TestClaudeTargetName(t *testing.T) {
	t.Parallel()
	if got := (claudeTarget{}).Name(); got != "claude" {
		t.Fatalf("Name() = %q, want claude", got)
	}
}

// TestClaudeTargetPresent covers detection of an on-disk .claude dir versus
// an untouched project, the signal uninstall's target auto-detection relies
// on.
func TestClaudeTargetPresent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		makeClaude bool
		want       bool
	}{
		{name: "claude dir present", makeClaude: true, want: true},
		{name: "untouched project", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if testCase.makeClaude {
				if err := os.MkdirAll(filepath.Join(base, ".claude"), 0o750); err != nil {
					t.Fatalf("mkdir .claude: %v", err)
				}
			}
			if got := (claudeTarget{}).Present(base); got != testCase.want {
				t.Errorf("Present(%q) = %v, want %v", base, got, testCase.want)
			}
		})
	}
}

// TestClaudeTargetUninstall_UnwiresBothSettingsFiles asserts every settings
// file in ctx.SettingsFiles gets its hook wiring removed, not just the one
// install happened to use - the fix for uninstall silently leaving live
// wiring behind in whichever file --hooks-scope did not name at install time.
func TestClaudeTargetUninstall_UnwiresBothSettingsFiles(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	ctx.SettingsFile = "settings.local.json"
	if _, err := (claudeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	localSettings := filepath.Join(base, ".claude", "settings.local.json")
	if _, statErr := os.Stat(localSettings); statErr != nil {
		t.Fatalf("Install did not wire %q: %v", localSettings, statErr)
	}

	uctx := UninstallContext{
		Base:             base,
		HookUninstallers: install.NewRegistry(),
		SettingsFiles:    []string{"settings.json", "settings.local.json"},
	}
	if _, err := (claudeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(localSettings); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", localSettings, statErr)
	}
}

func TestClaudeTargetInstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		ids          []string
		wantFiles    []string
		absentFiles  []string
		wantNotes    int
		wantWarnings int
	}{
		{
			name:      "single skill",
			ids:       []string{"go-style"},
			wantFiles: []string{filepath.Join("go-style", "SKILL.md")},
			wantNotes: 1,
		},
		{
			name: "router and skill",
			ids:  []string{"project-router", "go-style"},
			wantFiles: []string{
				filepath.Join("project-router", "SKILL.md"),
				filepath.Join("go-style", "SKILL.md"),
			},
			// project-router composes the feature-development workflow (+1 note).
			wantNotes: 3,
		},
		{
			name:         "unknown skill is skipped, not fatal",
			ids:          []string{"nonexistent-skill"},
			absentFiles:  []string{filepath.Join("nonexistent-skill", "SKILL.md")},
			wantWarnings: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			skills := filepath.Join(base, ".claude", "skills")
			ctx := newTestContext(t, base, testCase.ids)

			report, err := claudeTarget{}.Install(ctx)
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			for _, rel := range testCase.wantFiles {
				if _, statErr := os.Stat(filepath.Join(skills, rel)); statErr != nil {
					t.Errorf("missing rendered skill %q: %v", rel, statErr)
				}
			}
			for _, rel := range testCase.absentFiles {
				if _, statErr := os.Stat(filepath.Join(skills, rel)); statErr == nil {
					t.Errorf("expected %q to be absent", rel)
				}
			}
			if len(report.Notes) != testCase.wantNotes {
				t.Errorf(
					"Notes = %d, want %d (%v)",
					len(report.Notes),
					testCase.wantNotes,
					report.Notes,
				)
			}
			if len(report.Warnings) != testCase.wantWarnings {
				t.Errorf(
					"Warnings = %d, want %d (%v)",
					len(report.Warnings),
					testCase.wantWarnings,
					report.Warnings,
				)
			}
		})
	}
}

// TestClaudeTargetUninstall covers the full round trip (Install then
// Uninstall), --keep-skills leaving the rendered doc but still unwiring
// hooks, and idempotency (a second Uninstall is not an error).
func TestClaudeTargetUninstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		keepSkills bool
	}{
		{name: "removes skills and hook wiring by default"},
		{name: "keep-skills leaves the doc but still unwires hooks", keepSkills: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.NoHooks = false
			if _, err := (claudeTarget{}).Install(ctx); err != nil {
				t.Fatalf("Install: %v", err)
			}
			skillDoc := filepath.Join(base, ".claude", "skills", "go-style", "SKILL.md")
			if _, statErr := os.Stat(skillDoc); statErr != nil {
				t.Fatalf("Install did not write %q: %v", skillDoc, statErr)
			}

			uctx := UninstallContext{
				Base:             base,
				HookUninstallers: install.NewRegistry(),
				SettingsFiles:    []string{"settings.json"},
				KeepSkills:       testCase.keepSkills,
			}
			report, err := (claudeTarget{}).Uninstall(uctx)
			if err != nil {
				t.Fatalf("Uninstall: %v", err)
			}

			_, statErr := os.Stat(skillDoc)
			if testCase.keepSkills {
				if statErr != nil {
					t.Errorf("--keep-skills should have kept %q, stat err = %v", skillDoc, statErr)
				}
			} else if !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("expected %q to be removed, stat err = %v", skillDoc, statErr)
			}
			if len(report.Notes) == 0 {
				t.Error("expected at least one note describing what was removed/unwired")
			}
			settingsPath := filepath.Join(base, ".claude", "settings.json")
			if _, statErr = os.Stat(settingsPath); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf(
					"expected settings.json's hooks-only content to be removed, stat err = %v",
					statErr,
				)
			}
			claudeDir := filepath.Join(base, ".claude")
			_, statErr = os.Stat(claudeDir)
			if testCase.keepSkills {
				if statErr != nil {
					t.Errorf("--keep-skills should have kept %q, stat err = %v", claudeDir, statErr)
				}
			} else if !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("expected %q to be reaped once empty, stat err = %v", claudeDir, statErr)
			}

			// Idempotent: a second Uninstall against the already-reversed
			// project is not an error.
			if _, err = (claudeTarget{}).Uninstall(uctx); err != nil {
				t.Fatalf("second Uninstall: %v", err)
			}
		})
	}
}

// TestClaudeTargetUninstall_UserFileInClaudeDirSurvives asserts a .claude
// directory holding something of the user's outside of skills/workflows/
// hooks/bin is never reaped - fsx.RemoveEmptyDir only ever deletes an
// actually-empty directory.
func TestClaudeTargetUninstall_UserFileInClaudeDirSurvives(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	if _, err := (claudeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	userFile := filepath.Join(base, ".claude", "notes.txt")
	if err := os.WriteFile(userFile, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	uctx := UninstallContext{
		Base:             base,
		HookUninstallers: install.NewRegistry(),
		SettingsFiles:    []string{"settings.json"},
	}
	if _, err := (claudeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, statErr := os.Stat(userFile); statErr != nil {
		t.Errorf("expected %q to survive, stat err = %v", userFile, statErr)
	}
	claudeDir := filepath.Join(base, ".claude")
	if _, statErr := os.Stat(claudeDir); statErr != nil {
		t.Errorf("expected %q to survive (holds a user file), stat err = %v", claudeDir, statErr)
	}
}

// TestClaudeTargetUninstall_NoteGatedOnActualRemoval asserts the "removed
// hook wiring" note is printed only for a settings file Uninstall actually
// changed - the fix for a no-op printing the note unconditionally whenever
// the underlying call returned a nil error, even against a project with no
// settings file at all.
func TestClaudeTargetUninstall_NoteGatedOnActualRemoval(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		wireHook bool // install real hook wiring before uninstalling
		wantNote bool
	}{
		{name: "no settings file: no note"},
		{name: "real wiring: note present", wireHook: true, wantNote: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			if testCase.wireHook {
				ctx := newTestContext(t, base, []string{"go-style"})
				ctx.NoHooks = false
				if _, err := (claudeTarget{}).Install(ctx); err != nil {
					t.Fatalf("Install: %v", err)
				}
			} else if err := os.MkdirAll(filepath.Join(base, ".claude"), 0o750); err != nil {
				t.Fatalf("mkdir .claude: %v", err)
			}

			uctx := UninstallContext{
				Base:             base,
				HookUninstallers: install.NewRegistry(),
				SettingsFiles:    []string{"settings.json"},
				KeepSkills:       true,
			}
			report, err := (claudeTarget{}).Uninstall(uctx)
			if err != nil {
				t.Fatalf("Uninstall: %v", err)
			}

			gotNote := false
			for _, note := range report.Notes {
				if strings.Contains(note, "removed hook wiring") {
					gotNote = true
				}
			}
			if gotNote != testCase.wantNote {
				t.Errorf("removed hook wiring note = %v, want %v (notes: %v)",
					gotNote, testCase.wantNote, report.Notes)
			}
		})
	}
}
