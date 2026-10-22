package validate

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/evals"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/validate"
)

type Command struct{}

func (Command) Bind(*flag.FlagSet) {}

func (Command) Run(svc bootstrap.Services, _ []string) error {
	idx := svc.Index
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
