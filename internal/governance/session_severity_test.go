package governance

import (
	"cmp"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func severityRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	dash := "package x\n\n// note \u2014 dash\n"
	writeFileAtDir(t, dir, "web/a.go", dash)
	writeFileAtDir(t, dir, "docs/a.go", dash)
	writeFileAtDir(t, dir, "doc.md", "---\ngenerator: claude\n---\nbody\n")
	writeFileAtDir(t, dir, "big.go", "package big\n"+strings.Repeat("// line\n", 20))
	return dir
}

func severitiesByFile(t *testing.T, dir string, cfg *config.GovernanceConfig) map[string]string {
	t.Helper()

	sess := NewSession(&goldenRepo{root: dir}, "", nil, testCatalog(t), cfg)
	got := map[string]string{}
	for _, finding := range sess.Analyze(ScopeChanged, nil, FileSetAll).Findings {
		got[finding.AnalyzerID+" "+finding.File] = finding.Severity.String()
	}
	return got
}

func webOverride(severity string) pass.Override {
	return pass.NewOverride([]string{"web/**"}, map[string]pass.OverrideConfig{
		"typographic-markers": {Severity: severity},
	})
}

func TestSeverityPrecedence(t *testing.T) {
	t.Parallel()

	const typographic = "typographic-markers"
	testCases := []struct {
		name    string
		preset  string
		setup   func(cfg *config.GovernanceConfig)
		wantWeb string
		wantDoc string
	}{
		{"analyzer default", "", func(*config.GovernanceConfig) {}, "error", "error"},
		{
			"strict preset leaves severity alone", config.PresetStrict,
			func(*config.GovernanceConfig) {}, "error", "error",
		},
		{"global analyzer severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
		}, "warning", "warning"},
		{"per-path override beats the global severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.Overrides = append(cfg.Overrides, webOverride("info"))
		}, "info", "warning"},
		{"CLI flag beats the file severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.SetSeverity(typographic, "info")
		}, "info", "info"},
		{"per-path override beats the CLI flag", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.Overrides = append(cfg.Overrides, webOverride("error"))
			cfg.SetSeverity(typographic, "info")
		}, "error", "info"},
		{"later per-path override wins", "", func(cfg *config.GovernanceConfig) {
			cfg.Overrides = append(cfg.Overrides, webOverride("info"), webOverride("warning"))
		}, "warning", "error"},
	}
	dir := severityRepo(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Defaults()
			cfg.Preset = cmp.Or(testCase.preset, cfg.Preset)
			cfg.ApplyPreset()
			testCase.setup(cfg)
			got := severitiesByFile(t, dir, cfg)
			if got[typographic+" web/a.go"] != testCase.wantWeb {
				t.Errorf(
					"web/a.go severity = %q, want %q",
					got[typographic+" web/a.go"],
					testCase.wantWeb,
				)
			}
			if got[typographic+" docs/a.go"] != testCase.wantDoc {
				t.Errorf(
					"docs/a.go severity = %q, want %q",
					got[typographic+" docs/a.go"],
					testCase.wantDoc,
				)
			}
		})
	}
}

func TestOwnerSeverityGovernsSubIDs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(cfg *config.GovernanceConfig)
		want  map[string]string
	}{
		{"no severity keeps the defaults", func(*config.GovernanceConfig) {}, map[string]string{
			"ai-frontmatter doc.md": "error", "file-size big.go": "error",
		}},
		{
			"analyzer severity on text-markers reaches ai-frontmatter",
			func(cfg *config.GovernanceConfig) { cfg.SetSeverity("text-markers", "info") },
			map[string]string{"ai-frontmatter doc.md": "info", "file-size big.go": "error"},
		},
		{
			"file-size severity lowers the hard cap too",
			func(cfg *config.GovernanceConfig) { cfg.SetSeverity("file-size", "warning") },
			map[string]string{"ai-frontmatter doc.md": "error", "file-size big.go": "warning"},
		},
	}
	dir := severityRepo(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Defaults()
			cfg.ApplyPreset()
			cfg.SetParam("file-size", "max_lines", 5)
			cfg.SetParam("file-size", "error_lines", 10)
			testCase.setup(cfg)
			got := severitiesByFile(t, dir, cfg)
			for key, want := range testCase.want {
				if got[key] != want {
					t.Errorf("%s severity = %q, want %q (all: %v)", key, got[key], want, got)
				}
			}
		})
	}
}
