package singleletter

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

type testCase struct {
	name   string
	source string
	want   []string
}

func defaultSettings() settings {
	return settings{reporter: NewAnalyzer().reporter}
}

func summaries(cache *astcache.Cache) *syntax.Cache {
	read := func(string) ([]byte, bool) { return nil, true }
	return syntax.NewCache(
		[]syntax.Provider{gosyntax.New(cache)},
		syntax.Corpus{
			Sources: []syntax.Source{{Path: "pkg.go"}}, Read: read,
			Module: func() string { return "" },
		},
	)
}

func summarize(t *testing.T, source string) *syntax.File {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("pkg.go", []byte(source)); err != nil {
		t.Fatalf("parse: %v", err)
	}
	file, ok := summaries(cache).Get("pkg.go")
	if !ok {
		t.Fatal("no summary")
	}
	return file
}

func assertMessages(t *testing.T, source string, cfg settings, want []string) {
	t.Helper()

	fc := pass.FileChange{Path: "pkg.go", Extension: ".go"}
	got := checkFile(fc, summarize(t, source), cfg)
	if len(got) != len(want) {
		t.Fatalf("findings = %+v, want %q", got, want)
	}
	for pos, finding := range got {
		if finding.Message != want[pos] {
			t.Fatalf("message = %q, want %q", finding.Message, want[pos])
		}
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		change pass.FileChange
		expect int
	}{
		{"regular file", pass.FileChange{Path: "pkg.go", Extension: ".go"}, 1},
		{
			"generated file is skipped",
			pass.FileChange{Path: "pkg.go", Extension: ".go", Generated: true},
			0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("pkg.go", []byte("package pkg\n\nvar x int\n")); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &pass.Context{
				ASTCache:     cache,
				Syntax:       summaries(cache),
				ChangedFiles: []pass.FileChange{testCase.change},
			}
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	if got := analyzer.Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings without an AST cache = %+v", got)
	}
}

func TestFindingHint(t *testing.T) {
	t.Parallel()

	fc := pass.FileChange{Path: "pkg.go", Extension: ".go"}
	got := checkFile(fc, summarize(t, "package pkg\n\nvar x int\n"), defaultSettings())
	if len(got) != 1 || got[0].Line != 3 ||
		!strings.Contains(got[0].Suggestion, "what it is or holds; a short abbreviation") {
		t.Fatalf("findings = %+v", got)
	}
}

func allowedMessages(t *testing.T, params map[string]any) []string {
	t.Helper()

	const source = "package pkg\n\nconst X = 1\n\ntype T struct{}\n\nvar t = 3\n\n" +
		"var y = 2\n\nfunc run(p int) { q := p; _ = q }\n"
	cache := astcache.New()
	if _, err := cache.Parse("pkg.go", []byte(source)); err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := &pass.Context{
		ASTCache:     cache,
		Syntax:       summaries(cache),
		ChangedFiles: []pass.FileChange{{Path: "pkg.go", Extension: ".go"}},
		Config: &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
			"single-letter-name": {Enabled: true, Params: params},
		}},
	}
	findings := NewAnalyzer().Analyze(ctx)
	got := make([]string, 0, len(findings))
	for _, finding := range findings {
		got = append(got, finding.Message)
	}
	return got
}

func TestAnalyzeAllowedNamesAndExportedOnly(t *testing.T) {
	t.Parallel()

	const (
		constX = `constant "X" is a single letter`
		typeT  = `type "T" is a single letter`
		varT   = `variable "t" is a single letter`
		varY   = `variable "y" is a single letter`
		paramP = `parameter "p" is a single letter`
		varQ   = `variable "q" is a single letter`
	)
	every := []string{constX, typeT, varT, varY, paramP, varQ}
	allow := func(names ...string) map[string]any {
		return map[string]any{"allowed_names": names}
	}
	testCases := []struct {
		name   string
		params map[string]any
		want   []string
	}{
		{"default flags every single letter", nil, every},
		{"explicit false flags every", map[string]any{"exported_only": false}, every},
		{"allowed t spares t but not T", allow("t"), []string{constX, typeT, varY, paramP, varQ}},
		{"allowed names spares exact matches", allow("y", "p"), []string{
			constX, typeT, varT, varQ,
		}},
		{"allowed names is case sensitive", allow("x"), every},
		{"allowed names ignores non string entries", map[string]any{
			"allowed_names": []any{1, "y"},
		}, []string{constX, typeT, varT, paramP, varQ}},
		{"exported only keeps upper case", map[string]any{"exported_only": true}, []string{
			constX, typeT,
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := allowedMessages(t, testCase.params); !slices.Equal(got, testCase.want) {
				t.Fatalf("messages = %q, want %q", got, testCase.want)
			}
		})
	}
}
