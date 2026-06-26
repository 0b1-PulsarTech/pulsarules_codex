package cliopts

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestParseArgs covers the subcommands, the install flags (including the new
// --no-hooks / --print-hooks), and an unknown command.
func TestParseArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(*testing.T, *Options)
	}{
		{
			name:  "install project with hooks flags",
			args:  []string{"install", "--project", "/tmp/repo", "--no-hooks", "--print-hooks"},
			check: checkInstallProjectWithHooks,
		},
		{
			name: "install with git-hooks flags",
			args: []string{
				"install",
				"--project",
				"/tmp/repo",
				"--git-hooks",
				"commit-msg,pre-push",
			},
			check: checkInstallGitHooks,
		},
		{
			name:  "install with --no-git-hooks",
			args:  []string{"install", "--project", "/tmp/repo", "--no-git-hooks"},
			check: checkInstallNoGitHooks,
		},
		{
			name:  "install default git-hooks",
			args:  []string{"install", "--project", "/tmp/repo"},
			check: checkInstallDefaultGitHooks,
		},
		{
			name:  "generate default out",
			args:  []string{"generate"},
			check: checkGenerateDefaultOut,
		},
		{
			name:  "list defaults to skills",
			args:  []string{"list"},
			check: checkListDefaultsToSkills,
		},
		{name: "unknown command", args: []string{"frobnicate"}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts, err := ParseArgs(testCase.args)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got opts %+v", opts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.check != nil {
				testCase.check(t, opts)
			}
		})
	}
}

func checkInstallProjectWithHooks(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Command != "install" || opts.Project != "/tmp/repo" {
		t.Errorf("unexpected opts: %+v", opts)
	}
	if !opts.NoHooks || !opts.PrintHooks {
		t.Errorf("hook flags not parsed: %+v", opts)
	}
}

func checkInstallGitHooks(t *testing.T, opts *Options) {
	t.Helper()
	if opts.GitHooks != "commit-msg,pre-push" {
		t.Errorf("GitHooks = %q, want commit-msg,pre-push", opts.GitHooks)
	}
	if opts.NoGitHooks {
		t.Error("NoGitHooks should be false")
	}
}

func checkInstallNoGitHooks(t *testing.T, opts *Options) {
	t.Helper()
	if !opts.NoGitHooks {
		t.Error("NoGitHooks should be true")
	}
}

func checkInstallDefaultGitHooks(t *testing.T, opts *Options) {
	t.Helper()
	if opts.GitHooks != "commit-msg,pre-commit" {
		t.Errorf("default GitHooks = %q, want commit-msg,pre-commit", opts.GitHooks)
	}
}

func checkGenerateDefaultOut(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Out != filepath.Join(".", "generated") {
		t.Errorf("Out = %q, want ./generated", opts.Out)
	}
}

func checkListDefaultsToSkills(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Kind != "skills" {
		t.Errorf("Kind = %q, want skills", opts.Kind)
	}
}

// TestInstallDest covers global, project, and the two invalid combinations.
func TestInstallDest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		opts     Options
		wantErr  bool
		wantTail string
	}{
		{"project", Options{Project: "/repo"}, false, filepath.Join("/repo", ".claude", "skills")},
		{"both exclusive", Options{Global: true, Project: "/repo"}, true, ""},
		{"neither", Options{}, true, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dest, err := testCase.opts.InstallDest()
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", dest)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasSuffix(dest, testCase.wantTail) {
				t.Errorf("dest = %q, want suffix %q", dest, testCase.wantTail)
			}
		})
	}
}

// TestInstallDest_Global asserts global resolves under the home dir.
func TestInstallDest_Global(t *testing.T) {
	t.Parallel()

	opts := Options{Global: true}
	dest, err := opts.InstallDest()
	if err != nil {
		t.Fatalf("InstallDest: %v", err)
	}
	if !strings.HasSuffix(dest, filepath.Join(".claude", "skills")) {
		t.Errorf("dest = %q, want .claude/skills suffix", dest)
	}
}
