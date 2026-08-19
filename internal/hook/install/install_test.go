package install

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	names := reg.Names()
	if len(names) != 3 {
		t.Fatalf("expected 3 installers, got %d: %v", len(names), names)
	}
	for _, want := range []string{"claude", "git", "opencode"} {
		if !reg.Has(want) {
			t.Errorf("missing installer %q", want)
		}
	}
}

func TestInstall_UnknownName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Install("nonexistent", Context{Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unknown installer")
	}
}

// TestUninstall_UnknownName asserts uninstalling an unregistered name errors.
func TestUninstall_UnknownName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Uninstall("nonexistent", UninstallContext{Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unknown installer")
	}
}

// TestUninstall_Git asserts the "git" installer dispatches through to
// githook.Uninstall and is idempotent against a directory it never touched.
func TestUninstall_Git(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	if _, err := reg.Uninstall("git", UninstallContext{Dir: dir}); err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
}

// TestRegistryUninstall_Git asserts Registry.Uninstall reports which hooks
// the "git" installer actually removed - no note against a directory Install
// never touched, and a "removed git hooks" note once githook.Install wrote
// them - so a caller can tell a real removal from a no-op instead of
// assuming success from a nil error.
func TestRegistryUninstall_Git(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		install    bool
		wantRemove bool
	}{
		{name: "untouched directory reports nothing removed"},
		{name: "installed hooks are reported removed", install: true, wantRemove: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry()
			dir := t.TempDir()
			if testCase.install {
				if _, err := githook.Install(
					dir,
					[]string{"commit-msg", "pre-commit"},
					githook.Options{},
				); err != nil {
					t.Fatalf("githook.Install: %v", err)
				}
			}

			rpt, err := reg.Uninstall("git", UninstallContext{Dir: dir})
			if err != nil {
				t.Fatalf("Uninstall(git): %v", err)
			}
			gotRemoved := slices.ContainsFunc(rpt.Notes, func(s string) bool {
				return strings.Contains(s, "removed git hooks")
			})
			if gotRemoved != testCase.wantRemove {
				t.Errorf("Notes = %v, want removed=%v", rpt.Notes, testCase.wantRemove)
			}
		})
	}
}

// TestRegistryUninstall_Git_WarnsOrphanedBackup asserts the "git" installer
// warns on a leftover numbered backup slot, the same way claude and
// opencode already do. A foreign hook displaced twice strands its older
// copy at ".1" since Restore only ever consumes the base slot; before
// githook.Uninstall returned its own orphaned notes, nothing named it.
func TestRegistryUninstall_Git_WarnsOrphanedBackup(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	hooksPath := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksPath, 0o750); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(hooksPath, "pre-commit")
	writeForeign := func(body string) {
		t.Helper()
		// Install leaves the hook read-only (0o500); unlink first, same as a
		// person hand-editing the file would have to.
		_ = os.Remove(hookPath)
		if err := os.WriteFile(hookPath, []byte(body), 0o700); err != nil {
			t.Fatalf("seed foreign hook: %v", err)
		}
	}

	writeForeign("#!/bin/sh\n# first hand-written hook\n")
	if _, err := githook.Install(dir, []string{"pre-commit"}, githook.Options{}); err != nil {
		t.Fatalf("first githook.Install: %v", err)
	}
	writeForeign("#!/bin/sh\n# second hand-written hook\n")
	if _, err := githook.Install(dir, []string{"pre-commit"}, githook.Options{}); err != nil {
		t.Fatalf("second githook.Install: %v", err)
	}

	rpt, err := reg.Uninstall("git", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	wantSlot := hookPath + marker.BackupSuffix + ".1"
	gotOrphan := slices.ContainsFunc(rpt.Warnings, func(s string) bool {
		return strings.Contains(s, wantSlot)
	})
	if !gotOrphan {
		t.Fatalf("Warnings = %v, want a note naming %s", rpt.Warnings, wantSlot)
	}
}

