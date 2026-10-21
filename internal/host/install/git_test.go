package install

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func TestUninstallGit_GitRemovedHooks(t *testing.T) {
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

			rpt, err := UninstallGit(dir)
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

func TestUninstallGit_GitOrphanedBackup(
	t *testing.T,
) {
	t.Parallel()

	dir := t.TempDir()
	hooksPath := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksPath, 0o750); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(hooksPath, "pre-commit")
	writeForeignReplacingReadOnlyHookViaUnlinkFirst := func(body string) {
		t.Helper()
		_ = os.Remove(hookPath)
		if err := os.WriteFile(hookPath, []byte(body), 0o700); err != nil {
			t.Fatalf("seed foreign hook: %v", err)
		}
	}

	writeForeignReplacingReadOnlyHookViaUnlinkFirst("#!/bin/sh\n# first hand-written hook\n")
	if _, err := githook.Install(dir, []string{"pre-commit"}, githook.Options{}); err != nil {
		t.Fatalf("first githook.Install: %v", err)
	}
	writeForeignReplacingReadOnlyHookViaUnlinkFirst("#!/bin/sh\n# second hand-written hook\n")
	if _, err := githook.Install(dir, []string{"pre-commit"}, githook.Options{}); err != nil {
		t.Fatalf("second githook.Install: %v", err)
	}

	rpt, err := UninstallGit(dir)
	if err != nil {
		t.Fatalf("Uninstall(git): %v", err)
	}
	wantSlot := hookPath + ownedfile.BackupSuffix + ".1"
	gotOrphan := slices.ContainsFunc(rpt.Warnings, func(text string) bool {
		return strings.Contains(text, wantSlot)
	})
	if !gotOrphan {
		t.Fatalf("Warnings = %v, want a note naming %s", rpt.Warnings, wantSlot)
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

			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
				t.Fatalf("mkdir .git: %v", err)
			}

			_, err := Git(dir, testCase.gitHooks, githook.Options{})
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

	dir := t.TempDir()
	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	seedFile(t, preCommit, foreign, 0o755)

	rpt, err := Git(dir, []string{"pre-commit"}, githook.Options{})
	if err != nil {
		t.Fatalf("Install(git): %v", err)
	}
	if len(rpt.Warnings) != 1 || !strings.Contains(rpt.Warnings[0], "backed up existing") {
		t.Fatalf("Warnings = %v, want one backup notice", rpt.Warnings)
	}
	assertContent(t, preCommit+ownedfile.BackupSuffix, foreign)

	uninstallReport, err := UninstallGit(dir)
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

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	preCommit := []string{"pre-commit"}
	if _, err := Git(dir, preCommit, githook.Options{}); err != nil {
		t.Fatalf("first Install(git): %v", err)
	}
	rpt, err := Git(dir, preCommit, githook.Options{})
	if err != nil {
		t.Fatalf("second Install(git): %v", err)
	}
	if len(rpt.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none (the hook was already ours)", rpt.Warnings)
	}
	backupPath := filepath.Join(dir, ".git", "hooks", "pre-commit") + ownedfile.BackupSuffix
	if _, statErr := os.Stat(backupPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected no backup file, stat err = %v", statErr)
	}
}

func TestInstall_Git_UnknownHook(t *testing.T) {
	t.Parallel()

	_, err := Git(t.TempDir(), []string{"bogus"}, githook.Options{})
	want := "install git: install git hooks: unknown git hook " +
		`"bogus" (want ` + strings.Join(githook.HookNames(), "|") + ")"
	if err == nil || err.Error() != want {
		t.Fatalf("Git err = %v, want %q", err, want)
	}
}

func TestUninstallGit_FailureWording(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")
	if err := os.MkdirAll(hookPath, 0o755); err != nil {
		t.Fatalf("seed a directory in place of the hook: %v", err)
	}

	_, err := UninstallGit(dir)
	want := `uninstall git: uninstall git hooks: uninstall hook "commit-msg": check "` +
		hookPath + `": read "` + hookPath + `": read ` + hookPath + `: is a directory`
	if err == nil || err.Error() != want {
		t.Fatalf("UninstallGit err = %v, want %q", err, want)
	}
}
