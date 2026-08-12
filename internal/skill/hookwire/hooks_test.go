package hookwire

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestInstallHook(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	info, err := os.Stat(script)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("script not executable: mode %v", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(claudeDir, "hooks", "README.md")); err != nil {
		t.Errorf("README not installed: %v", err)
	}
}

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

	backedUp2, err := InstallHook(fakeTemplates(), claudeDir)
	if err != nil {
		t.Fatalf("second InstallHook: %v", err)
	}
	if len(backedUp2) != 0 {
		t.Errorf("second InstallHook backedUp = %v, want none (it owns the file now)", backedUp2)
	}
}

func TestInstallHook_NextNumberedSlot(t *testing.T) {
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
