package flags

import (
	"flag"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/character"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/branch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

type Tuning struct {
	TypographicSeverity string
	BranchExtraTypes    string
}

func (tuning *Tuning) Bind(fs *flag.FlagSet) {
	fs.StringVar(
		&tuning.TypographicSeverity,
		"typographic-severity",
		"",
		"how hard a typographic-marker finding lands (`error|warning|info`)",
	)
	fs.StringVar(
		&tuning.BranchExtraTypes,
		"branch-extra-types",
		"",
		"comma-separated branch types (`a,b`) allowed beyond the Conventional Commit "+
			"and gitflow types",
	)
}

func (tuning Tuning) Validate() error {
	// A typo that installs cleanly would fail every commit from inside the hook.
	if err := finding.ValidateSeverityName(tuning.TypographicSeverity); err != nil {
		return fmt.Errorf("invalid --typographic-severity: %w", err)
	}
	// The extra types are written into a shell script, so the alphabet is checked here.
	if err := branch.ValidateExtraTypes(tuning.BranchExtraTypes); err != nil {
		return fmt.Errorf("invalid --branch-extra-types: %w", err)
	}
	return nil
}

func (tuning Tuning) Apply(cfg *config.GovernanceConfig) {
	if tuning.TypographicSeverity != "" {
		cfg.SetSeverity(character.TypographicID, tuning.TypographicSeverity)
	}
	if tuning.BranchExtraTypes != "" {
		cfg.SetParam(branch.ID, branch.ParamExtraTypes, tuning.BranchExtraTypes)
	}
}
