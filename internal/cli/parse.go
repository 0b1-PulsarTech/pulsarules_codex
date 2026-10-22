package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

type Invocation struct {
	Name     string
	root     string
	logLevel string
	env      flags.Environment
	runner   runner
	args     []string

	bootstrapFailed func(out io.Writer, err error) int
}

func Parse(args []string, env flags.Environment, usageOut io.Writer) (Invocation, error) {
	if len(args) == 0 {
		return Invocation{env: env}, nil
	}
	name := args[0]
	if IsHelp(name) || version.IsVersion(name) {
		return Invocation{Name: name, env: env}, nil
	}
	for _, candidate := range table() {
		if candidate.name == name {
			return parseEntry(candidate, args[1:], env, usageOut)
		}
	}
	return Invocation{}, fmt.Errorf("unknown command %q", name)
}

func parseEntry(
	candidate entry,
	flagArgs []string,
	env flags.Environment,
	usageOut io.Writer,
) (Invocation, error) {
	call := Invocation{
		Name:            candidate.name,
		env:             env,
		bootstrapFailed: candidate.bootstrapFailed,
	}
	call.runner = candidate.build(env)
	flagSet := newFlagSet(candidate.name, call.runner, env, &call.root, &call.logLevel)
	var usageErr error
	flagSet.Usage = func() { usageErr = Usage(usageOut) }
	if err := flagSet.Parse(flagArgs); err != nil {
		if usageErr != nil {
			// Usage that cannot be printed is a failure, never the quiet exit of -h.
			return Invocation{}, fmt.Errorf("parse %s flags: %w", candidate.name, usageErr)
		}
		// %w keeps flag.ErrHelp, so main exits 0 after the usage rather than reporting an error.
		return Invocation{}, fmt.Errorf("parse %s flags: %w", candidate.name, err)
	}
	call.args = flagSet.Args()
	return call, nil
}

func newFlagSet(
	name string,
	bound runner,
	env flags.Environment,
	root, logLevel *string,
) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.BindCommon(flagSet, root, logLevel, env)
	bound.command.Bind(flagSet)
	return flagSet
}
