package loader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, config.Name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

type loadCase struct {
	name     string
	content  string
	explicit bool
	missing  bool
	noRoot   bool
	layout   string
	wantNil  bool
	wantErr  string
}

func (testCase loadCase) setup(t *testing.T) (root, explicit string) {
	t.Helper()
	if testCase.noRoot {
		return "", ""
	}
	root = t.TempDir()
	switch {
	case testCase.layout != "":
		return root, testCase.special(t, root)
	case testCase.missing && testCase.explicit:
		return root, filepath.Join(root, "nope.yaml")
	case testCase.missing:
		return root, ""
	}
	path := writeConfig(t, root, testCase.content)
	if testCase.explicit {
		return root, path
	}
	return root, ""
}

func (testCase loadCase) special(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "special.yaml")
	switch testCase.layout {
	case "fifo":
		makeFifo(t, path)
	case "dir":
		return root
	case "device":
		if err := os.Symlink("/dev/zero", path); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	case "link":
		if err := os.Symlink(writeConfig(t, root, testCase.content), path); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}
	return path
}

func fileKindCases() []loadCase {
	const notRegular = ": read config: not a regular file"
	return []loadCase{
		{name: "a fifo", layout: "fifo", wantErr: notRegular},
		{name: "a directory", layout: "dir", wantErr: notRegular},
		{name: "a symlink to a device", layout: "device", wantErr: notRegular},
		{name: "a symlink to a regular file", layout: "link", content: "version: 1\n"},
	}
}

func loadCases() []loadCase {
	return slices.Concat(fileKindCases(), reservedIDCases(), []loadCase{
		{name: "valid", content: "version: 1\nanalyzers: {big-comment: {enabled: false}}\n"},
		{name: "no root", noRoot: true, wantNil: true},
		{name: "default file absent", missing: true, wantNil: true},
		{name: "explicit file absent", missing: true, explicit: true, wantErr: "read config"},
		{
			name:    "unknown key",
			content: "version: 1\nbogus: 1\n",
			wantErr: ".pulsarules.yaml: line 2: bogus: unknown key",
		},
		{
			name:    "unknown param",
			content: "version: 1\nanalyzers: {file-size: {params: {max_line: 1}}}\n",
			wantErr: ".pulsarules.yaml: analyzers.file-size.params.max_line: unknown param",
		},
		{
			name:    "pipeline analyzer is not configurable",
			content: "version: 1\nanalyzers: {output: {enabled: false}}\n",
			wantErr: ".pulsarules.yaml: analyzers.output: unknown analyzer",
		},
		{
			name:    "rule-injection is not configurable",
			content: "version: 1\nanalyzers: {rule-injection: {enabled: false}}\n",
			wantErr: "analyzers.rule-injection: unknown analyzer",
		},
		{
			name:     "explicit path named in the error",
			content:  "version: 3\n",
			explicit: true,
			wantErr:  "version: unsupported 3",
		},
	})
}

func reservedIDCases() []loadCase {
	return []loadCase{
		{
			name: "delegated id of a registry analyzer",
			content: "version: 1\ndelegated: [{id: rule-injection, run: [x], " +
				"format: dart-analyze-json}]\n",
			wantErr: "delegated[0].id: \"rule-injection\" is a built-in analyzer id",
		},
		{
			name: "delegated id of the config analyzer",
			content: "version: 1\ndelegated: [{id: config, run: [x], " +
				"format: dart-analyze-json}]\n",
			wantErr: "delegated[0].id: \"config\" is a built-in analyzer id",
		},
	}
}

type loadResult struct {
	file *config.File
	err  error
}

func loadWithin(t *testing.T, root, explicit string) loadResult {
	t.Helper()
	const deadline = 5 * time.Second
	done := make(chan loadResult, 1)
	go func() {
		file, err := Load(root, explicit)
		done <- loadResult{file: file, err: err}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case result := <-done:
		return result
	case <-ctx.Done():
		t.Fatalf("Load() still blocked after %v: a special file was opened without O_NONBLOCK",
			deadline)
	}
	return loadResult{}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	for _, testCase := range loadCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root, explicit := testCase.setup(t)
			result := loadWithin(t, root, explicit)
			file, err := result.file, result.err
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("Load() error = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if (file == nil) != testCase.wantNil {
				t.Fatalf("Load() file = %+v, wantNil %v", file, testCase.wantNil)
			}
		})
	}
}

