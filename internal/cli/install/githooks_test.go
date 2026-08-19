package install

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// TestValidateGitHooks asserts every recognized --git-hooks name passes and
// an unknown name fails loudly, naming the valid set, instead of silently
// installing nothing for it.
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

// TestValidateTypographicSeverity asserts a bad value is rejected at install,
// before it is baked into a script that would then fail every commit from a
// place the person committing cannot see.
func TestValidateTypographicSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
	}{
		{name: "empty keeps the analyzer default"},
		{name: "error", severity: "error"},
		{name: "warning", severity: "warning"},
		{name: "info", severity: "info"},
		{name: "a typo is rejected", severity: "fatal", wantErr: true},
		{name: "capitalised is rejected", severity: "Warning", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := core.ValidateSeverityName(testCase.severity)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ValidateSeverityName(%q) err = %v, wantErr %v",
					testCase.severity, err, testCase.wantErr)
			}
		})
	}
}
