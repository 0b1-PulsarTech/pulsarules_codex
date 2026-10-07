package gosyntax

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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

func TestProviderIdentity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want bool
	}{
		{ext: ".go", want: true},
		{ext: ".GO", want: true},
		{ext: ".dart"},
		{ext: ""},
	}
	provider := New(nil)
	if provider.Language() != syntax.LangGo {
		t.Fatalf("Language() = %q", provider.Language())
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()

			if got := provider.Handles(testCase.ext); got != testCase.want {
				t.Fatalf("Handles(%q) = %v", testCase.ext, got)
			}
		})
	}
}

type summarizeCase struct {
	name        string
	path        string
	body        string
	noModule    bool
	unread      bool
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
			wantParse: "file has no syntax tree",
		},
		{
			name: "unreadable source becomes a parse error", path: "gone.go", unread: true,
			wantParse: "file is not readable",
		},
		{name: "cancelled context stops early", path: "ok.go", cancelled: true, wantErr: true},
	}
}

func TestProviderSummarize(t *testing.T) {
	t.Parallel()

	moduleRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/app\n"), 0o600,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, testCase := range summarizeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := moduleRoot
			if testCase.noModule {
				root = t.TempDir()
			}
			got, err := runSummarize(root, testCase)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if !testCase.wantErr {
				assertSummary(t, got[testCase.path], testCase)
			}
		})
	}
}

func runSummarize(root string, testCase summarizeCase) (map[string]*syntax.File, error) {
	cache := astcache.New()
	if testCase.body != "" {
		_, _ = cache.Parse(testCase.path, []byte(testCase.body))
	}
	read := func(string) ([]byte, bool) { return []byte(testCase.body), !testCase.unread }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if testCase.cancelled {
		cancel()
	}
	return New(cache).Summarize(ctx, syntax.Corpus{
		Root:    root,
		Sources: []syntax.Source{{Path: testCase.path}},
		Read:    read,
	})
}

func assertSummary(t *testing.T, file *syntax.File, testCase summarizeCase) {
	t.Helper()

	if file == nil || file.Language != syntax.LangGo {
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

	testCases := []struct {
		name   string
		unread bool
		want   string
	}{
		{name: "readable but absent from the cache", want: "file has no syntax tree"},
		{name: "unreadable", unread: true, want: "file is not readable"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			read := func(string) ([]byte, bool) { return []byte("package x\n"), !testCase.unread }
			got := New(cache).parseError("fresh.go", read)
			if got != testCase.want {
				t.Fatalf("parseError = %q, want %q", got, testCase.want)
			}
			if cache.Get("fresh.go") != nil {
				t.Fatal("parseError must not parse into the shared cache")
			}
		})
	}
}
