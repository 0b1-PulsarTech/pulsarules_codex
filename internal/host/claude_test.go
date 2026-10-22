package host

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeTargetName(t *testing.T) {
	t.Parallel()
	if got := (claudeTarget{}).Name(); got != "claude" {
		t.Fatalf("Name() = %q, want claude", got)
	}
}

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

func TestClaudeUninstall_BothSettings(t *testing.T) {
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
		Base:          base,
		SettingsFiles: []string{"settings.json", "settings.local.json"},
	}
	if _, err := (claudeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(localSettings); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", localSettings, statErr)
	}
}

func TestClaudeUninstall_LocalScopeNoSettings(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	ctx.SettingsFile = "settings.local.json"
	if _, err := (claudeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	settingsJSON := filepath.Join(base, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsJSON); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf(
			"settings.json must never exist for a local-scope install, stat err = %v", statErr,
		)
	}

	uctx := UninstallContext{
		Base:          base,
		SettingsFiles: []string{"settings.json", "settings.local.json"},
	}
	report, err := (claudeTarget{}).Uninstall(uctx)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	for _, note := range report.Notes {
		if strings.Contains(note, settingsJSON) {
			t.Errorf(
				"note %q references settings.json, which never existed", note,
			)
		}
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

func TestClaudeTargetUninstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		keepSkills bool
		checkSkill func(testing.TB, string)
		checkDir   func(testing.TB, string)
	}{
		{
			name:       "removes skills and hook wiring by default",
			checkSkill: assertMissing,
			checkDir:   assertMissing,
		},
		{
			name:       "keep-skills leaves the doc but still unwires hooks",
			keepSkills: true,
			checkSkill: assertPresent,
			checkDir:   assertPresent,
		},
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
			requirePresent(t, skillDoc)

			uctx := UninstallContext{
				Base:          base,
				SettingsFiles: []string{"settings.json"},
				KeepSkills:    testCase.keepSkills,
			}
			report, err := (claudeTarget{}).Uninstall(uctx)
			if err != nil {
				t.Fatalf("Uninstall: %v", err)
			}

			testCase.checkSkill(t, skillDoc)
			if len(report.Notes) == 0 {
				t.Error("expected at least one note describing what was removed/unwired")
			}
			assertMissing(t, filepath.Join(base, ".claude", "settings.json"))
			testCase.checkDir(t, filepath.Join(base, ".claude"))

			if _, err = (claudeTarget{}).Uninstall(uctx); err != nil {
				t.Fatalf("second Uninstall: %v", err)
			}
		})
	}
}

func TestClaudeUninstall_UserFile(t *testing.T) {
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
		Base:          base,
		SettingsFiles: []string{"settings.json"},
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

func TestClaudeUninstall_NoteOnRemoval(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		wireHook bool
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
				Base:          base,
				SettingsFiles: []string{"settings.json"},
				KeepSkills:    true,
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

func TestClaudeInstall_ProjectAtTheUserDirIsAGlobalInstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		link       bool
		elsewhere  bool
		wantGlobal bool
	}{
		{name: "the same path", wantGlobal: true},
		{name: "reached through a symlink", link: true, wantGlobal: true},
		{name: "a different directory stays a project install", elsewhere: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			base := home
			if testCase.link {
				base = filepath.Join(t.TempDir(), "home")
				if err := os.Symlink(home, base); err != nil {
					t.Skipf("symlinks are unavailable: %v", err)
				}
			}
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.NoHooks = false
			ctx.UserClaudeDir = filepath.Join(home, ".claude")
			if testCase.elsewhere {
				ctx.UserClaudeDir = filepath.Join(t.TempDir(), ".claude")
			}
			report, err := (claudeTarget{}).Install(ctx)
			if err != nil {
				t.Fatalf("Install: %v", err)
			}

			claudeDir := filepath.Join(base, ".claude")
			want := hookStyle{testCase.wantGlobal, testCase.wantGlobal, testCase.wantGlobal}
			if got := observeHookStyle(t, claudeDir, report); got != want {
				t.Errorf("hook style = %+v, want %+v", got, want)
			}

			uctx := UninstallContext{Base: base, SettingsFiles: []string{"settings.json"}}
			if _, err = (claudeTarget{}).Uninstall(uctx); err != nil {
				t.Fatalf("Uninstall: %v", err)
			}
			if _, statErr := os.Stat(
				filepath.Join(claudeDir, "hooks"),
			); !errors.Is(
				statErr,
				fs.ErrNotExist,
			) {
				t.Errorf("uninstall left the hooks dir, stat err = %v", statErr)
			}
		})
	}
}

func readFile(tb testing.TB, path string) string {
	tb.Helper()
	body, err := os.ReadFile(path) //nolint:gosec // test fixture path.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(body)
}

func TestClaudeInstall_ShadowCheckNeverAbortsTheInstall(t *testing.T) {
	t.Parallel()

	base, userDir := t.TempDir(), t.TempDir()
	seedFile(t, filepath.Join(userDir, "skills"), "a file where the skills dir belongs")
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	ctx.UserClaudeDir = userDir

	report, err := (claudeTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(base, ".claude", "settings.json")); statErr != nil {
		t.Errorf("the hooks were not wired after the shadow check: %v", statErr)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "could not check") {
		t.Errorf("Warnings = %q, want one could-not-check line", report.Warnings)
	}
}

type hookStyle struct {
	settingsGlobal bool
	binAbsolute    bool
	noted          bool
}

func observeHookStyle(tb testing.TB, claudeDir string, report Report) hookStyle {
	tb.Helper()
	settings := readFile(tb, filepath.Join(claudeDir, "settings.json"))
	script := readFile(tb, filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh"))
	wantNote := "is the user Claude directory; the hook was wired as a global install"
	return hookStyle{
		settingsGlobal: !strings.Contains(settings, `"$CLAUDE_PROJECT_DIR/.claude/hooks`),
		binAbsolute:    strings.Contains(script, "bin='"+filepath.ToSlash(claudeDir)+"/bin/"),
		noted:          strings.Contains(strings.Join(report.Notes, "\n"), wantNote),
	}
}
