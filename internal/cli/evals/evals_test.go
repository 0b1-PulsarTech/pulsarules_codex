package evals

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
)

type artifactMode int

const (
	artifactUnset artifactMode = iota
	artifactAbsent
	artifactWritten
)

type evalsCase struct {
	name      string
	mode      artifactMode
	artifact  string
	scenario  string
	wantErrIs string
	wantExit  bool
}

func evalsCommand(t *testing.T, testCase evalsCase) *Command {
	t.Helper()

	cmd := &Command{scenario: testCase.scenario}
	switch testCase.mode {
	case artifactUnset:
	case artifactAbsent:
		cmd.artifact = filepath.Join(t.TempDir(), "absent.txt")
	case artifactWritten:
		cmd.artifact = filepath.Join(t.TempDir(), "artifact.txt")
		if err := os.WriteFile(cmd.artifact, []byte(testCase.artifact), 0o600); err != nil {
			t.Fatalf("write artifact: %v", err)
		}
	}
	return cmd
}

func TestRunEvals_Grading(t *testing.T) {
	t.Parallel()

	testCases := []evalsCase{
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

			cmd := evalsCommand(t, testCase)
			err := cmd.Run(remy.NewInjector(remy.Config{DuckTypeElements: true}), nil)
			if testCase.wantErrIs != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErrIs) {
					t.Fatalf("error = %v, want it to mention %q", err, testCase.wantErrIs)
				}
				return
			}
			if testCase.wantExit && !errors.Is(err, command.ErrFindings) {
				t.Fatalf("error = %v, want command.ErrFindings for a failed assertion", err)
			}
		})
	}
}

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

func TestCommand_Bind(t *testing.T) {
	t.Parallel()

	var cmd Command
	fs := flag.NewFlagSet("evals", flag.ContinueOnError)
	cmd.Bind(fs)
	if err := fs.Parse([]string{"--artifact", "a.txt", "--scenario", "s1"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cmd.artifact != "a.txt" || cmd.scenario != "s1" {
		t.Errorf("artifact %q scenario %q, want a.txt and s1", cmd.artifact, cmd.scenario)
	}
}
