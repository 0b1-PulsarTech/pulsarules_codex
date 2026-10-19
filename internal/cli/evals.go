package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
)

func runEvals(_ remy.Injector, opts *version.Options) error {
	if opts.Artifact == "" {
		return errors.New("evals: --artifact is required (the code or transcript to grade)")
	}
	artifact, err := os.ReadFile(opts.Artifact) //nolint:gosec // The operator's own argument.
	if err != nil {
		return fmt.Errorf("evals: read artifact: %w", err)
	}

	scenarios, err := evals.Load()
	if err != nil {
		return fmt.Errorf("evals: load scenarios: %w", err)
	}
	selected := selectScenarios(scenarios, opts.Scenario)
	if len(selected) == 0 {
		return fmt.Errorf("evals: no scenario matches %q", opts.Scenario)
	}

	// No model is invoked; a human grades the needs-judge assertions.
	return reportEvals(os.Stdout, selected, string(artifact))
}

func selectScenarios(scenarios []evals.Scenario, id string) []evals.Scenario {
	if id == "" {
		return scenarios
	}
	var selected []evals.Scenario
	for _, scenario := range scenarios {
		if scenario.ID == id {
			selected = append(selected, scenario)
		}
	}
	return selected
}

func reportEvals(out io.Writer, scenarios []evals.Scenario, artifact string) error {
	var totalPassed, totalGraded, totalJudge int
	for _, scenario := range scenarios {
		result := evals.Grade(scenario, artifact)
		passed, graded := result.MachineTally()
		totalPassed += passed
		totalGraded += graded

		_, _ = fmt.Fprintf(out, "%s (%s): %d/%d machine assertions passed\n",
			result.ScenarioID, result.Skill, passed, graded)
		for _, assertion := range result.Results {
			switch assertion.Status {
			case evals.StatusFail:
				_, _ = fmt.Fprintf(out, "  FAIL        %s: %s\n", assertion.ID, assertion.Text)
			case evals.StatusNeedsJudge:
				totalJudge++
				// Each is printed in full, not just counted: the operator reads it
				// against the artifact.
				_, _ = fmt.Fprintf(out, "  needs judge %s: %s\n", assertion.ID, assertion.Text)
			}
		}
	}

	_, _ = fmt.Fprintf(out,
		"\n%d/%d machine assertions passed across %d scenario(s); %d need a human judge\n",
		totalPassed, totalGraded, len(scenarios), totalJudge)
	if totalGraded > 0 && totalPassed < totalGraded {
		return ErrFindings
	}
	return nil
}
