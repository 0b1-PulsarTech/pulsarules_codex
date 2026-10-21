package registry

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRules_TargetsExist(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	for analyzerID, ruleID := range Rules() {
		if !idx.RuleExists(ruleID) {
			t.Errorf("rule %q of %q is not in the knowledge index", ruleID, analyzerID)
		}
	}
	pinned := []string{"commits", "named-returns", "source-files", "flag-arguments"}
	for _, ruleID := range pinned {
		if !idx.RuleExists(ruleID) {
			t.Errorf("pinned rule %q is not in the knowledge index", ruleID)
		}
	}
}

func TestConfigRules(t *testing.T) {
	t.Parallel()

	rules := ConfigRules()
	if !slices.Contains(rules.ReservedIDs, config.AnalyzerID) {
		t.Errorf("ReservedIDs = %v, want the %q finding id", rules.ReservedIDs, config.AnalyzerID)
	}
	for _, id := range specIDs() {
		if !slices.Contains(rules.ReservedIDs, id) {
			t.Errorf("registered analyzer %q is not reserved against a delegated id", id)
		}
	}
	if want := configurableIDs(); !slices.Equal(rules.AnalyzerIDs, want) {
		t.Errorf("AnalyzerIDs = %v, want the configurable ids %v", rules.AnalyzerIDs, want)
	}
	if slices.Contains(rules.AnalyzerIDs, config.AnalyzerID) {
		t.Errorf("AnalyzerIDs = %v, want %q not configurable", rules.AnalyzerIDs, config.AnalyzerID)
	}
	if len(rules.DelegatedFormats) == 0 || len(rules.Params) == 0 {
		t.Errorf("ConfigRules() = %+v, want formats and params", rules)
	}
}

func TestPresetsNameRegisteredAnalyzers(t *testing.T) {
	t.Parallel()

	rules := ConfigRules()
	for _, name := range config.Presets() {
		cfg := config.Defaults()
		cfg.Preset = name
		cfg.ApplyPreset()
		for id, analyzer := range cfg.Analyzers {
			if !slices.Contains(rules.AnalyzerIDs, id) {
				t.Errorf("preset %q names %q, which is not a configurable analyzer", name, id)
			}
			for key := range analyzer.Params {
				if !slices.Contains(rules.Params[id], key) {
					t.Errorf("preset %q sets %s param %q, which the analyzer lacks", name, id, key)
				}
			}
		}
	}
}

func TestConfigRulesAcceptSubIDsAsDelegatedIDs(t *testing.T) {
	t.Parallel()

	rules := ConfigRules()
	for _, id := range []string{"ai-frontmatter", "short-decl-reuse"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			content := "version: 1\ndelegated:\n  - id: " + id +
				"\n    format: " + rules.DelegatedFormats[0] + "\n    run: [tool]\n"
			if _, err := projectfile.Parse("probe.yaml", []byte(content), rules); err != nil {
				t.Fatalf("Parse() error = %v, want a delegated id equal to a sub-id accepted", err)
			}
		})
	}
}
