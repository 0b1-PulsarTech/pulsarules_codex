package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
)

type artifactMode int

const (
	artifactUnset artifactMode = iota
	artifactAbsent
	artifactWritten
)

// TestRunEvals_ReachesGradeThroughTheDispatcher is the wiring test, not the algorithm test:
// evals.Grade was fully unit-tested while having no caller outside its own package, so the
// documented operator procedure ("call evals.Grade") named something no operator could invoke.
// This asserts the command exists, is dispatched, and grades a real artifact.
func TestRunEvals_ReachesGradeThroughTheDispatcher(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		mode      artifactMode
		artifact  string
		scenario  string
		wantErrIs string
		wantExit  bool
	}{
		{
			name:      "missing artifact flag",
			mode:      artifactUnset,
			wantErrIs: "--artifact is required",
		},
		{
			name:      "artifact file does not exist",
			mode:      artifactAbsent,
			wantErrIs: "read artifact",
		},
		{
			name:      "scenario id matches nothing",
			mode:      artifactWritten,
			artifact:  "anything",
			scenario:  "no-such-scenario",
			wantErrIs: "no scenario matches",
		},
		{
			name:     "a failing machine assertion exits non-zero",
			mode:     artifactWritten,
			artifact: "package main\n",
			scenario: scenarioFailingOn(t, "package main\n"),
			wantExit: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			opts := &cliopts.Options{Command: "evals", Scenario: testCase.scenario}
			switch testCase.mode {
			case artifactUnset:
			case artifactAbsent:
				opts.Artifact = filepath.Join(t.TempDir(), "absent.txt")
			case artifactWritten:
				opts.Artifact = filepath.Join(t.TempDir(), "artifact.txt")
				if err := os.WriteFile(
					opts.Artifact,
					[]byte(testCase.artifact),
					0o600,
				); err != nil {
					t.Fatalf("write artifact: %v", err)
				}
			}

			err := Run(remy.NewInjector(remy.Config{DuckTypeElements: true}), opts)
			if testCase.wantErrIs != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErrIs) {
					t.Fatalf("error = %v, want it to mention %q", err, testCase.wantErrIs)
				}
				return
			}
			var exitErr *ExitError
			if testCase.wantExit && !errors.As(err, &exitErr) {
				t.Fatalf("error = %v, want an *ExitError for a failed assertion", err)
			}
		})
	}
}

// scenarioFailingOn picks a scenario id by MEASURING rather than hardcoding: the first whose
// machine assertions do not all pass against artifact. A hardcoded id would silently stop testing
// the non-zero-exit path the day that scenario's assertions change.
func scenarioFailingOn(tb testing.TB, artifact string) string {
	tb.Helper()
	scenarios, err := evals.Load()
	if err != nil {
		tb.Fatalf("evals.Load: %v", err)
	}
	for _, scenario := range scenarios {
		passed, total := evals.Grade(scenario, artifact).MachineTally()
		if total > 0 && passed < total {
			return scenario.ID
		}
	}
	tb.Fatalf("no embedded scenario fails on %q; cannot test the non-zero-exit path", artifact)
	return ""
}
