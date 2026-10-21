package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func TestValidateGitHooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		hooks   []string
		wantErr bool
	}{
		{name: "empty list is valid", hooks: nil},
		{name: "known hooks are valid", hooks: []string{"commit-msg", "pre-commit", "pre-push"}},
		{name: "unknown hook is rejected", hooks: []string{"bogus"}, wantErr: true},
		{
			name:    "one bad name among good ones is rejected",
			hooks:   []string{"commit-msg", "bogus"},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateGitHooks(testCase.hooks)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error for unknown git hook name")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveGitHooks_NamesTheBadFlag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		opts     *version.Options
		wantFlag string
	}{
		{
			name:     "a severity typo names its flag",
			opts:     &version.Options{TypographicSeverity: "fatal"},
			wantFlag: "--typographic-severity",
		},
		{
			name:     "an extra-types typo names its flag",
			opts:     &version.Options{BranchExtraTypes: "Release"},
			wantFlag: "--branch-extra-types",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := resolveGitHooks(testCase.opts)
			if err == nil || !strings.Contains(err.Error(), testCase.wantFlag) {
				t.Fatalf("resolveGitHooks err = %v, want one naming %q", err, testCase.wantFlag)
			}
		})
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestInstallPostTargets_Branches(t *testing.T) {
	testCases := []struct {
		name         string
		noGitHooks   bool
		gitHooks     []string
		blockGitDir  bool
		wantHookFile bool
		wantStdout   string
		wantStderr   string
	}{
		{
			name:         "hooks and binary installed",
			gitHooks:     []string{"pre-commit"},
			wantHookFile: true,
			wantStdout:   "installed governance binary",
		},
		{
			name:       "no-git-hooks still copies the binary",
			noGitHooks: true,
			gitHooks:   []string{"pre-commit"},
			wantStdout: "installed governance binary",
		},
		{
			name:       "hook install failure is a warning",
			gitHooks:   []string{"bogus"},
			wantStdout: "installed governance binary",
			wantStderr: "warning: git hooks: ",
		},
		{
			name:        "binary copy failure is a warning",
			noGitHooks:  true,
			blockGitDir: true,
			wantStderr:  "warning: binary: ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.blockGitDir {
				seedFile(t, filepath.Join(dir, ".git"), "not a directory")
			}
			opts := &version.Options{NoGitHooks: testCase.noGitHooks}
			cfg := installConfig{absBase: dir, gitHooks: testCase.gitHooks}

			var runErr error
			stdout, stderr := captureStreams(t, func() {
				runErr = installPostTargets(opts, cfg)
			})
			if runErr != nil {
				t.Fatalf("installPostTargets: %v", runErr)
			}
			if !strings.Contains(stdout, testCase.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, testCase.wantStdout)
			}
			if !strings.Contains(stderr, testCase.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, testCase.wantStderr)
			}
			if testCase.wantStderr == "" && stderr != "" {
				t.Errorf("stderr = %q, want none", stderr)
			}
			hookFile := filepath.Join(dir, ".git", "hooks", "pre-commit")
			if _, statErr := os.Stat(hookFile); (statErr == nil) != testCase.wantHookFile {
				t.Errorf("hook file present = %v, want %v", statErr == nil, testCase.wantHookFile)
			}
		})
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestInstallPostTargets_BakesOptions(t *testing.T) {
	testCases := []struct {
		name      string
		hook      string
		opts      version.Options
		wantFlags []string
	}{
		{
			name:      "typographic severity reaches the pre-commit hook",
			hook:      "pre-commit",
			opts:      version.Options{TypographicSeverity: "warning"},
			wantFlags: []string{"--typographic-severity warning"},
		},
		{
			name:      "branch extra types reach the pre-push hook",
			hook:      "pre-push",
			opts:      version.Options{BranchExtraTypes: "spike,experiment"},
			wantFlags: []string{"--branch-extra-types spike,experiment"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := installConfig{absBase: dir, gitHooks: []string{testCase.hook}}

			var runErr error
			captureStreams(t, func() {
				runErr = installPostTargets(&testCase.opts, cfg)
			})
			if runErr != nil {
				t.Fatalf("installPostTargets: %v", runErr)
			}
			script, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", testCase.hook))
			if err != nil {
				t.Fatalf("read hook: %v", err)
			}
			for _, flag := range testCase.wantFlags {
				if !strings.Contains(string(script), flag) {
					t.Errorf("hook script lacks %q:\n%s", flag, script)
				}
			}
		})
	}
}

func seedFile(tb testing.TB, filePath, content string) {
	tb.Helper()
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", filePath, err)
	}
}
