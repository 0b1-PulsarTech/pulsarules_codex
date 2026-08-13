package hookwire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestInstallHook copies the script (executable) and README into <claudeDir>/hooks.
func TestInstallHook(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	stat, err := os.Stat(script)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if stat.Mode().Perm()&0o100 == 0 {
		t.Errorf("script not executable: mode %v", stat.Mode().Perm())
	}
	if _, err = os.Stat(filepath.Join(claudeDir, "hooks", "README.md")); err != nil {
		t.Errorf("README not installed: %v", err)
	}
}

// TestInstallHook_BacksUpForeignReadme is the regression test for the
// data-loss defect: a user-authored README.md (no marker.Installed) is
// renamed to a ".pulsarules-backup" slot rather than destroyed, and
// InstallHook reports the rename through backedUp; a second call against
// its own previously installed README overwrites in place, no backup.
func TestInstallHook_BacksUpForeignReadme(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readme := filepath.Join(hooksDir, "README.md")
	foreign := "# My own notes about this hooks dir\n"
	if err := os.WriteFile(readme, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign readme: %v", err)
	}

	backedUp, err := InstallHook(fakeTemplates(), claudeDir)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	backupPath := readme + marker.BackupSuffix
	wantMsg := marker.BackupMessage(readme, backupPath)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}

	// A second call finds its own README (marker.Installed) and overwrites
	// it in place, with no further backup.
	secondBackedUp, err := InstallHook(fakeTemplates(), claudeDir)
	if err != nil {
		t.Fatalf("second InstallHook: %v", err)
	}
	if len(secondBackedUp) != 0 {
		t.Errorf(
			"second InstallHook backedUp = %v, want none (it owns the file now)",
			secondBackedUp,
		)
	}
}

// TestInstallHook_NeverOverwritesExistingBackup asserts a foreign README
// backed up when a ".pulsarules-backup" slot is already occupied (by an
// earlier, unresolved backup) falls back to the next free numbered slot
// instead of clobbering it.
func TestInstallHook_NeverOverwritesExistingBackup(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readme := filepath.Join(hooksDir, "README.md")
	current := "# current notes\n"
	if err := os.WriteFile(readme, []byte(current), 0o600); err != nil {
		t.Fatalf("seed foreign readme: %v", err)
	}
	priorBackup := readme + marker.BackupSuffix
	priorContent := "# an even earlier version of the user's notes\n"
	if err := os.WriteFile(priorBackup, []byte(priorContent), 0o600); err != nil {
		t.Fatalf("seed prior backup: %v", err)
	}

	backedUp, err := InstallHook(fakeTemplates(), claudeDir)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	newBackup := readme + marker.BackupSuffix + ".1"
	wantMsg := marker.BackupMessage(readme, newBackup)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}
	got, readErr := os.ReadFile(priorBackup) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read prior backup: %v", readErr)
	}
	if string(got) != priorContent {
		t.Errorf("prior backup content = %q, want %q (clobbered)", got, priorContent)
	}
	got, readErr = os.ReadFile(newBackup) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read new backup: %v", readErr)
	}
	if string(got) != current {
		t.Errorf("new backup content = %q, want %q", got, current)
	}
}

// TestInstallHook_RendersRealTemplate is the byte-identical regression test:
// InstallHook against the actual embedded templates (not a fake) must still
// write a script whose bin/skills/log paths are the concrete
// ".claude/bin/pulsarules_cli", ".claude/skills", and
// ".claude/hook-execution.log", proving the render leaves no "{{" behind.
func TestInstallHook_RendersRealTemplate(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err = InstallHook(templates, claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	got, err := os.ReadFile(
		filepath.Join(claudeDir, "hooks", reminderScriptAsset),
	) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read rendered script: %v", err)
	}
	script := string(got)
	if strings.Contains(script, "{{") {
		t.Errorf("rendered script still carries a template placeholder:\n%s", script)
	}
	for _, want := range []string{
		`bin="$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"`,
		`export PULSARULES_SKILLS_DIR="$CLAUDE_PROJECT_DIR/.claude/skills"`,
		`export PULSARULES_LOG_PATH="$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("rendered script missing %q:\n%s", want, script)
		}
	}
}

// TestRenderReminderScript covers the render helper directly: the success
// substitution and the template-parse failure path.
func TestRenderReminderScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{
			name: "substitutes every path",
			body: `bin="{{.BinaryRelPath}}" skills="{{.SkillsRelPath}}" log="{{.LogRelPath}}"`,
			want: `bin=".claude/bin/pulsarules_cli" skills=".claude/skills" log=".claude/hook-execution.log"`,
		},
		{
			name:    "malformed template fails to parse",
			body:    `{{.BinaryRelPath`,
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, renderErr := renderReminderScript("t", []byte(testCase.body))
			if testCase.wantErr {
				if renderErr == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if renderErr != nil {
				t.Fatalf("renderReminderScript: %v", renderErr)
			}
			if string(got) != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