func TestBuildPrecedence(t *testing.T) {
	t.Parallel()

	file := &config.File{
		Version: 1,
		Preset:  config.PresetMinimal,
		Analyzers: map[string]config.AnalyzerEntry{
			"file-size": {Params: map[string]any{"max_lines": 7}},
		},
	}
	testCases := []struct {
		name       string
		file       *config.File
		flag       string
		wantPreset string
		wantLines  any
	}{
		{"no file, no flag", nil, "", config.PresetRecommended, nil},
		{"no file, flag", nil, config.PresetStrict, config.PresetStrict, 180},
		{"file preset", file, "", config.PresetMinimal, 7},
		{"flag beats file preset", file, config.PresetStrict, config.PresetStrict, 7},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Build(testCase.file, testCase.flag)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if cfg.Preset != testCase.wantPreset {
				t.Errorf("Preset = %q, want %q", cfg.Preset, testCase.wantPreset)
			}
			if got := cfg.Analyzers["file-size"].Params["max_lines"]; got != testCase.wantLines {
				t.Errorf("max_lines = %v, want %v", got, testCase.wantLines)
			}
		})
	}
}

type enabledCase struct {
	name        string
	preset      string
	analyzers   map[string]config.AnalyzerEntry
	overrides   []map[string]config.OverrideRules
	flag        string
	wantEnabled map[string]bool
	wantErr     string
}

func enabledCases() []enabledCase {
	enable, disable := true, false
	narrow := config.OverrideRules{Enabled: &enable}
	fileSizeOff := map[string]config.AnalyzerEntry{"file-size": {Enabled: &disable}}
	return []enabledCase{
		{
			name:        "file beats the minimal preset",
			preset:      config.PresetMinimal,
			analyzers:   map[string]config.AnalyzerEntry{"control-flow": {Enabled: &enable}},
			wantEnabled: map[string]bool{"control-flow": true, "complexity": false},
		},
		{
			name:      "override enabling a file-disabled analyzer",
			analyzers: fileSizeOff,
			overrides: []map[string]config.OverrideRules{{"file-size": narrow}},
			wantErr: ".pulsarules.yaml: overrides[0].analyzers.file-size.enabled: " +
				"file-size is disabled for the whole project; an override can only narrow it",
		},
		{
			name:      "override enabling a preset-disabled analyzer",
			preset:    config.PresetMinimal,
			overrides: []map[string]config.OverrideRules{{"complexity": narrow}},
			wantErr:   "overrides[0].analyzers.complexity.enabled: complexity is disabled",
		},
		{
			name:      "override enabling a disabled analyzer by sub-id",
			analyzers: map[string]config.AnalyzerEntry{"golangci-lint": {Enabled: &disable}},
			overrides: []map[string]config.OverrideRules{{"golangci-lint/errcheck": narrow}},
			wantErr: "overrides[0].analyzers.golangci-lint/errcheck.enabled: " +
				"golangci-lint is disabled",
		},
		{
			name:      "override disabling stays valid",
			analyzers: fileSizeOff,
			overrides: []map[string]config.OverrideRules{{"file-size": {Enabled: &disable}}},
		},
		{
			name:      "override enabling an enabled analyzer",
			overrides: []map[string]config.OverrideRules{{"file-size": narrow}},
		},
	}
}

