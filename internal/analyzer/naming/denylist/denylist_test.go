package denylist

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestMain(m *testing.M) {
	if tssyntax.RunChild(os.Args, os.Exit) {
		return
	}
	os.Exit(m.Run())
}

type stubProvider struct{ files map[string]*syntax.File }

func (stub stubProvider) Languages() []language.ID { return []language.ID{language.Dart} }

func (stub stubProvider) Summarize(
	_ context.Context, _ syntax.Corpus,
) (map[string]*syntax.File, error) {
	return stub.files, nil
}

func contextFor(
	sources map[string]string, stubs map[string]*syntax.File, params map[string]any,
) *pass.Context {
	size := len(sources) + len(stubs)
	changed, corpus := make([]pass.FileChange, 0, size), make([]syntax.Source, 0, size)
	contents := map[string][]byte{}
	for name, content := range sources {
		contents[name] = []byte(content)
		changed = append(changed, pass.FileChange{Path: name, Extension: ".ts"})
		corpus = append(corpus, syntax.Source{Path: name})
	}
	for name := range stubs {
		changed = append(changed, pass.FileChange{Path: name, Extension: ".dart"})
		corpus = append(corpus, syntax.Source{Path: name})
	}
	cache := syntax.NewCache(
		[]syntax.Provider{tssyntax.New(), stubProvider{files: stubs}},
		syntax.Corpus{
			Sources: corpus,
			Read:    func(name string) ([]byte, bool) { return contents[name], true },
		},
	)
	return &pass.Context{
		ChangedFiles: changed,
		Syntax:       cache,
		Config: &pass.AnalysisConfig{
			Analyzers: map[string]pass.AnalyzerConfig{ID: {Enabled: true, Params: params}},
		},
	}
}

func TestAnalyze_TypeScriptSources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []int
	}{
		{"declaration is reported", "const draft = 1;\n", []int{1}},
		{"regex after paren", "f(/Draft saved<\\/option>/);\n", nil},
		{"regex after return", "function pick() {\n  return /^Legacy$/;\n}\n", nil},
		{"regex after comma", "f(1, /Legacy rules/);\n", nil},
		{"anchored regex is not an identifier", "const rule = /^Legacy$/;\n", nil},
		{"division then declaration", "const half = total / 2 / 4;\nconst draft = 1;\n", []int{2}},
		{"nested template", "t = `a ${ok ? `legacy ${v}` : 'no'}`;\nconst draft = 1;\n", []int{2}},
		{"allowed name passes", "const draftMode = 1;\n", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params := map[string]any{
				"tokens":        []any{"draft", "legacy", "without", "expired"},
				"allowed_names": []any{"draftMode"},
			}
			ctx := contextFor(map[string]string{"src/view.ts": testCase.source}, nil, params)
			var got []int
			for _, hit := range NewAnalyzer().Analyze(ctx) {
				if hit.Line > 0 {
					got = append(got, hit.Line)
				}
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("lines = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAnalyze_DartNames(t *testing.T) {
	t.Parallel()

	stub := &syntax.File{Names: []syntax.Name{
		{Name: "historyWithoutReload", Kind: syntax.NameFunction, Line: 3},
		{Name: "probeSessionExpired", Kind: syntax.NameFunction, Line: 7},
		{Name: "ignored", Kind: syntax.NameVariable, Line: 9},
		{Name: "legacyKept", Kind: syntax.NameVariable, Exempt: true},
	}}
	params := map[string]any{"tokens": []any{"without", "expired", "legacy"}}
	ctx := contextFor(nil, map[string]*syntax.File{"lib/view.dart": stub}, params)
	hits := NewAnalyzer().Analyze(ctx)
	got := make([]string, 0, len(hits))
	for _, hit := range hits {
		got = append(got, hit.Message)
	}
	want := []string{
		`function "historyWithoutReload" contains forbidden word "without"`,
		`function "probeSessionExpired" contains forbidden word "expired"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("messages = %q, want %q", got, want)
	}
}

func TestAnalyze_Inert(t *testing.T) {
	t.Parallel()

	stub := &syntax.File{Names: []syntax.Name{{Name: "draft", Kind: syntax.NameVariable, Line: 1}}}
	testCases := []struct {
		name   string
		params map[string]any
	}{{"no params", nil}, {"empty tokens", map[string]any{"tokens": []any{}}}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stubs := map[string]*syntax.File{"draft/draft.dart": stub}
			ctx := contextFor(nil, stubs, testCase.params)
			if got := NewAnalyzer().Analyze(ctx); len(got) != 0 {
				t.Fatalf("findings = %v, want none while tokens is empty", got)
			}
		})
	}
}

func TestAnalyze_GeneratedIsSkipped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		generated bool
		want      int
	}{
		{"generated file is skipped", true, 0},
		{"hand-written file is reported", false, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stub := &syntax.File{Names: []syntax.Name{{Name: "draft", Line: 1}}}
			params := map[string]any{"tokens": []any{"draft"}}
			ctx := contextFor(nil, map[string]*syntax.File{"lib/view.dart": stub}, params)
			ctx.ChangedFiles[0].Generated = testCase.generated
			if got := NewAnalyzer().Analyze(ctx); len(got) != testCase.want {
				t.Fatalf("findings = %v, want %d", got, testCase.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	want := pass.Descriptor{ID: "forbidden-token", Rule: "naming"}
	want.Params = []string{"tokens", "allowed_names"}
	if got := Describe(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Describe() = %+v, want %+v", got, want)
	}
}
