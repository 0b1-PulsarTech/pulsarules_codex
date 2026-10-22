package version

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

type parseArgsCase struct {
	name    string
	args    []string
	wantErr bool
	check   func(*testing.T, *Options)
}

func runParseArgsCases(t *testing.T, testCases []parseArgsCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts, err := ParseArgs(testCase.args, flags.Environment{})
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
		{name: "unknown command", args: []string{"frobnicate"}, wantErr: true},
	})
}

func TestParseArgs_Governance(t *testing.T) {
	t.Parallel()

	runParseArgsCases(t, []parseArgsCase{
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
		{
			name:  "governance takes a config path",
			args:  []string{"governance", "--project", "/tmp/repo", "--config", "x/app.yaml"},
			check: checkGovernanceConfigPath,
		},
	})
}

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
	if want := []string{"commit-msg", "pre-push"}; !slices.Equal(opts.GitHooks, want) {
		t.Errorf("GitHooks = %q, want %q", opts.GitHooks, want)
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
	if want := []string{"commit-msg", "pre-commit"}; !slices.Equal(opts.GitHooks, want) {
		t.Errorf("default GitHooks = %q, want %q", opts.GitHooks, want)
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

func checkGovernanceConfigPath(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Config != "x/app.yaml" {
		t.Errorf("Config = %q, want x/app.yaml", opts.Config)
	}
}

func checkGovernanceDefaults(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Preset != "" || opts.Config != "" {
		t.Errorf("Preset = %q, Config = %q, want both unset so the file can supply them",
			opts.Preset, opts.Config)
	}
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
		{name: "relative project is made absolute", opts: Options{Project: "."}, want: workDir(t)},
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

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestBaseDir_GlobalRelativeHomeIsMadeAbsolute(t *testing.T) {
	t.Setenv("HOME", "rel")

	opts := Options{Global: true}
	base, err := opts.BaseDir()
	if err != nil {
		t.Fatalf("BaseDir: %v", err)
	}
	if want := filepath.Join(workDir(t), "rel"); base != want {
		t.Errorf("base = %q, want %q", base, want)
	}
}

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

func workDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return dir
}