// TestInstall_Git_GitHooks asserts the "git" installer writes exactly the
// hooks named in ctx.GitHooks - proving --git-hooks reaches githook.Install
// instead of falling back to its hardcoded pair - and that a zero-value (or
// explicit empty) Context installs nothing: the "commit-msg,pre-commit"
// default lives solely on the CLI flag, never resurrected here.
func TestInstall_Git_GitHooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		gitHooks []string
		want     []string
	}{
		{
			name:     "custom list installs only what was asked",
			gitHooks: []string{"pre-push"},
			want:     []string{"pre-push"},
		},
		{
			name: "zero value installs nothing",
		},
		{
			name:     "explicit empty list installs nothing",
			gitHooks: []string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry()
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
				t.Fatalf("mkdir .git: %v", err)
			}

			_, err := reg.Install("git", Context{Dir: dir, GitHooks: testCase.gitHooks})
			if err != nil {
				t.Fatalf("Install(git): %v", err)
			}
			for _, name := range testCase.want {
				if _, statErr := os.Stat(
					filepath.Join(dir, ".git", "hooks", name),
				); statErr != nil {
					t.Errorf("expected hook %q to be written: %v", name, statErr)
				}
			}
			if len(testCase.want) == 0 {
				if _, statErr := os.Stat(
					filepath.Join(dir, ".git", "hooks"),
				); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("expected no .git/hooks dir to be created, stat err = %v", statErr)
				}
			}
		})
	}
}

// TestInstall_Git_BacksUpForeignHook proves the backup-and-replace feature
// end to end through the Registry: a hand-written pre-commit hook survives
// under a ".pulsarules-backup" slot, the Report carries the backup notice,
// and a subsequent Uninstall restores it - completing the reversal.
func TestInstall_Git_BacksUpForeignHook(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir .git/hooks: %v", err)
	}
	preCommit := filepath.Join(hooksDir, "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	if err := os.WriteFile(preCommit, []byte(foreign), 0o755); err != nil {
		t.Fatalf("seed foreign hook: %v", err)
	}

	rpt, err := reg.Install("git", Context{Dir: dir, GitHooks: []string{"pre-commit"}})
	if err != nil {
		t.Fatalf("Install(git): %v", err)
	}
	if len(rpt.Warnings) != 1 || !strings.Contains(rpt.Warnings[0], "backed up existing") {
		t.Fatalf("Warnings = %v, want one backup notice", rpt.Warnings)
	}
	backupPath := preCommit + marker.BackupSuffix
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}

	uninstallReport, err := reg.Uninstall("git", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	if !slices.ContainsFunc(uninstallReport.Notes, func(s string) bool {
		return strings.Contains(s, preCommit)
	}) {
		t.Fatalf("Notes = %v, want one message naming %q", uninstallReport.Notes, preCommit)
	}
	got, readErr = os.ReadFile(preCommit) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read restored hook: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("restored content = %q, want %q", got, foreign)
	}
}

// TestInstall_Git_OwnHookOverwrittenWithoutBackup asserts installing over a
// hook this installer already wrote never produces a backup or a warning -
// backup-and-replace only fires against a foreign file.
func TestInstall_Git_OwnHookOverwrittenWithoutBackup(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	ctx := Context{Dir: dir, GitHooks: []string{"pre-commit"}}
	if _, err := reg.Install("git", ctx); err != nil {
		t.Fatalf("first Install(git): %v", err)
	}
	rpt, err := reg.Install("git", ctx)
	if err != nil {
		t.Fatalf("second Install(git): %v", err)
	}
	if len(rpt.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none (the hook was already ours)", rpt.Warnings)
	}
	backupPath := filepath.Join(dir, ".git", "hooks", "pre-commit") + marker.BackupSuffix
	if _, statErr := os.Stat(backupPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected no backup file, stat err = %v", statErr)
	}
}

// TestInstall_Git_UnknownHook asserts an unrecognized --git-hooks name fails
// the install instead of silently installing nothing for that name.
func TestInstall_Git_UnknownHook(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Install("git", Context{Dir: t.TempDir(), GitHooks: []string{"bogus"}})
	if err == nil {
		t.Fatal("expected error for unknown git hook name")
	}
}

// TestRegistryUninstall_Opencode asserts Registry.Uninstall reports whether
// the "opencode" installer actually removed a plugin file, mirroring the git
// case's idempotency contract.
func TestRegistryUninstall_Opencode(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()

	rpt, err := reg.Uninstall("opencode", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(opencode) on untouched dir: %v", err)
	}
	if len(rpt.Notes) != 0 {
		t.Errorf("Notes = %v, want none for untouched dir", rpt.Notes)
	}
}
