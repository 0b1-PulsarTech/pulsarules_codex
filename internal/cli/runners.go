package cli

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/commitlint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/evals"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/generate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/list"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/packaging"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/validate"
)

func generateRunner(flags.Environment) runner {
	cmd := &generate.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.Index, svc.Renderer)
	}}
}

func validateRunner(flags.Environment) runner {
	cmd := validate.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.Index)
	}}
}

func listRunner(flags.Environment) runner {
	cmd := list.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, args []string) error {
		return cmd.Run(svc.Index, args)
	}}
}

func installRunner(env flags.Environment) runner {
	cmd := install.New(env)
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.Index, svc.Templates, svc.Renderer, svc.Targets)
	}}
}

func uninstallRunner(env flags.Environment) runner {
	cmd := uninstall.New(env)
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.Targets)
	}}
}

func packageRunner(flags.Environment) runner {
	cmd := &packaging.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.Index, svc.Renderer)
	}}
}

func commitlintRunner(flags.Environment) runner {
	cmd := &commitlint.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.OpenRepository, svc.Index, svc.LoadCatalog, svc.Loader)
	}}
}

func governanceRunner(env flags.Environment) runner {
	cmd := governance.New(env)
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.OpenRepository, svc.Index, svc.LoadCatalog, svc.Loader)
	}}
}

func cleanRunner(flags.Environment) runner {
	cmd := &clean.Command{}
	return runner{command: cmd, run: func(svc bootstrap.Services, _ []string) error {
		return cmd.Run(svc.OpenRepository)
	}}
}

func evalsRunner(flags.Environment) runner {
	cmd := &evals.Command{}
	return runner{command: cmd, run: func(_ bootstrap.Services, _ []string) error {
		return cmd.Run()
	}}
}

func hookRunner(env flags.Environment) runner {
	cmd := hook.New(env)
	return runner{command: cmd, run: func(svc bootstrap.Services, args []string) error {
		return cmd.Run(hook.Collaborators{
			Index:       svc.Index,
			Templates:   svc.Templates,
			OpenLogger:  svc.OpenLogger,
			LoadCatalog: svc.LoadCatalog,
			Loader:      svc.Loader,
		}, args)
	}}
}
