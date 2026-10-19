package pass

import (
	"errors"
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func newChangedGoASTsCache(t *testing.T) *astcache.Cache {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("main.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse main.go: %v", err)
	}
	if _, err := cache.Parse("helper_test.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse helper_test.go: %v", err)
	}
	return cache
}

func TestAnalysisContext_ChangedGoASTs_NoCache(t *testing.T) {
	t.Parallel()

	ctx := &Context{
		ChangedFiles: []FileChange{{Path: "main.go", Extension: ".go"}},
	}

	for range ctx.ChangedGoASTs() {
		t.Fatal("expected no yields when ASTCache is nil")
	}
}

func TestAnalysisContext_ChangedGoASTs_Skips(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)

	testCases := []struct {
		name  string
		fc    FileChange
		parse bool
	}{
		{
			name: "non-go file skipped",
			fc:   FileChange{Path: "README.md", Extension: ".md"},
		},
		{
			name: "path absent from cache skipped",
			fc:   FileChange{Path: "missing.go", Extension: ".go"},
		},
		{
			name:  "testdata dir skipped",
			fc:    FileChange{Path: "pkg/testdata/x.go", Extension: ".go"},
			parse: true,
		},
		{
			name:  "vendor dir skipped",
			fc:    FileChange{Path: "vendor/mod/y.go", Extension: ".go"},
			parse: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := cache
			if testCase.parse {
				cache = newChangedGoASTsCache(t)
				if _, err := cache.Parse(testCase.fc.Path, []byte("package x\n")); err != nil {
					t.Fatalf("parse %s: %v", testCase.fc.Path, err)
				}
			}
			ctx := &Context{
				ASTCache:     cache,
				Skips:        source.NewSkipSet([]string{"testdata", "vendor"}),
				ChangedFiles: []FileChange{testCase.fc},
			}
			for range ctx.ChangedGoASTs() {
				t.Fatalf("expected %s to be skipped", testCase.fc.Path)
			}
		})
	}
}

func TestChangedGoASTs_TestFiles(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)
	ctx := &Context{
		ASTCache: cache,
		ChangedFiles: []FileChange{
			{Path: "README.md", Extension: ".md"},
			{Path: "helper_test.go", Extension: ".go", IsTest: true},
			{Path: "missing.go", Extension: ".go"},
			{Path: "main.go", Extension: ".go"},
		},
	}

	var got []FileChange
	for fc, file := range ctx.ChangedGoASTs() {
		if file == nil {
			t.Fatalf("expected a non-nil *ast.File for %s", fc.Path)
		}
		got = append(got, fc)
	}
	if len(got) != 2 || got[0].Path != "helper_test.go" || got[1].Path != "main.go" {
		t.Fatalf("got %v, want helper_test.go and main.go", got)
	}
}

func TestChangedGoASTs_CallerBreak(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)
	if _, err := cache.Parse("second.go", []byte("package main\n")); err != nil {
		t.Fatalf("parse second.go: %v", err)
	}
	ctx := &Context{
		ASTCache: cache,
		ChangedFiles: []FileChange{
			{Path: "main.go", Extension: ".go"},
			{Path: "second.go", Extension: ".go"},
		},
	}

	count := 0
	for range ctx.ChangedGoASTs() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestRunPerGoFile_NoCache(t *testing.T) {
	t.Parallel()

	ctx := &Context{
		ChangedFiles: []FileChange{{Path: "main.go", Extension: ".go"}},
	}
	called := false
	got := RunPerGoFile(ctx, func(FileChange, *ast.File) []finding.Finding {
		called = true
		return nil
	})
	if called {
		t.Fatal("expected check not to be called when ASTCache is nil")
	}
	if got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestRunPerGoFile_ProductionAndTests(t *testing.T) {
	t.Parallel()

	cache := newChangedGoASTsCache(t)
	ctx := &Context{
		ASTCache: cache,
		ChangedFiles: []FileChange{
			{Path: "main.go", Extension: ".go"},
			{Path: "helper_test.go", Extension: ".go", IsTest: true},
		},
	}

	got := RunPerGoFile(ctx, func(fc FileChange, file *ast.File) []finding.Finding {
		if file == nil {
			t.Fatalf("expected a non-nil *ast.File for %s", fc.Path)
		}
		return []finding.Finding{{AnalyzerID: "fake", File: fc.Path}}
	})
	if len(got) != 2 || got[0].File != "main.go" || got[1].File != "helper_test.go" {
		t.Fatalf("got %v, want findings for main.go and helper_test.go", got)
	}
}

type readCounter struct {
	source.Provider
	reads int
}

func (counter *readCounter) Read(path string) ([]byte, bool) {
	counter.reads++
	return counter.Provider.Read(path)
}

func TestAnalysisContext_GoPackages(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{"a/a.go": {Data: []byte("package a\n")}}
	counter := &readCounter{Provider: source.NewFSProvider(tree, "", source.NewSkipSet(nil))}
	ctx := &Context{Sources: counter}

	first := ctx.GoPackages()
	if second := ctx.GoPackages(); second != first {
		t.Error("GoPackages() built a second index in one run")
	}
	if counter.reads != 0 {
		t.Errorf("source read %d times across two GoPackages() calls, want 0", counter.reads)
	}
	if files := first.Files("a"); len(files) != 1 || counter.reads != 1 {
		t.Errorf("Files(a) = %d files after %d reads, want 1 and 1", len(files), counter.reads)
	}
}

func TestAnalysisContext_GoPackages_NoSources(t *testing.T) {
	t.Parallel()

	if dirs := (&Context{}).GoPackages().Dirs(); len(dirs) != 0 {
		t.Errorf("Dirs() = %v, want none", dirs)
	}
}

func TestAnalysisContext_GoModule(t *testing.T) {
	t.Parallel()

	withModule := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(withModule, "go.mod"), []byte("module example.com/m\n"), 0o600,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	testCases := []struct {
		name     string
		root     string
		wantPath string
		wantErr  error
	}{
		{name: "no root", root: ""},
		{name: "module", root: withModule, wantPath: "example.com/m"},
		{name: "no go.mod", root: t.TempDir(), wantErr: fs.ErrNotExist},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := (&Context{ProjectDir: testCase.root}).GoModule()
			if got.Path != testCase.wantPath || !errors.Is(got.Err, testCase.wantErr) {
				t.Errorf("GoModule() = %+v, want path %q err %v",
					got, testCase.wantPath, testCase.wantErr)
			}
		})
	}
}

func TestAnalysisContext_GoModule_ReadsOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ctx := &Context{ProjectDir: root}
	if err := os.WriteFile(
		filepath.Join(root, "go.mod"), []byte("module example.com/first\n"), 0o600,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	first := ctx.GoModule()
	if err := os.WriteFile(
		filepath.Join(root, "go.mod"), []byte("module example.com/second\n"), 0o600,
	); err != nil {
		t.Fatalf("rewrite go.mod: %v", err)
	}
	if got := ctx.GoModule(); got != first || got.Path != "example.com/first" {
		t.Errorf("GoModule() = %+v after a rewrite, want the memoised %+v", got, first)
	}
}
