package projectfile

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
)

func TestApplyTo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		preset  string
		content string
		check   func(t *testing.T, cfg *config.GovernanceConfig)
	}{
		{"scalars", config.PresetStrict, fullProjectYAML, checkScalars},
		{"analyzers", config.PresetStrict, fullProjectYAML, checkAnalyzers},
		{"overrides", config.PresetStrict, fullProjectYAML, checkOverrides},
		{
			"merges into the preset",
			config.PresetStrict,
			"version: 1\nanalyzers:\n  file-size: {params: {extensions: [.go]}}\n" +
				"  complexity: {enabled: true}\n",
			checkMerged,
		},
		{
			"keeps the preset enabled",
			config.PresetMinimal,
			"version: 1\nanalyzers:\n  doc-comment: {params: {}}\n" +
				"  control-flow: {enabled: true}\n",
			checkKeepsEnabled,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil {
				t.Fatalf("decodeFile() error = %v", err)
			}
			cfg := config.Defaults()
			cfg.Preset = testCase.preset
			cfg.ApplyPreset()
			if err = file.ApplyTo(cfg); err != nil {
				t.Fatalf("ApplyTo() error = %v", err)
			}
			testCase.check(t, cfg)
		})
	}
}

func checkScalars(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if !cfg.IncludeGenerated {
		t.Error("IncludeGenerated = false, want true")
	}
	if !reflect.DeepEqual(cfg.SkipDirs, []string{"fixtures"}) {
		t.Errorf("SkipDirs = %v", cfg.SkipDirs)
	}
	if !reflect.DeepEqual(cfg.TestGlobs, []string{"web/test/**"}) {
		t.Errorf("TestGlobs = %v", cfg.TestGlobs)
	}
	if len(cfg.Delegated) != 1 || cfg.Delegated[0].ID != "dart-analyze" {
		t.Errorf("Delegated = %+v", cfg.Delegated)
	}
}

func checkAnalyzers(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if got := cfg.Analyzers["big-comment"]; !got.Enabled || got.Params["max_lines"] != 1 {
		t.Errorf("big-comment = %+v", got)
	}
	if got := cfg.Analyzers["doc-comment"]; got.Enabled {
		t.Errorf("doc-comment = %+v, want disabled", got)
	}
	if got := ruleSeverity(cfg, "doc-comment", finding.SeverityInfo); got != "error" {
		t.Errorf("severity rule for doc-comment = %q", got)
	}
}

func ruleSeverity(cfg *config.GovernanceConfig, analyzerID string, from finding.Severity) string {
	probe := []finding.Finding{{AnalyzerID: analyzerID, Severity: from}}
	return overrides.Apply(probe, analyzerID, cfg.SeverityRules)[0].Severity.String()
}

func checkOverrides(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if len(cfg.Overrides) != 1 {
		t.Fatalf("Overrides = %+v, want one", cfg.Overrides)
	}
	rules := cfg.Overrides[0].Analyzers
	if enabled := rules["single-letter-name"].Enabled; enabled == nil || *enabled {
		t.Errorf("single-letter-name = %+v, want disabled", rules["single-letter-name"])
	}
	if got := rules["file-size"].Severity; got != "warning" {
		t.Errorf("file-size override severity = %q, want warning", got)
	}
}

func checkMerged(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	fileSize := cfg.Analyzers["file-size"]
	if fileSize.Params["max_lines"] != 180 || fileSize.Params["extensions"] == nil {
		t.Errorf("file-size params = %v, want preset and file keys merged", fileSize.Params)
	}
	if got := cfg.Analyzers["complexity"].Params["max_complexity"]; got != 10 {
		t.Errorf("complexity max_complexity = %v, want preset value kept", got)
	}
}

func checkKeepsEnabled(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if cfg.Analyzers["doc-comment"].Enabled {
		t.Error("doc-comment re-enabled by a file entry that never set enabled")
	}
	if !cfg.Analyzers["control-flow"].Enabled {
		t.Error("control-flow still disabled after an explicit enabled: true")
	}
}

