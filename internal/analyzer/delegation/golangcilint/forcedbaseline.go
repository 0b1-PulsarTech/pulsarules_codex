package golangcilint

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var forcedBaselineReporter = core.NewReporter(
	"golangci-lint", core.SeverityWarning, core.CatSyntax,
)

// why: -E adds a linter on top of whatever the target's config ENABLES, but it loses to an explicit
// linters.disable entry - verified on golangci-lint v2.12.2 under both `default: standard` and
// `default: all`. That is the whole hole in the forced baseline, and it is silent: the run simply
// reports nothing for that linter, which is indistinguishable from finding nothing. Reading the
// config back and warning is what keeps "the baseline is forced" from being a claim we cannot make.
type disabledLinters struct {
	Linters struct {
		Disable []string `yaml:"disable"`
	} `yaml:"linters"`
}

// forcedBaselineFindings reports every forced linter the target's own config disables. A missing or
// unreadable config is not a finding: no config means nothing is disabled, which is the case -E
// already wins.
func forcedBaselineFindings(configPath string) []core.Finding {
	if configPath == "" {
		return nil
	}
	//nolint:gosec // path resolved from the analysed project dir, not from user input.
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	var cfg disabledLinters
	if err = yaml.Unmarshal(raw, &cfg); err != nil {
		return []core.Finding{forcedBaselineReporter.At(configPath, 0, fmt.Sprintf(
			"cannot read the golangci-lint config, so the forced baseline (%v) cannot be confirmed: %v",
			forcedLinters,
			err,
		), "fix the YAML so the forced-linter baseline can be verified")}
	}

	var findings []core.Finding
	for _, linter := range cfg.Linters.Disable {
		if !slices.Contains(forcedLinters, linter) {
			continue
		}
		findings = append(findings, forcedBaselineReporter.At(configPath, 0, fmt.Sprintf(
			"linters.disable turns off %q, which this baseline forces via -E; an explicit disable "+
				"wins over -E, so that check silently reports nothing", linter,
		), fmt.Sprintf("drop %q from linters.disable, or accept the gap knowingly", linter)))
	}
	return findings
}
