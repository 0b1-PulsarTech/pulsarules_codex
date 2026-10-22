package validate

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/validate"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Command struct{}

func (Command) Bind(*flag.FlagSet) {}

func (Command) Run(inj remy.Injector, _ []string) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	result := validate.Validate(
		idx,
		registry.RuleAnalyzersCheck,
		evals.ValidateCheck,
	)
	if !result.OK() {
		printProblems(os.Stderr, result.Errors)
		return errors.New("validation failed")
	}
	fmt.Printf("ok: %d rules, %d patterns, %d workflows, %d skills\n",
		len(idx.Rules), len(idx.Patterns), len(idx.Workflows), len(idx.Skills))
	return nil
}

func printProblems(out io.Writer, problems []string) {
	_, _ = fmt.Fprintln(out, "validation failed:")
	for _, problem := range problems {
		_, _ = fmt.Fprintf(out, "  - %s\n", printable.Line(problem))
	}
}

func (Command) Terms() []command.Term {
	return []command.Term{command.Opt("root")}
}
