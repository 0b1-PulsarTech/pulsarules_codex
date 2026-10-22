package cli

import (
	"flag"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

type Invocation struct {
	Name     string
	Root     string
	LogLevel string
	Env      flags.Environment
	Runner   command.Runner
	Args     []string
}

func Parse(args []string, env flags.Environment) (Invocation, error) {
	if len(args) == 0 {
		return Invocation{Env: env}, nil
	}
	name := args[0]
	if version.IsHelp(name) || version.IsVersion(name) {
		return Invocation{Name: name, Env: env}, nil
	}
	for _, candidate := range table() {
		if candidate.name == name {
			return parseEntry(candidate, args[1:], env)
		}
	}
	return parseLegacy(args, env)
}

func parseEntry(candidate entry, flagArgs []string, env flags.Environment) (Invocation, error) {
	call := Invocation{Name: candidate.name, Env: env, Runner: candidate.build(env)}
	fs := flag.NewFlagSet(candidate.name, flag.ContinueOnError)
	fs.Usage = func() { _ = version.Usage() }
	flags.BindCommon(fs, &call.Root, &call.LogLevel, env)
	call.Runner.Bind(fs)
	if err := fs.Parse(flagArgs); err != nil {
		// %w keeps flag.ErrHelp, so main exits 0 after fs.Usage rather than reporting an error.
		return Invocation{}, fmt.Errorf("parse %s flags: %w", candidate.name, err)
	}
	call.Args = fs.Args()
	return call, nil
}

func parseLegacy(args []string, env flags.Environment) (Invocation, error) {
	opts, err := version.ParseArgs(args, env)
	if err != nil {
		return Invocation{}, err
	}
	run := commands()[opts.Command]
	return Invocation{
		Name:     opts.Command,
		Root:     opts.Root,
		LogLevel: opts.LogLevel,
		Env:      env,
		Runner:   legacyRunner{name: opts.Command, opts: opts, run: run},
	}, nil
}
