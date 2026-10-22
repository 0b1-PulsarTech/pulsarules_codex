package evals

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
)

type Command struct {
	artifact string
	scenario string
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.StringVar(&cmd.artifact, "artifact", "", "path to the produced `FILE` to grade")
	fs.StringVar(&cmd.scenario, "scenario", "", "grade only this scenario `ID`")
}

func (cmd *Command) Run(_ remy.Injector, _ []string) error {
	if cmd.artifact == "" {
		return errors.New("evals: --artifact is required (the code or transcript to grade)")
	}
	artifact, err := os.ReadFile(cmd.artifact) //nolint:gosec // The operator's own argument.
	if err != nil {
		return fmt.Errorf("evals: read artifact: %w", err)
	}

	scenarios, err := evals.Load()
	if err != nil {
		return fmt.Errorf("evals: load scenarios: %w", err)
	}
	selected := selectScenarios(scenarios, cmd.scenario)
	if len(selected) == 0 {
		return fmt.Errorf("evals: no scenario matches %q", cmd.scenario)
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
		return command.ErrFindings
	}
	return nil
}

func (cmd *Command) Terms() []command.Term {
	return []command.Term{command.Req("artifact"), command.Opt("scenario")}
}
