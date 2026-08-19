package githook

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// TestHookScript asserts every rendered script carries the ownership marker, the
// shebang, and the common-dir resolution a worktree needs, and that an unknown
// hook name renders nothing rather than an empty script.
func TestHookScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		hook        string
		wantOK      bool
		wantCommand string
	}{
		{name: "commit-msg", hook: "commit-msg", wantOK: true, wantCommand: "commitlint"},
		{name: "pre-commit", hook: "pre-commit", wantOK: true, wantCommand: "--scope commit"},
		{name: "pre-push", hook: "pre-push", wantOK: true, wantCommand: "governance"},
		{name: "unknown hook renders nothing", hook: "post-merge"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			script, ok := hookScript(testCase.hook, Options{})
			if ok != testCase.wantOK {
				t.Fatalf("hookScript(%q) ok = %v, want %v", testCase.hook, ok, testCase.wantOK)
			}
			if !testCase.wantOK {
				if script != "" {
					t.Errorf("script = %q, want empty", script)
				}
				return
			}
			for _, want := range []string{
				"#!/bin/sh", marker.Installed, "--git-common-dir", binaryName, testCase.wantCommand,
			} {
				if !strings.Contains(script, want) {
					t.Errorf("script for %q is missing %q", testCase.hook, want)
				}
			}
		})
	}
}

// TestHookNames asserts the reported names match the rendered set, so a hook
// added to one table cannot go missing from install.
func TestHookNames_MatchRenderedScripts(t *testing.T) {
	t.Parallel()

	for _, name := range HookNames() {
		if _, ok := hookScript(name, Options{}); !ok {
			t.Errorf("HookNames reports %q but no script renders for it", name)
		}
		if hookSpecs[name].description == "" {
			t.Errorf("hook %q has no description", name)
		}
	}
}

// TestHookScript_BakesInstallPolicy asserts install-time policy reaches the
// governed hooks and only those. A git hook takes no arguments from the person
// committing, so a flag missing here can never reach the gate.
func TestHookScript_BakesInstallPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		hook     string
		opts     Options
		wantFlag bool
	}{
		{
			name:     "pre-commit carries the chosen severity",
			hook:     "pre-commit",
			opts:     Options{TypographicSeverity: "warning"},
			wantFlag: true,
		},
		{
			name:     "pre-push carries the chosen severity",
			hook:     "pre-push",
			opts:     Options{TypographicSeverity: "warning"},
			wantFlag: true,
		},
		{
			name: "commit-msg runs no governance, so it carries nothing",
			hook: "commit-msg",
			opts: Options{TypographicSeverity: "warning"},
		},
		{name: "an unset policy bakes no flag", hook: "pre-commit"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			script, ok := hookScript(testCase.hook, testCase.opts)
			if !ok {
				t.Fatalf("hookScript(%q) reported no script", testCase.hook)
			}
			got := strings.Contains(script, "--typographic-severity warning")
			if got != testCase.wantFlag {
				t.Errorf(
					"script carries the flag = %v, want %v\n%s",
					got,
					testCase.wantFlag,
					script,
				)
			}
		})
	}
}
