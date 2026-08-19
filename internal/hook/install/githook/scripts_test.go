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

			script, ok := hookScript(testCase.hook)
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
		if _, ok := hookScript(name); !ok {
			t.Errorf("HookNames reports %q but no script renders for it", name)
		}
		if hookSpecs[name].description == "" {
			t.Errorf("hook %q has no description", name)
		}
	}
}
