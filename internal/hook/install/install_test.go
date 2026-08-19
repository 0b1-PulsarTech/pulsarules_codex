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

func TestRegistryUninstall_GitRemovedHooks(t *testing.T) {
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
			gotRemoved := slices.ContainsFunc(rpt.Notes, func(text string) bool {
				return strings.Contains(text, "removed git hooks")
			})
			if gotRemoved != testCase.wantRemove {
				t.Errorf("Notes = %v, want removed=%v", rpt.Notes, testCase.wantRemove)
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

func TestInstall_Git_ForeignHookBackup(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	dir := t.TempDir()
	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	seedFile(t, preCommit, foreign, 0o755)

	rpt, err := reg.Install("git", Context{Dir: dir, GitHooks: []string{"pre-commit"}})
	if err != nil {
		t.Fatalf("Install(git): %v", err)
	}
	if len(rpt.Warnings) != 1 || !strings.Contains(rpt.Warnings[0], "backed up existing") {
		t.Fatalf("Warnings = %v, want one backup notice", rpt.Warnings)
	}
	assertContent(t, preCommit+marker.BackupSuffix, foreign)

	uninstallReport, err := reg.Uninstall("git", UninstallContext{Dir: dir})
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	if !slices.ContainsFunc(uninstallReport.Notes, func(text string) bool {
		return strings.Contains(text, preCommit)
	}) {
		t.Fatalf("Notes = %v, want one message naming %q", uninstallReport.Notes, preCommit)
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

func TestInstall_Git_UnknownHook(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, err := reg.Install("git", Context{Dir: t.TempDir(), GitHooks: []string{"bogus"}})
	if err == nil {
		t.Fatal("expected error for unknown git hook name")
	}
}

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