func presetEnabledCases() []enabledCase {
	enable, disable := true, false
	narrow := config.OverrideRules{Enabled: &enable}
	return []enabledCase{
		{
			name:   "preset enables, one override disables, a later one re-enables",
			preset: config.PresetStrict,
			overrides: []map[string]config.OverrideRules{
				{"complexity": {Enabled: &disable}}, {"complexity": narrow},
			},
			wantEnabled: map[string]bool{"complexity": true},
		},
		{
			name:        "flag preset minimal disables for the whole project",
			flag:        config.PresetMinimal,
			wantEnabled: map[string]bool{"complexity": false},
		},
		{
			name:      "override enabling an analyzer the flag preset disabled",
			flag:      config.PresetMinimal,
			overrides: []map[string]config.OverrideRules{{"complexity": narrow}},
			wantErr:   "overrides[0].analyzers.complexity.enabled: complexity is disabled",
		},
	}
}

func (testCase enabledCase) projectFile() *config.File {
	file := &config.File{
		Path:      config.Name,
		Version:   1,
		Preset:    testCase.preset,
		Analyzers: testCase.analyzers,
	}
	for _, rules := range testCase.overrides {
		file.Overrides = append(file.Overrides,
			config.OverrideEntry{Files: []string{"a/**"}, Analyzers: rules})
	}
	return file
}

func TestBuildEnabled(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(enabledCases(), presetEnabledCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Build(testCase.projectFile(), testCase.flag)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("Build() error = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			for id, want := range testCase.wantEnabled {
				if got := cfg.Analyzers[id].Enabled; got != want {
					t.Errorf("%s enabled = %v, want %v", id, got, want)
				}
			}
		})
	}
}

func TestForRoot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		content      string
		wantMessage  string
		wantMaxLines any
	}{
		{name: "no file"},
		{
			name:         "valid file is applied",
			content:      "version: 1\nanalyzers: {big-comment: {params: {max_lines: 1}}}\n",
			wantMaxLines: 1,
		},
		{name: "unknown key", content: "version: 1\nbogus: 1\n", wantMessage: "bogus"},
		{
			name: "override enabling a disabled analyzer",
			content: "version: 1\nanalyzers: {file-size: {enabled: false}}\n" +
				"overrides: [{files: [a], analyzers: {file-size: {enabled: true}}}]\n",
			wantMessage: "can only narrow it",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if testCase.content != "" {
				writeConfig(t, dir, testCase.content)
			}
			cfg, findings := ForRoot(dir)
			if cfg.Preset != config.PresetRecommended || len(cfg.Overrides) != 0 {
				t.Errorf("cfg = %+v, want defaults", cfg)
			}
			if testCase.wantMessage == "" {
				got := cfg.Analyzers["big-comment"].Params["max_lines"]
				if len(findings) != 0 || got != testCase.wantMaxLines {
					t.Fatalf("ForRoot() = %+v, %+v", got, findings)
				}
				return
			}
			assertConfigFinding(t, findings, config.Name, testCase.wantMessage)
		})
	}
}

func assertConfigFinding(t *testing.T, findings []finding.Finding, file, message string) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want exactly one", findings)
	}
	got := findings[0]
	if got.AnalyzerID != ConfigAnalyzerID || got.Severity != finding.SeverityError {
		t.Errorf("finding = %+v, want an error from the config analyzer", got)
	}
	if got.File != file || !strings.Contains(got.Message, message) {
		t.Errorf("finding = %+v, want file %q and message %q", got, file, message)
	}
}

func TestForRepo(t *testing.T) {
	t.Parallel()

	cfg, findings := ForRepo(nil)
	if len(findings) != 0 || cfg.Preset != config.PresetRecommended {
		t.Fatalf("ForRepo(nil) = %+v, %+v", cfg, findings)
	}

	dir := t.TempDir()
	if err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	writeConfig(t, dir, "version: 1\nbogus: 1\n")
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	if _, findings = ForRepo(repo); len(findings) != 1 {
		t.Fatalf("ForRepo(repo) findings = %+v, want the config error", findings)
	}
}

func TestRulesReserveEveryBuiltInID(t *testing.T) {
	t.Parallel()

	reserved := Rules().ReservedIDs
	for _, id := range append(registry.IDs(), ConfigAnalyzerID) {
		if !slices.Contains(reserved, id) {
			t.Errorf("ReservedIDs lacks %q", id)
		}
	}
}
