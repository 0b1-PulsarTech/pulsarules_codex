package controlflow

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type stubProvider struct{ files map[string]*syntax.File }

func (stubProvider) Languages() []language.ID { return []language.ID{language.Dart} }

func (provider stubProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	out := map[string]*syntax.File{}
	for _, src := range corpus.Sources {
		out[src.Path] = provider.files[src.Path]
	}
	return out, nil
}

func stubContext(file *syntax.File, params map[string]any) *pass.Context {
	read := func(string) ([]byte, bool) { return []byte("x"), true }
	return &pass.Context{
		ChangedFiles: []pass.FileChange{{Path: "a.dart", Extension: ".dart"}},
		Config: &pass.AnalysisConfig{
			Analyzers: map[string]pass.AnalyzerConfig{ID: {Enabled: true, Params: params}},
		},
		Syntax: syntax.NewCache(
			[]syntax.Provider{stubProvider{files: map[string]*syntax.File{"a.dart": file}}},
			syntax.Corpus{Sources: []syntax.Source{{Path: "a.dart"}}, Read: read},
		),
	}
}

func TestAnalyze_SyntaxElses(t *testing.T) {
	t.Parallel()

	const afterExitCase = "else after a branch that already returns, throws, breaks or " +
		"continues; drop the else"
	plain := []syntax.Else{{Line: 3, AfterExit: true}, {Line: 7}, {Line: 9, AfterExit: true}}
	testCases := []struct {
		name   string
		elses  []syntax.Else
		params map[string]any
		want   []string
	}{
		{
			name:  "default flags only an else after an exit",
			elses: plain,
			want:  []string{"3:" + afterExitCase, "9:" + afterExitCase},
		},
		{
			name:   "forbid_else flags every else",
			elses:  plain,
			params: map[string]any{paramForbidElse: true},
			want: []string{
				"3:" + forbidElseMessage,
				"7:" + forbidElseMessage,
				"9:" + forbidElseMessage,
			},
		},
		{
			name:   "forbid_else off keeps the default",
			elses:  []syntax.Else{{Line: 7}},
			params: map[string]any{paramForbidElse: false},
		},
		{
			name:   "a collection element else only with forbid_else",
			elses:  []syntax.Else{{Line: 4}},
			params: map[string]any{paramForbidElse: true},
			want:   []string{"4:" + forbidElseMessage},
		},
		{name: "a collection element else is silent by default", elses: []syntax.Else{{Line: 4}}},
		{name: "no else", params: map[string]any{paramForbidElse: true}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := &syntax.File{Path: "a.dart", Elses: testCase.elses}
			findings := NewAnalyzer().Analyze(stubContext(file, testCase.params))
			got := make([]string, 0, len(findings))
			for _, hit := range findings {
				if hit.AnalyzerID != ID+"/else" || hit.Severity != finding.SeverityWarning {
					t.Fatalf("finding = %+v, want id control-flow/else at warning", hit)
				}
				got = append(got, fmt.Sprintf("%d:%s", hit.Line, hit.Message))
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("findings = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAnalyze_NestingKeepsItsOwnID(t *testing.T) {
	t.Parallel()

	file := &syntax.File{
		Path:   "a.dart",
		Blocks: []syntax.Block{{Line: 2, Depth: 5}},
		Elses:  []syntax.Else{{Line: 3, AfterExit: true}},
	}
	ids := map[int]string{}
	for _, hit := range NewAnalyzer().Analyze(stubContext(file, nil)) {
		ids[hit.Line] = hit.AnalyzerID
	}
	if ids[2] != ID || ids[3] != ID+"/else" || len(ids) != 2 {
		t.Fatalf("ids by line = %v, want nesting on %q and else on %q", ids, ID, ID+"/else")
	}
}
