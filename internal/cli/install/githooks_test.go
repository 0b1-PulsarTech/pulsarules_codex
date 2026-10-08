package install

import (
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
