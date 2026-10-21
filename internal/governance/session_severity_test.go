package governance

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

func TestToAnalysisConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetMinimal
	cfg.ApplyPreset()

	cfg.Analyzers["kept-on"] = config.AnalyzerConfig{Enabled: true}

	pc := toAnalysisConfig(cfg)
	if pc == nil {
		t.Fatal("expected non-nil pipeline config")
	}
	testCases := []struct {
		id      string
		enabled bool
	}{
		{"complexity", false},
		{"kept-on", true},
	}
	for _, testCase := range testCases {
		got, found := pc.Analyzers[testCase.id]
		if !found || got.Enabled != testCase.enabled {
			t.Errorf("%s = %+v (found %v), want Enabled %v kept", testCase.id, got, found,
				testCase.enabled)
		}
	}
}

func TestToAnalysisConfig_ExplicitSetParam(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.SetParam(movepurity.ID, movepurity.ParamMinSimilarity, 99)

	pc := toAnalysisConfig(cfg)
	params := pc.Analyzers[movepurity.ID].Params
	if got := params[movepurity.ParamMinSimilarity]; got != 99 {
		t.Fatalf("min_similarity = %v, want explicit 99 to survive projection", got)
	}
}

func TestToAnalysisConfigProjectFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(cfg *config.GovernanceConfig)
		check func(t *testing.T, cfg *config.GovernanceConfig, got *pass.AnalysisConfig)
	}{
		{"tests and delegated", setTestsAndDelegated, checkTestsAndDelegated},
		{"severity rules precede file rules", setSeverityRules, checkSeverityRules},
		{"disabled delegated is dropped", setDisabledDelegated, checkDisabledDelegated},
		{"no project file", func(*config.GovernanceConfig) {}, checkNoProjectFields},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Defaults()
			testCase.setup(cfg)
			testCase.check(t, cfg, toAnalysisConfig(cfg))
		})
	}
}

func setTestsAndDelegated(cfg *config.GovernanceConfig) {
	cfg.TestGlobs = []string{"web/test/**"}
	cfg.Delegated = []pass.DelegatedCommand{{ID: "dart-analyze", Run: []string{"dart"}}}
}

func checkTestsAndDelegated(t *testing.T, cfg *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if !reflect.DeepEqual(got.TestGlobs, cfg.TestGlobs) {
		t.Errorf("TestGlobs = %v", got.TestGlobs)
	}
	if !reflect.DeepEqual(got.Delegated, cfg.Delegated) {
		t.Errorf("Delegated = %v", got.Delegated)
	}
}

func setSeverityRules(cfg *config.GovernanceConfig) {
	cfg.SetSeverity("naming", "info")
	cfg.SetSeverity("big-comment", "error")
	cfg.Overrides = []pass.Override{{Files: glob.CompileSet([]string{"a/**"})}}
}

func checkSeverityRules(t *testing.T, _ *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 3 {
		t.Fatalf("Overrides = %+v, want 2 severity rules then the file rule", got.Overrides)
	}
	if _, first := got.Overrides[0].Analyzers["naming"]; !first {
		t.Errorf("severity rules lost their order: %+v", got.Overrides)
	}
	if !got.Overrides[2].Covers("a/x") || got.Overrides[2].Covers("b/x") {
		t.Errorf("file rule must come after the severity rules: %+v", got.Overrides)
	}
}

func checkNoProjectFields(t *testing.T, _ *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 0 || len(got.TestGlobs) != 0 || len(got.Delegated) != 0 {
		t.Fatalf("project fields set without a project file: %+v", got)
	}
}

func setDisabledDelegated(cfg *config.GovernanceConfig) {
	cfg.Delegated = []pass.DelegatedCommand{
		{ID: "off", Run: []string{"a"}},
		{ID: "on"},
		{ID: "unlisted"},
	}
	cfg.Analyzers = map[string]config.AnalyzerConfig{
		"off": {Enabled: false},
		"on":  {Enabled: true},
	}
}

func delegatedIDs(entries []pass.DelegatedCommand) []string {
	ids := make([]string, len(entries))
	for idx, entry := range entries {
		ids[idx] = entry.ID
	}
	return ids
}

func checkDisabledDelegated(
	t *testing.T, cfg *config.GovernanceConfig, got *pass.AnalysisConfig,
) {
	t.Helper()
	if ids := delegatedIDs(got.Delegated); !slices.Equal(ids, []string{"on", "unlisted"}) {
		t.Errorf("Delegated ids = %v, want [on unlisted]", ids)
	}
	wantSource := []string{"off", "on", "unlisted"}
	if ids := delegatedIDs(cfg.Delegated); !slices.Equal(ids, wantSource) {
		t.Errorf("source Delegated ids = %v, want %v", ids, wantSource)
	}
}

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
		{"params severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetParam(typographic, "severity", "info")
		}, "info", "info"},
		{
			"analyzer severity beats an earlier params severity",
			"",
			func(cfg *config.GovernanceConfig) {
				cfg.SetParam(typographic, "severity", "info")
				cfg.SetSeverity(typographic, "warning")
			},
			"warning",
			"warning",
		},
		{"per-path override beats the global severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.Overrides = append(cfg.Overrides, webOverride("info"))
		}, "info", "warning"},
		{"CLI flag beats the file severity", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.SetParam(typographic, "severity", "info")
		}, "info", "info"},
		{"per-path override beats the CLI flag", "", func(cfg *config.GovernanceConfig) {
			cfg.SetSeverity(typographic, "warning")
			cfg.Overrides = append(cfg.Overrides, webOverride("error"))
			cfg.SetParam(typographic, "severity", "info")
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
			"params severity on text-markers reaches ai-frontmatter",
			func(cfg *config.GovernanceConfig) { cfg.SetParam("text-markers", "severity", "warning") },
			map[string]string{"ai-frontmatter doc.md": "warning", "file-size big.go": "error"},
		},
		{
			"analyzer severity on text-markers reaches ai-frontmatter",
			func(cfg *config.GovernanceConfig) { cfg.SetSeverity("text-markers", "info") },
			map[string]string{"ai-frontmatter doc.md": "info", "file-size big.go": "error"},
		},
		{
			"params severity on file-size lowers the hard cap too",
			func(cfg *config.GovernanceConfig) { cfg.SetParam("file-size", "severity", "warning") },
			map[string]string{"ai-frontmatter doc.md": "error", "file-size big.go": "warning"},
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

func TestMovePurityDefaultSeverityIsOwnedByTheAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := toAnalysisConfig(config.Defaults())
	if _, hasParam := cfg.Analyzers[movepurity.ID].Params["severity"]; hasParam {
		t.Fatalf(
			"config projects a severity for %s: %+v",
			movepurity.ID,
			cfg.Analyzers[movepurity.ID],
		)
	}
	if len(cfg.Overrides) != 0 {
		t.Fatalf("default config carries severity rules: %+v", cfg.Overrides)
	}
}
