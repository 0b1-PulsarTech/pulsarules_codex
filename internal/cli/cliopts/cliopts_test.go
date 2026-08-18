package cliopts

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// parseArgsCase is one ParseArgs input/assertion pair, shared by TestParseArgs
// and TestParseArgs_Uninstall so both drive the same run loop.
type parseArgsCase struct {
	name    string
	args    []string
	wantErr bool
	check   func(*testing.T, *Options)
}

// runParseArgsCases parses each case's args and either asserts an error, or
// runs its check against the parsed Options.
func runParseArgsCases(t *testing.T, testCases []parseArgsCase) {
	t.Helper()
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

// TestParseArgs covers the subcommands, the install flags (including the new
// --no-hooks / --print-hooks), and an unknown command.
func TestParseArgs(t *testing.T) {
	t.Parallel()

	runParseArgsCases(t, []parseArgsCase{
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
	})
}

// TestParseArgs_Uninstall covers the uninstall subcommand's flags: --target,
// --keep-skills, and its default.
func TestParseArgs_Uninstall(t *testing.T) {
	t.Parallel()

	runParseArgsCases(t, []parseArgsCase{
		{
			name: "project with target and keep-skills",
			args: []string{
				"uninstall",
				"--project",
				"/tmp/repo",
				"--target",
				"opencode",
				"--keep-skills",
			},
			check: checkUninstallKeepSkills,
		},
		{
			name:  "default keep-skills is false",
			args:  []string{"uninstall", "--project", "/tmp/repo"},
			check: checkUninstallDefaultKeepSkills,
		},
		{
			name:  "default hooks-scope is empty (narrows to nothing yet)",
			args:  []string{"uninstall", "--project", "/tmp/repo"},
			check: checkUninstallDefaultHooksScope,
		},
	})
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

func checkUninstallKeepSkills(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Command != "uninstall" || opts.Project != "/tmp/repo" {
		t.Errorf("unexpected opts: %+v", opts)
	}
	if !slices.Contains(opts.Target, "opencode") {
		t.Errorf("Target = %v, want to contain opencode", opts.Target)
	}
	if !opts.KeepSkills {
		t.Error("KeepSkills should be true")
	}
}

func checkUninstallDefaultKeepSkills(t *testing.T, opts *Options) {
	t.Helper()
	if opts.KeepSkills {
		t.Error("KeepSkills should default to false")
	}
}

func checkUninstallDefaultHooksScope(t *testing.T, opts *Options) {
	t.Helper()
	if opts.HooksScope != "" {
		t.Errorf(
			"HooksScope = %q, want empty (uninstall defaults to unwiring both files)",
			opts.HooksScope,
		)
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

// TestBaseDir covers project, and the two invalid combinations; BaseDir stays
// host-neutral (a bare project/home path), leaving the ".claude" layout to
// each target.Strategy that resolves a destination underneath it.
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

// TestBaseDir_Global asserts global resolves to the home dir.
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

// TestSettingsFiles covers the default (both scopes, since uninstall cannot
// recover which one install used), each explicit narrowing, and the invalid
// value.
func TestSettingsFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		hooksScope string
		want       []string
		wantErr    bool
	}{
		{name: "default unwires both", want: []string{"settings.json", "settings.local.json"}},
		{
			name:       "project narrows to project",
			hooksScope: "project",
			want:       []string{"settings.json"},
		},
		{
			name:       "local narrows to local",
			hooksScope: "local",
			want:       []string{"settings.local.json"},
		},
		{name: "invalid scope errors", hooksScope: "bogus", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts := Options{HooksScope: testCase.hooksScope}
			got, err := opts.SettingsFiles()
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error for --hooks-scope %q", testCase.hooksScope)
				}
				return
			}
			if err != nil {
				t.Fatalf("SettingsFiles: %v", err)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("SettingsFiles() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestExplicitTargets covers no --target (nil, no default applied), a
// repeated --target deduped in order, and contrasts with Targets' "claude"
// default - uninstall must be able to tell the two apart.
func TestExplicitTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		target []string
		want   []string
	}{
		{name: "none passed", want: nil},
		{name: "single target", target: []string{"opencode"}, want: []string{"opencode"}},
		{
			name:   "duplicate deduped preserving order",
			target: []string{"opencode", "claude", "opencode"},
			want:   []string{"opencode", "claude"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts := Options{Target: testCase.target}
			if got := opts.ExplicitTargets(); !slices.Equal(got, testCase.want) {
				t.Errorf("ExplicitTargets() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestGitHookNames covers the default two-hook value, a custom list
// including pre-push, whitespace tolerance, and blank/duplicate entries
// being dropped - the parse that lets --git-hooks reach the installer.
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
