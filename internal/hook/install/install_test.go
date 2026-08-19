package install

import (
	"errors"
	"fmt"
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
	err := reg.Install("nonexistent", Context{Dir: t.TempDir()})
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

// TestRegistryUninstall_Git asserts Registry.Uninstall reports which hook
// names the "git" installer actually removed - empty against a directory
// Install never touched, and the installed names once githook.Install wrote
// them - so a caller can tell a real removal from a no-op instead of
// assuming success from a nil error.
func TestRegistryUninstall_Git(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		install     bool
		wantRemoved []string
	}{
		{name: "untouched directory reports nothing removed"},
		{
			name:        "installed hooks are reported removed",
			install:     true,
			wantRemoved: []string{"commit-msg", "pre-commit"},
		},
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

			result, err := reg.Uninstall("git", UninstallContext{Dir: dir})
			if err != nil {
				t.Fatalf("Uninstall(git): %v", err)
			}
			removed := slices.Clone(result.Removed)
			slices.Sort(removed)
			want := slices.Clone(testCase.wantRemoved)
			slices.Sort(want)
			if !slices.Equal(removed, want) {
				t.Errorf("removed = %v, want %v", removed, want)
			}
		})
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

			err := reg.Install("git", Context{Dir: dir, GitHooks: testCase.gitHooks})
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
// under a ".pulsarules-backup" slot, ctx.Warn receives the backup notice,
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

	var warnings []string
	err := reg.Install("git", Context{
		Dir:      dir,
		GitHooks: []string{"pre-commit"},
		Warn:     func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) },
	})
	if err != nil {
		t.Fatalf("Install(git): %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "backed up existing") {
		t.Fatalf("warnings = %v, want one backup notice", warnings)
	}
	backupPath := preCommit + marker.BackupSuffix
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}

	result, err := reg.Uninstall("git", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	if len(result.Restored) != 1 || !strings.Contains(result.Restored[0], preCommit) {
		t.Fatalf("Restored = %v, want one message naming %q", result.Restored, preCommit)
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

	var warnings []string
	warn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }
	ctx := Context{Dir: dir, GitHooks: []string{"pre-commit"}, Warn: warn}
	if err := reg.Install("git", ctx); err != nil {
		t.Fatalf("first Install(git): %v", err)
	}
	if err := reg.Install("git", ctx); err != nil {
		t.Fatalf("second Install(git): %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none (the hook was already ours)", warnings)
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
	err := reg.Install("git", Context{Dir: t.TempDir(), GitHooks: []string{"bogus"}})
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

	result, err := reg.Uninstall("opencode", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(opencode) on untouched dir: %v", err)
	}
	if len(result.Removed) != 0 {
		t.Errorf("removed = %v, want none for untouched dir", result.Removed)
	}
}