func TestApplyToRejectsWideningOverride(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			"disabled by the file",
			"version: 1\nanalyzers: {file-size: {enabled: false}}\n" +
				"overrides: [{files: [a], analyzers: {file-size: {enabled: true}}}]\n",
			".pulsarules.yaml: overrides[0].analyzers.file-size.enabled: " +
				"file-size is disabled for the whole project; an override can only narrow it",
		},
		{
			"sub-id names the base analyzer",
			"version: 1\nanalyzers: {golangci-lint: {enabled: false}}\n" +
				"overrides: [{files: [a], analyzers: " +
				"{golangci-lint/errcheck: {enabled: true}}}]\n",
			".pulsarules.yaml: overrides[0].analyzers.golangci-lint/errcheck.enabled: " +
				"golangci-lint is disabled for the whole project; an override can only narrow it",
		},
		{
			"enabled analyzer",
			"version: 1\noverrides: [{files: [a], analyzers: {file-size: {enabled: true}}}]\n",
			"",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil {
				t.Fatalf("decodeFile() error = %v", err)
			}
			err = file.ApplyTo(config.Defaults())
			if testCase.want == "" {
				if err != nil {
					t.Fatalf("ApplyTo() error = %v, want nil", err)
				}
				return
			}
			assertPlainMessage(t, err)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("ApplyTo() error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}

func TestApplyToBoundsWideningMessage(t *testing.T) {
	t.Parallel()

	const copies = 400
	long := "doc-comment/" + strings.Repeat("x", 1000)
	content := "version: 1\nanalyzers: {doc-comment: {enabled: false}}\noverrides:\n" +
		"  - &o {files: [a], analyzers: {\"" + long + "\": {enabled: true}}}\n" +
		strings.Repeat("  - *o\n", copies-1)
	file, err := decodeFile([]byte(content))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	err = file.ApplyTo(config.Defaults())
	if err == nil {
		t.Fatal("ApplyTo() error = nil, want the widening overrides rejected")
	}
	assertPlainMessage(t, err)
	message := err.Error()
	if len(message) > 4096 {
		t.Errorf("message is %d bytes, want under 4 KiB", len(message))
	}
	shown := strings.Count(message, "can only narrow it")
	tail := fmt.Sprintf("... and %d more problems", copies-shown)
	if shown == 0 || !strings.HasSuffix(message, tail) {
		t.Errorf("message = %q, want tail %q after %d shown", message, tail, shown)
	}
}

func TestApplyToClipsALongDelegatedID(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 300)
	content := "version: 1\nanalyzers: {" + long + ": {enabled: false}}\n" +
		"overrides: [{files: [a], analyzers: {" + long + ": {enabled: true}}}]\n"
	file, err := decodeFile([]byte(content))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	err = file.ApplyTo(config.Defaults())
	want := strings.Repeat("x", 80) + "... is disabled for the whole project"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("ApplyTo() error = %v, want containing %q", err, want)
	}
}

func globOverrides(sizes ...int) string {
	entries := make([]string, len(sizes))
	for idx, size := range sizes {
		entries[idx] = "{files: [" + strings.Repeat("a, ", size) + "]}"
	}
	return "version: 1\noverrides: [" + strings.Join(entries, ", ") + "]\n"
}

func TestValidateOverrides(t *testing.T) {
	t.Parallel()

	const head = "version: 1\noverrides: "
	const tooMany = "overrides and tests: more than 256 globs in total"
	many := make([]int, 600)
	for idx := range many {
		many[idx] = 2
	}
	runValidateCases(t, []validateCase{
		{
			"without files",
			head + "[{analyzers: {file-size: {enabled: false}}}]\n",
			"overrides[0].files: at least one glob",
		},
		{"bad glob", head + "[{files: [\"\"], analyzers: {}}]\n", "overrides[0].files[0]:"},
		{
			"unknown analyzer",
			head + "[{files: [a], analyzers: {nope: {enabled: false}}}]\n",
			"overrides[0].analyzers.nope: unknown analyzer",
		},
		{
			"bad severity",
			head + "[{files: [a], analyzers: {file-size: {severity: x}}}]\n",
			"overrides[0].analyzers.file-size.severity: invalid severity",
		},
		{
			"params",
			head + "[{files: [a], analyzers: {big-comment: {params: {max_lines: 1}}}}]\n",
			"overrides[0].analyzers.big-comment.params: " +
				"per-path params are only supported for file-size",
		},
		{
			"sub id",
			head + "[{files: [a], analyzers: {golangci-lint/errcheck: {enabled: false}}}]\n",
			"",
		},
		{"exactly 256 globs in total", globOverrides(128, 128), ""},
		{"257 globs in total", globOverrides(128, 129), tooMany},
		{"two lists of 200 and 57 globs", globOverrides(200, 57), tooMany},
		{"two globs in each of 600 overrides", globOverrides(many...), tooMany},
	})
}
