package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
)

// why: the harness deliberately does not invoke a model, so grading is step (3) of an operator
// procedure a human drives (see ARCHITECTURE.md). That procedure said "call evals.Grade" while the
// only way to call it was to write Go - a documented workflow nobody could actually run. This
// command is that call site.
func runEvals(_ remy.Injector, opts *cliopts.Options) error {
	if opts.Artifact == "" {
		return fmt.Errorf(
			"evals: --artifact is required (the produced code or transcript to grade)",
		)
	}
	artifact, err := os.ReadFile(
		opts.Artifact,
	) //nolint:gosec // path is the operator's own argument.
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

// why: the needs-judge assertions are printed in full rather than summarised as a count, because
// step (4) of the procedure is a human reading each one against the artifact - a bare "3 need a
// judge" would send the operator back to the JSON to find out which three.
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
				_, _ = fmt.Fprintf(out, "  needs judge %s: %s\n", assertion.ID, assertion.Text)
			case evals.StatusPass:
			}
		}
	}

	_, _ = fmt.Fprintf(out,
		"\n%d/%d machine assertions passed across %d scenario(s); %d need a human judge\n",
		totalPassed, totalGraded, len(scenarios), totalJudge)
	if totalGraded > 0 && totalPassed < totalGraded {
		return &ExitError{Code: 1}
	}
	return nil
}
