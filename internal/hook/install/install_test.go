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

func TestUninstall_UnknownNameErrors(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Uninstall("nonexistent", UninstallContext{Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unknown installer")
	}
}

func TestUninstall_GitDispatch(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	if _, err := reg.Uninstall("git", UninstallContext{Dir: dir}); err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
}

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

func TestInstall_Git_ContextHooks(
	t *testing.T,
) {
	t.Parallel()

	testCases := []struct {
		name     string
		gitHooks []string
		want     []string
	}{
		{
			name:     "custom list installs only what was asked",
			gitHooks: []string{"commit-msg"},
			want:     []string{"commit-msg"},
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

func TestInstall_Git_BacksUpForeignHook(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	seedFile(t, preCommit, foreign, 0o755)

	var warnings []string
	err := reg.Install("git", Context{
		Dir:      dir,
		GitHooks: []string{"pre-commit"},
		Warn: func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		t.Fatalf("Install(git): %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "backed up existing") {
		t.Fatalf("warnings = %v, want one backup notice", warnings)
	}
	assertContent(t, preCommit+marker.BackupSuffix, foreign)

	result, err := reg.Uninstall("git", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	if len(result.Restored) != 1 || !strings.Contains(result.Restored[0], preCommit) {
		t.Fatalf("Restored = %v, want one message naming %q", result.Restored, preCommit)
	}
	assertContent(t, preCommit, foreign)
}

func seedFile(tb testing.TB, path, content string, mode os.FileMode) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		tb.Fatalf("seed %q: %v", path, err)
	}
}

func assertContent(tb testing.TB, path, want string) {
	tb.Helper()

	got, err := os.ReadFile(path) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		tb.Errorf("%q content = %q, want %q", path, got, want)
	}
}

func TestInstall_Git_OwnHookOverwrite(
	t *testing.T,
) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	var warnings []string
	warn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
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

func TestInstall_Git_UnknownHook(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	err := reg.Install("git", Context{Dir: t.TempDir(), GitHooks: []string{"bogus"}})
	if err == nil {
		t.Fatal("expected error for unknown git hook name")
	}
}

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
