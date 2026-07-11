package delegation

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestExtractGolangciConfigPath(t *testing.T) {
	t.Parallel()

	withParams := func(params map[string]any) *core.AnalysisContext {
		return &core.AnalysisContext{Config: &core.AnalysisConfig{
			Analyzers: map[string]core.AnalyzerConfig{"golangci-lint": {Params: params}},
		}}
	}

	testCases := []struct {
		name string
		ctx  *core.AnalysisContext
		want string
	}{
		{
			name: "configured path wins",
			ctx:  withParams(map[string]any{"config_path": "build/.golangci.yml"}),
			want: "build/.golangci.yml",
		},
		{
			name: "analyzer absent from the config",
			ctx: &core.AnalysisContext{Config: &core.AnalysisConfig{
				Analyzers: map[string]core.AnalyzerConfig{},
			}},
			want: "",
		},
		{name: "param absent", ctx: withParams(map[string]any{}), want: ""},
		{name: "nil config does not panic", ctx: &core.AnalysisContext{}, want: ""},
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

// TestGolangcilintAnalyzer_Contract pins the identity the registry in
// analysis/specs.go keys on; a silent change there would drop the analyzer
// from its stage without failing anything else.
func TestGolangcilintAnalyzer_Contract(t *testing.T) {
	t.Parallel()

	a := NewGolangcilintAnalyzer("golangci-lint")
	if got := a.ID(); got != "golangci-lint" {
		t.Errorf("ID() = %q, want %q", got, "golangci-lint")
	}
	if got := a.Stage(); got != core.StageStatic {
		t.Errorf("Stage() = %v, want StageStatic", got)
	}
	if got := a.Category(); got != core.CatSyntax {
		t.Errorf("Category() = %v, want CatSyntax", got)
	}
	if req := a.Needs(); req.NeedsAST || req.NeedsGitHistory {
		t.Errorf("Needs() = %+v, want no requirements", req)
	}
}

// TestGolangcilintAnalyzer_AnalyzeWithoutBinary proves the adapter reports the
// failure instead of returning nothing when the binary cannot be executed.
func TestGolangcilintAnalyzer_AnalyzeWithoutBinary(t *testing.T) {
	t.Parallel()

	a := NewGolangcilintAnalyzer("/nonexistent/golangci-lint")
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: t.TempDir(),
		Config:     &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{}},
	})
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	if findings[0].AnalyzerID != "golangci-lint" {
		t.Errorf("AnalyzerID = %q, want %q", findings[0].AnalyzerID, "golangci-lint")
	}
}
