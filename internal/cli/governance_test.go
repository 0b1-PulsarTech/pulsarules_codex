package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func TestRunGovernance_RequiresProjectDir(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "")

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := runGovernance(inj, &version.Options{})
	if !errors.Is(err, errProjectDirRequired) {
		t.Errorf("err = %v, want %v", err, errProjectDirRequired)
	}
}

func TestGovernanceConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *version.Options
		wantPreset string
		wantPath   any
	}{
		{
			name:       "no flags keeps the default preset",
			opts:       &version.Options{},
			wantPreset: config.PresetRecommended,
		},
		{
			name:       "preset flag wins",
			opts:       &version.Options{Preset: config.PresetStrict},
			wantPreset: config.PresetStrict,
		},
		{
			name:       "golangci config lands in the analyzer params",
			opts:       &version.Options{GolangciConfig: "build/.golangci.yml"},
			wantPreset: config.PresetRecommended,
			wantPath:   "build/.golangci.yml",
		},
		{
			name:       "preset and config path compose",
			opts:       &version.Options{Preset: config.PresetMinimal, GolangciConfig: "x.yml"},
			wantPreset: config.PresetMinimal,
			wantPath:   "x.yml",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(testCase.opts, "")
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

	_, err := governanceConfig(&version.Options{Preset: "stict"}, "")
	if err == nil {
		t.Fatal("governanceConfig: expected an error for an unknown preset")
	}
	presets := []string{config.PresetRecommended, config.PresetStrict, config.PresetMinimal}
	for _, want := range presets {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name valid preset %q", err, want)
		}
	}
}

func TestGovernanceConfig_IncludeGenerated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts *version.Options
		want bool
	}{
		{name: "suppressed by default", opts: &version.Options{}, want: false},
		{
			name: "flag turns the filter off",
			opts: &version.Options{IncludeGenerated: true},
			want: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(testCase.opts, "")
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.IncludeGenerated; got != testCase.want {
				t.Errorf("IncludeGenerated = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGovernanceConfig_StrictLowersFileSize(t *testing.T) {
	t.Parallel()

	cfg, err := governanceConfig(&version.Options{Preset: config.PresetStrict}, "")
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	if got := cfg.Analyzers["file-size"].Params["max_lines"]; got != 180 {
		t.Errorf("strict max_lines = %v, want 180", got)
	}
	if got := cfg.Analyzers["complexity"].Params["max_complexity"]; got != 10 {
		t.Errorf("strict max_complexity = %v, want 10", got)
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
			name: "a list reaches the analyzer", spec: "spike,experiment",
			wantSet: "spike,experiment",
		},
		{name: "uppercase names the flag and the form", spec: "Release", wantErr: wantMessage},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(&version.Options{BranchExtraTypes: testCase.spec}, "")
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
	if err := os.WriteFile(path, []byte(content), disk.File); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	return dir
}

type precedenceCase struct {
	name         string
	opts         *version.Options
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
		{"file only", &version.Options{}, config.PresetMinimal, "file.yml", "error"},
		{
			"preset flag beats the file",
			&version.Options{Preset: config.PresetStrict},
			config.PresetStrict, "file.yml", "error",
		},
		{
			"param flag beats the file",
			&version.Options{GolangciConfig: "flag.yml"},
			config.PresetMinimal, "flag.yml", "error",
		},
		{
			"severity flag beats the file",
			&version.Options{TypographicSeverity: "warning"},
			config.PresetMinimal, "file.yml", "warning",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := governanceConfig(testCase.opts, root)
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
			_, err := governanceConfig(&version.Options{}, root)
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

	cfg, err := governanceConfig(&version.Options{Config: explicit}, "")
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	if got := cfg.Analyzers["big-comment"].Params["max_lines"]; got != 2 {
		t.Errorf("max_lines = %v, want 2", got)
	}
	missing := &version.Options{Config: filepath.Join(dir, "nope.yaml")}
	if _, err = governanceConfig(missing, dir); err == nil {
		t.Error("governanceConfig with a missing --config: expected an error")
	}
}

func TestGovernanceConfig_NoProjectFileIsUnchanged(t *testing.T) {
	t.Parallel()

	cfg, err := governanceConfig(&version.Options{}, t.TempDir())
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
			if err := os.WriteFile(path, []byte(private), disk.File); err != nil {
				t.Fatalf("write private config: %v", err)
			}
			cfg, err := governanceConfig(&version.Options{Preset: testCase.preset}, root)
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if len(cfg.Overrides) != testCase.wantOverrides {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverrides)
			}
		})
	}
}
