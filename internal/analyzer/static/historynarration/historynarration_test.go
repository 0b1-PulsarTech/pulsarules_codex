package historynarration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
)

type scanCase struct {
	name   string
	lang   string
	source string
	expect int
}

func runScan(t *testing.T, testCase scanCase) {
	t.Helper()
	source := testCase.source
	if testCase.lang == "go" {
		source = "package p\n\nfunc f() {\n" + source + "}\n"
	}
	name, ext, found := strings.Cut(testCase.lang, ".")
	if !found {
		ext = name
	}
	syntax := langspec.Lookup(name).CommentSyntax()
	fc := core.FileChange{Path: "f", Extension: "." + ext}
	got := scanFile([]byte(source), fc, syntax, NewAnalyzer(nil).reporter(), parsedGo{})
	if len(got) != testCase.expect {
		t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
	}
}

func TestScanFileUnparsableGo(t *testing.T) {
	t.Parallel()

	syntax := langspec.Lookup("go").CommentSyntax()
	fc := core.FileChange{Path: "f.go", Extension: ".go"}
	got := scanFile(
		[]byte("// used to be a loop\nfunc ("),
		fc,
		syntax,
		NewAnalyzer(nil).reporter(),
		parsedGo{},
	)
	if len(got) != 1 || got[0].Line != 1 {
		t.Fatalf("findings = %+v, want one on line 1", got)
	}
}

func TestScanFileSplitPhraseLine(t *testing.T) {
	t.Parallel()

	syntax := langspec.Lookup("go").CommentSyntax()
	fc := core.FileChange{Path: "f.go", Extension: ".go"}
	source := "package p\n\n// the guard was\n// replaced by a check\n"
	got := scanFile([]byte(source), fc, syntax, NewAnalyzer(nil).reporter(), parsedGo{})
	if len(got) != 1 || got[0].Line != 3 {
		t.Fatalf("findings = %+v, want one on line 3", got)
	}
}

func TestScanFileMessageNamesPhrase(t *testing.T) {
	t.Parallel()

	syntax := langspec.Lookup("go").CommentSyntax()
	fc := core.FileChange{Path: "f.go", Extension: ".go"}
	source := "package p\n\n// we never ever used to retry\n"
	got := scanFile([]byte(source), fc, syntax, NewAnalyzer(nil).reporter(), parsedGo{})
	want := "comment narrates history: \"never ever used to retry\""
	if len(got) != 1 || got[0].Message != want {
		t.Fatalf("findings = %+v, want one with message %q", got, want)
	}
}

func TestAnalyzeUsesCachedGoAST(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "a.go"), []byte("package foo\n"), 0o644,
	); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cache := astcache.New()
	if _, err := cache.Parse("a.go", []byte("package foo\n\n// used to be a loop\n")); err != nil {
		t.Fatalf("parse: %v", err)
	}
	langs := core.NewLanguageRegistry()
	langs.Register(langspec.Lookup("go"))
	got := NewAnalyzer(langs).Analyze(&core.AnalysisContext{
		ProjectDir:   dir,
		ASTCache:     cache,
		Sources:      core.NewSourceProvider(dir, core.NewSkipSet(core.DefaultSkipDirs())),
		ChangedFiles: []core.FileChange{{Path: "a.go", Extension: ".go"}},
	})
	if len(got) != 1 || got[0].Line != 3 {
		t.Fatalf("findings = %+v, want one on line 3 taken from the cached AST", got)
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		withLang bool
		file     string
		expect   int
	}{
		{"go file", true, "a.go", 1},
		{"no language registry", false, "a.go", 0},
		{"unregistered extension", true, "a.xyz", 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			content := []byte("package foo\n\n// used to be a loop\nfunc bar() {}\n")
			if err := os.WriteFile(filepath.Join(dir, testCase.file), content, 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			var langs *core.LanguageRegistry
			if testCase.withLang {
				langs = core.NewLanguageRegistry()
				langs.Register(langspec.Lookup("go"))
			}
			got := NewAnalyzer(langs).Analyze(&core.AnalysisContext{
				ProjectDir: dir,
				Sources:    core.NewSourceProvider(dir, core.NewSkipSet(core.DefaultSkipDirs())),
				ChangedFiles: []core.FileChange{
					{Path: testCase.file, Extension: filepath.Ext(testCase.file)},
				},
			})
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.ID() != "history-narration" || analyzer.Stage() != core.StageStatic {
		t.Fatalf("id = %q stage = %v", analyzer.ID(), analyzer.Stage())
	}
}
