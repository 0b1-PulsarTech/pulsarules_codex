package golangcilint

import (
	"fmt"
	"os"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/safeyaml"
)

type disabledLinters struct {
	Linters struct {
		Disable []string `yaml:"disable"`
	} `yaml:"linters"`
}

func (analyzer *Analyzer) forcedBaselineFindings(configPath string) []finding.Finding {
	if configPath == "" {
		return nil
	}
	//nolint:gosec // path resolved from the analysed project dir, not from user input.
	raw, err := os.ReadFile(configPath)
	// A missing or unreadable config disables nothing, which is the case -E already wins.
	if err != nil {
		return nil
	}
	forcedLinters := forcedLinters()
	var cfg disabledLinters
	if err = safeyaml.Unmarshal(raw, &cfg); err != nil {
		return []finding.Finding{analyzer.warning.At(configPath, 0, fmt.Sprintf(
			"cannot read the golangci-lint config, so the forced baseline "+
				"(%v) cannot be confirmed: %v",
			forcedLinters,
			err,
		), "fix the YAML so the forced-linter baseline can be verified")}
	}

	var findings []finding.Finding
	for _, linter := range cfg.Linters.Disable {
		if !slices.Contains(forcedLinters, linter) {
			continue
		}
		// An explicit disable beats -E (golangci-lint v2.12.2, under both `default: standard`
		// and `default: all`), and silently: the run reports nothing for that linter.
		findings = append(findings, analyzer.warning.At(configPath, 0, fmt.Sprintf(
			"linters.disable turns off %q, which this baseline forces via -E; "+
				"an explicit disable wins over -E, so that check silently reports nothing",
			linter,
		), fmt.Sprintf("drop %q from linters.disable, or accept the gap knowingly", linter)))
	}
	return findings
}
