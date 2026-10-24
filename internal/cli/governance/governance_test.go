package governance

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRunGovernance_RequiresProjectDir(t *testing.T) {
	t.Parallel()

	err := (&Command{}).Run(unopenableRepository(), nil, nil, nil)
	if !errors.Is(err, errProjectDirRequired) {
		t.Errorf("err = %v, want %v", err, errProjectDirRequired)
	}
}

func TestRunGovernance_EnvProjectDirSatisfiesTheRequirement(t *testing.T) {
	t.Parallel()

	opts := &Command{env: flags.Environment{ProjectDir: t.TempDir()}}
	err := opts.Run(unopenableRepository(), nil, nil, nil)
	if errors.Is(err, errProjectDirRequired) {
		t.Errorf("err = %v, want the env project dir to count", err)
	}
}

func TestGovernanceConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *Command
		wantPreset string
		wantPath   any
	}{
		{
			name:       "no flags keeps the default preset",
			opts:       &Command{},
			wantPreset: config.PresetRecommended,
		},
		{
			name:       "preset flag wins",
			opts:       &Command{preset: config.PresetStrict},
			wantPreset: config.PresetStrict,
		},
		{
			name:       "golangci config lands in the analyzer params",
			opts:       &Command{golangciConfig: "build/.golangci.yml"},
			wantPreset: config.PresetRecommended,
			wantPath:   "build/.golangci.yml",
		},
		{
			name:       "preset and config path compose",
			opts:       &Command{preset: config.PresetMinimal, golangciConfig: "x.yml"},
			wantPreset: config.PresetMinimal,
			wantPath:   "x.yml",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := testCase.opts.governanceConfig(testLoader(), "")
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if cfg.Preset != testCase.wantPreset {
				t.Errorf("Preset = %q, want %q", cfg.Preset, testCase.wantPreset)
			}
			got := cfg.Analyzers["golangci-lint"].Params["config_path"]
			if got != testCase.wantPath {
				t.Errorf("config_path = %v, want %v", got, testCase.wantPath)
			}
		})
	}
}

func TestGovernanceConfig_InvalidPreset(t *testing.T) {
	t.Parallel()

	const want = `invalid --preset "stict" (want recommended|strict|minimal)`
	_, err := (&Command{preset: "stict"}).governanceConfig(testLoader(), "")
	if err == nil || err.Error() != want {
		t.Fatalf("governanceConfig error = %v, want %q", err, want)
	}
}

func TestGovernanceConfig_IncludeGenerated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts *Command
		want bool
	}{
		{name: "suppressed by default", opts: &Command{}, want: false},
		{name: "flag turns the filter off", opts: &Command{includeGenerated: true}, want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := testCase.opts.governanceConfig(testLoader(), "")
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.IncludeGenerated; got != testCase.want {
				t.Errorf("IncludeGenerated = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGovernanceConfig_BranchExtraTypes(t *testing.T) {
	t.Parallel()

	const wantMessage = `invalid --branch-extra-types: "Release" is not a comma-separated list ` +
		`of lowercase names, e.g. spike,experiment`

	testCases := []struct {
		name    string
		spec    string
		wantErr string
		wantSet any
	}{
		{name: "unset leaves the params alone", wantSet: nil},
		{
			name:    "a list reaches the analyzer",
			spec:    "spike,experiment",
			wantSet: "spike,experiment",
		},
		{name: "uppercase names the flag and the form", spec: "Release", wantErr: wantMessage},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			opts := &Command{tuning: flags.Tuning{BranchExtraTypes: testCase.spec}}
			cfg, err := opts.governanceConfig(testLoader(), "")
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("err = %v, want one naming %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.Analyzers["branch-name"].Params["extra_types"]; got != testCase.wantSet {
				t.Errorf("extra_types param = %v, want %v", got, testCase.wantSet)
			}
		})
	}
}

func writeProjectConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, projectfile.Name)
	if err := os.WriteFile(path, []byte(content), disk.ModeFile); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	return dir
}

