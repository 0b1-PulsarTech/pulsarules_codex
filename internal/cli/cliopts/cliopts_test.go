package cliopts

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

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
		{
			name:  "governance suppresses generated files by default",
			args:  []string{"governance", "--project", "/tmp/repo"},
			check: checkGovernanceDefaults,
		},
		{
			name:  "governance with --include-generated",
			args:  []string{"governance", "--project", "/tmp/repo", "--include-generated"},
			check: checkGovernanceIncludeGenerated,
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

func checkGovernanceDefaults(t *testing.T, opts *Options) {
	t.Helper()
	if opts.IncludeGenerated {
		t.Error("IncludeGenerated = true, want false without the flag")
	}
}

func checkGovernanceIncludeGenerated(t *testing.T, opts *Options) {
	t.Helper()
	if !opts.IncludeGenerated {
		t.Error("IncludeGenerated = false, want true with --include-generated")
	}
}

func TestBaseDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		opts    Options
		want    string
		wantErr bool
	}{
		{name: "project", opts: Options{Project: "/repo"}, want: "/repo"},
		{name: "both exclusive", opts: Options{Global: true, Project: "/repo"}, wantErr: true},
		{name: "neither", opts: Options{}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base, err := testCase.opts.BaseDir()
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", base)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if base != testCase.want {
				t.Errorf("base = %q, want %q", base, testCase.want)
			}
		})
	}
}

func TestBaseDir_Global(t *testing.T) {
	t.Parallel()

	opts := Options{Global: true}
	base, err := opts.BaseDir()
	if err != nil {
		t.Fatalf("BaseDir: %v", err)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatalf("resolve home: %v", homeErr)
	}
	if base != home {
		t.Errorf("base = %q, want %q", base, home)
	}
}

func TestGitHookNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		gitHooks string
		want     []string
	}{
		{
			name:     "default pair",
			gitHooks: "commit-msg,pre-commit",
			want:     []string{"commit-msg", "pre-commit"},
		},
		{
			name:     "custom list with pre-push",
			gitHooks: "commit-msg,pre-push",
			want:     []string{"commit-msg", "pre-push"},
		},
		{
			name:     "tolerates spaces and blank entries",
			gitHooks: " commit-msg , , pre-push ",
			want:     []string{"commit-msg", "pre-push"},
		},
		{
			name:     "deduplicates preserving order",
			gitHooks: "pre-push,commit-msg,pre-push",
			want:     []string{"pre-push", "commit-msg"},
		},
		{name: "empty string yields no hooks", gitHooks: "", want: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts := Options{GitHooks: testCase.gitHooks}
			if got := opts.GitHookNames(); !slices.Equal(got, testCase.want) {
				t.Errorf("GitHookNames() = %v, want %v", got, testCase.want)
			}
		})
	}
}
