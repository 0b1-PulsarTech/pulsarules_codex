package golangcilint

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestExtractGolangciConfigPath(t *testing.T) {
	t.Parallel()

	withParams := func(params map[string]any) *pass.Context {
		return &pass.Context{Config: &pass.AnalysisConfig{
			Analyzers: map[string]pass.AnalyzerConfig{"golangci-lint": {Params: params}},
		}}
	}

	testCases := []struct {
		name string
		ctx  *pass.Context
		want string
	}{
		{
			name: "configured path wins",
			ctx:  withParams(map[string]any{"config_path": "build/.golangci.yml"}),
			want: "build/.golangci.yml",
		},
		{
			name: "analyzer absent from the config",
			ctx: &pass.Context{Config: &pass.AnalysisConfig{
				Analyzers: map[string]pass.AnalyzerConfig{},
			}},
			want: "",
		},
		{name: "param absent", ctx: withParams(map[string]any{}), want: ""},
		{name: "nil config does not panic", ctx: &pass.Context{}, want: ""},
		{name: "nil params", ctx: withParams(nil), want: ""},
		{
			name: "wrong type degrades to empty rather than panicking",
			ctx:  withParams(map[string]any{"config_path": 42}),
			want: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := extractGolangciConfigPath(testCase.ctx); got != testCase.want {
				t.Errorf("extractGolangciConfigPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAnalyzer_StageRegistryKey(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer("golangci-lint")
	if got := a.ID(); got != "golangci-lint" {
		t.Errorf("ID() = %q, want %q", got, "golangci-lint")
	}
	if got := a.Stage(); got != pass.StageStatic {
		t.Errorf("Stage() = %v, want StageStatic", got)
	}
	if req := a.Needs(); req.NeedsAST || req.NeedsGitHistory {
		t.Errorf("Needs() = %+v, want no requirements", req)
	}
}

func TestAnalyzer_SkipsWithoutBinary(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer("/nonexistent/golangci-lint")
	findings := a.Analyze(&pass.Context{
		ProjectDir: t.TempDir(),
		Config:     &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{}},
	})
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	if findings[0].AnalyzerID != "golangci-lint" {
		t.Errorf("AnalyzerID = %q, want %q", findings[0].AnalyzerID, "golangci-lint")
	}
}
