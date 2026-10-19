package gosyntax

import (
	"context"
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func parseSource(t *testing.T, content string) (*astcache.Cache, *ast.File) {
	t.Helper()

	cache := astcache.New()
	file, err := cache.Parse("pkg.go", []byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return cache, file
}

func summarizeSource(
	t *testing.T, content string, source syntax.Source, modulePath string,
) *syntax.File {
	t.Helper()

	cache, file := parseSource(t, content)
	return summarizeFile(cache.FileSet(), source, file, modulePath)
}

func plainSource() syntax.Source { return syntax.Source{Path: "pkg.go"} }

func TestProviderLanguages(t *testing.T) {
	t.Parallel()

	got := New(nil).Languages()
	if len(got) != 1 || got[0] != language.Go {
		t.Fatalf("Languages() = %v, want [go]", got)
	}
}

type summarizeCase struct {
	name        string
	path        string
	body        string
	noModule    bool
	cancelled   bool
	wantErr     bool
	wantParse   string
	wantFuncs   int
	wantImports int
	wantGroups  int
}

func summarizeCases() []summarizeCase {
	return []summarizeCase{
		{
			name: "parsed file is summarised", path: "ok.go",
			body:      "package x\n\nimport \"fmt\"\n\nfunc run() { fmt.Println() }\n",
			wantFuncs: 1, wantImports: 1, wantGroups: 3,
		},
		{
			name: "missing go.mod leaves import data empty", path: "ok.go", noModule: true,
			body: "package x\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint\n",
		},
		{
			name: "syntax error becomes a parse error", path: "bad.go", body: "package\n",
			wantParse: "parse bad.go: bad.go:1:9: expected 'IDENT', found 'EOF'",
		},
		{name: "cancelled context stops early", path: "ok.go", cancelled: true, wantErr: true},
	}
}

func TestProviderSummarize(t *testing.T) {
	t.Parallel()

	for _, testCase := range summarizeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := runSummarize(testCase)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if !testCase.wantErr {
				assertSummary(t, got[testCase.path], testCase)
			}
		})
	}
}

func runSummarize(testCase summarizeCase) (map[string]*syntax.File, error) {
	cache := astcache.New()
	if testCase.body != "" {
		_, _ = cache.Parse(testCase.path, []byte(testCase.body))
	}
	module := "example.com/app"
	if testCase.noModule {
		module = ""
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if testCase.cancelled {
		cancel()
	}
	return New(cache).Summarize(ctx, syntax.Corpus{
		Sources: []syntax.Source{{Path: testCase.path, Content: []byte(testCase.body)}},
		Module:  func() string { return module },
	})
}

func assertSummary(t *testing.T, file *syntax.File, testCase summarizeCase) {
	t.Helper()

	if file == nil || file.Language != language.Go {
		t.Fatalf("file = %+v", file)
	}
	if file.ParseError != testCase.wantParse {
		t.Fatalf("ParseError = %q, want %q", file.ParseError, testCase.wantParse)
	}
	if len(file.Functions) != testCase.wantFuncs ||
		len(file.Imports) != testCase.wantImports ||
		len(file.ImportGroups) != testCase.wantGroups {
		t.Fatalf("file = %+v", file)
	}
}

func TestProviderParseError(t *testing.T) {
	t.Parallel()

	cache := astcache.New()
	got := New(cache).parseError("fresh.go")
	if got != "file has no syntax tree" {
		t.Fatalf("parseError = %q", got)
	}
	if cache.Get("fresh.go") != nil {
		t.Fatal("parseError must not parse into the shared cache")
	}
}