type precedenceCase struct {
	name         string
	opts         *Command
	wantPreset   string
	wantGolangci any
	wantSeverity string
}

func (testCase precedenceCase) assert(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if cfg.Preset != testCase.wantPreset {
		t.Errorf("Preset = %q, want %q", cfg.Preset, testCase.wantPreset)
	}
	if got := cfg.Analyzers["big-comment"].Params["max_lines"]; got != 1 {
		t.Errorf("big-comment max_lines = %v, want the file's 1", got)
	}
	got := cfg.Analyzers["golangci-lint"].Params["config_path"]
	if got != testCase.wantGolangci {
		t.Errorf("config_path = %v, want %v", got, testCase.wantGolangci)
	}
	if got = effectiveSeverity(cfg, "typographic-markers"); got != testCase.wantSeverity {
		t.Errorf("effective severity = %q, want %q", got, testCase.wantSeverity)
	}
	if !cfg.IncludeGenerated {
		t.Error("IncludeGenerated = false, want the file's true")
	}
}

func TestGovernanceConfig_ProjectFilePrecedence(t *testing.T) {
	t.Parallel()

	const content = "version: 1\npreset: minimal\ninclude_generated: true\n" +
		"analyzers:\n  big-comment: {params: {max_lines: 1}}\n" +
		"  golangci-lint: {params: {config_path: file.yml}}\n" +
		"  typographic-markers: {severity: error}\n"
	root := writeProjectConfig(t, content)

	testCases := []precedenceCase{
		{"file only", &Command{}, config.PresetMinimal, "file.yml", "error"},
		{
			"preset flag beats the file",
			&Command{preset: config.PresetStrict},
			config.PresetStrict, "file.yml", "error",
		},
		{
			"param flag beats the file",
			&Command{golangciConfig: "flag.yml"},
			config.PresetMinimal, "flag.yml", "error",
		},
		{
			"severity flag beats the file",
			&Command{tuning: flags.Tuning{TypographicSeverity: "warning"}},
			config.PresetMinimal, "file.yml", "warning",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := testCase.opts.governanceConfig(testLoader(), root)
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			testCase.assert(t, cfg)
		})
	}
}

func TestGovernanceConfig_ProjectFileErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{"unknown key", "version: 1\nbogus: 1\n", ".pulsarules.yaml: line 2: bogus: unknown key"},
		{"second document", "version: 1\n---\nversion: 1\n", "only one YAML document is allowed"},
		{
			"widening override",
			"version: 1\nanalyzers: {file-size: {enabled: false}}\n" +
				"overrides: [{files: [a], analyzers: {file-size: {enabled: true}}}]\n",
			"overrides[0].analyzers.file-size.enabled: file-size is disabled",
		},
		{
			"unknown param", "version: 1\nanalyzers: {file-size: {params: {max_line: 1}}}\n",
			".pulsarules.yaml: analyzers.file-size.params.max_line: unknown param",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := writeProjectConfig(t, testCase.content)
			_, err := (&Command{}).governanceConfig(testLoader(), root)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("governanceConfig error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}

func TestGovernanceConfig_ExplicitConfigPath(t *testing.T) {
	t.Parallel()

	const content = "version: 1\nanalyzers: {big-comment: {params: {max_lines: 2}}}\n"
	dir := writeProjectConfig(t, content)
	explicit := filepath.Join(dir, projectfile.Name)

	cfg, err := (&Command{config: explicit}).governanceConfig(testLoader(), "")
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	if got := cfg.Analyzers["big-comment"].Params["max_lines"]; got != 2 {
		t.Errorf("max_lines = %v, want 2", got)
	}
	missing := &Command{config: filepath.Join(dir, "nope.yaml")}
	if _, err = missing.governanceConfig(testLoader(), dir); err == nil {
		t.Error("governanceConfig with a missing --config: expected an error")
	}
}

func TestGovernanceConfig_NoProjectFileIsUnchanged(t *testing.T) {
	t.Parallel()

	cfg, err := (&Command{}).governanceConfig(testLoader(), t.TempDir())
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	want := config.Defaults()
	want.ApplyPreset()
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("cfg = %+v, want the plain defaults %+v", cfg, want)
	}
}

func TestGovernanceConfig_PresetFlag(t *testing.T) {
	t.Parallel()

	const (
		base = "version: 1\noverrides:\n" +
			"  - {files: ['lib/**'], analyzers: {doc-comment: {enabled: true}}}\n"
		private = "version: 1\npreset: minimal\n"
	)
	testCases := []struct {
		name, preset  string
		wantOverrides int
	}{
		{"private preset drops the base override", "", 0},
		{"the flag restores the preset the override needs", config.PresetRecommended, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := writeProjectConfig(t, base)
			cmd := exec.CommandContext(t.Context(), "git", "-C", root, "init", "-q")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v\n%s", err, out)
			}
			path := filepath.Join(root, ".git", "pulsarules.yaml")
			if err := os.WriteFile(path, []byte(private), disk.ModeFile); err != nil {
				t.Fatalf("write private config: %v", err)
			}
			cfg, err := (&Command{preset: testCase.preset}).governanceConfig(testLoader(), root)
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if len(cfg.Overrides) != testCase.wantOverrides {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverrides)
			}
		})
	}
}

func TestParseScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		raw     string
		want    governance.Scope
		wantErr string
	}{
		{name: "empty is the full scope", raw: "", want: governance.ScopeFull},
		{name: "commit", raw: "commit", want: governance.ScopeCommit},
		{
			name: "unknown names the flag", raw: "bogus", want: governance.ScopeFull,
			wantErr: `invalid --scope "bogus" (want full|commit|changed)`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseScope(testCase.raw)
			if got != testCase.want {
				t.Errorf("parseScope(%q) = %v, want %v", testCase.raw, got, testCase.want)
			}
			if (err == nil) != (testCase.wantErr == "") ||
				err != nil && err.Error() != testCase.wantErr {
				t.Fatalf("parseScope(%q) error = %v, want %q", testCase.raw, err, testCase.wantErr)
			}
		})
	}
}

func TestRunGovernance_CatalogFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init", "-q").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	open := func() (*vcs.Repository, error) { return repo, nil }
	catalog := func() (*emoji.Catalog, error) { return nil, errors.New("boom") }
	newLayered := func() *loader.Layered { return testLoader() }

	err = (&Command{project: dir}).Run(open, &knowledge.Index{}, catalog, newLayered)
	if err == nil || err.Error() != "get emoji catalog: boom" {
		t.Fatalf("Run() error = %v, want %q", err, "get emoji catalog: boom")
	}
}

func TestRunGovernance_RepositoryFailure(t *testing.T) {
	t.Parallel()

	err := (&Command{project: t.TempDir()}).Run(unopenableRepository(), nil, nil, nil)
	if err == nil || err.Error() != "open repository: boom" {
		t.Fatalf("Run() error = %v, want %q", err, "open repository: boom")
	}
}

func testLoader() *loader.Layered {
	return loader.New(registry.ConfigRules(), "")
}

func unopenableRepository() func() (*vcs.Repository, error) {
	return func() (*vcs.Repository, error) { return nil, errors.New("boom") }
}

func TestGovernanceConfig_UsesTheGivenLoader(t *testing.T) {
	t.Parallel()

	const unlistedAnalyzer = "version: 1\nanalyzers: {not-an-analyzer: {enabled: false}}\n"
	testCases := []struct {
		name    string
		loader  *loader.Layered
		wantErr bool
	}{
		{"registry rules reject it", testLoader(), true},
		{"a loader built for any analyzer accepts it", anyAnalyzerLoader(), false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := writeProjectConfig(t, unlistedAnalyzer)
			_, err := (&Command{}).governanceConfig(testCase.loader, root)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("governanceConfig error = %v, want error %v", err, testCase.wantErr)
			}
		})
	}
}

func anyAnalyzerLoader() *loader.Layered {
	return loader.New(projectfile.Rules{AnyAnalyzer: true}, "")
}
