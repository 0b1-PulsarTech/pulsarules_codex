package cli

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
)

func Run(inj remy.Injector, opts *cliopts.Options) error {
	switch opts.Command {
	case "generate":
		return runGenerate(inj, opts)
	case "list":
		return runList(inj, opts)
	case "validate":
		return runValidate(inj, opts)
	case "package":
		return runPackage(inj, opts)
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

func BootstrapOptions(opts *cliopts.Options) bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	return bootstrap.Options{
		Root: opts.Root,
	}
}
