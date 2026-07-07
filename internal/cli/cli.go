package cli

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/generate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/list"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/packaging"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/validate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func Run(inj remy.Injector, opts *version.Options) error {
	switch opts.Command {
	case "generate":
		return generate.RunGenerate(inj, opts)
	case "list":
		return list.RunList(inj, opts)
	case "validate":
		return validate.RunValidate(inj, opts)
	case "package":
		return packaging.RunPackage(inj, opts)
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

func BootstrapOptions(opts *version.Options) bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	return bootstrap.Options{
		Root: opts.Root,
	}
}
