package simplification

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func testLangs() *pass.LanguageRegistry {
	langs := pass.NewLanguageRegistry()
	langs.Register(language.Lookup("go"))
	langs.Register(language.Lookup("clike"))
	return langs
}

type analyzeCase struct {
	name    string
	file    string
	content string
	wantLen int
}

func analyzeCases() []analyzeCase {
	return []analyzeCase{
		{
			"upgrade path label on the marker line", "same_line_upgrade_path.go",
			"package foo\n\n" +
				"// simplification: skip the rare case. " +
				"Upgrade path: handle it once seen twice.\n" +
				"func f() {}\n",
			0,
		},
		{
			"revisit when label on the marker line", "revisit_when.go",
			"package foo\n\n" +
				"// simplification: skip the rare case; revisit when it shows up twice.\n" +
				"func f() {}\n",
			0,
		},
		{
			"no label at all fires", "no_label.go",
			"package foo\n\n" +
				"// simplification: skip the rare case because it never happens in practice.\n" +
				"func f() {}\n",
			1,
		},
		{
			"upgrade path label on a later block line", "later_line.go",
			"package foo\n\n" +
				"// simplification: skip the rare case, which is ample signal for now.\n" +
				"// This keeps the function small.\n" +
				"// Upgrade path: handle it once the rare case is seen twice.\n" +
				"func f() {}\n",
			0,
		},
		{
			"no marker at all", "no_marker.go",
			"package foo\n\n// f does a thing.\nfunc f() {}\n",
			0,
		},
		{
			"word simplification in prose, not a marker", "prose_mention.go",
			"package foo\n\n" +
				"// f applies a simplification the caller already validated, nothing more.\n" +
				"func f() {}\n",
			0,
		},
		{
			"no label fires in a clike file too", "no_label.dart",
			"class Foo {\n" +
				"  // simplification: skip the rare case because it never happens in practice.\n" +
				"  void f() {}\n" +
				"}\n",
			1,
		},
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcs := source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs()))
	langs := testLangs()

	for _, testCase := range analyzeCases() {
		if err := os.WriteFile(
			filepath.Join(dir, testCase.file), []byte(testCase.content), 0o644,
		); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewAnalyzer(langs).Analyze(&pass.Context{
				ProjectDir: dir,
				Sources:    srcs,
				ChangedFiles: []pass.FileChange{
					{Path: testCase.file, Extension: filepath.Ext(testCase.file)},
				},
			})
			if len(got) != testCase.wantLen {
				t.Fatalf("got %d findings, want %d: %+v", len(got), testCase.wantLen, got)
			}
		})
	}
}

func TestAnalyze_NoLangs(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	got := analyzer.Analyze(
		&pass.Context{ChangedFiles: []pass.FileChange{{Path: "x.go", Extension: ".go"}}},
	)
	if got != nil {
		t.Fatalf("expected nil findings with no language registry, got %+v", got)
	}
}

func TestAnalyze_NoSources(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(testLangs())
	got := analyzer.Analyze(
		&pass.Context{ChangedFiles: []pass.FileChange{{Path: "x.go", Extension: ".go"}}},
	)
	if got != nil {
		t.Fatalf("expected nil findings with no source provider, got %+v", got)
	}
}

func TestAnalyze_UnknownLanguageSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "notes.json")
	if err := os.WriteFile(path, []byte(`{"simplification": "not code"}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	analyzer := NewAnalyzer(testLangs())
	got := analyzer.Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{{Path: "notes.json", Extension: ".json"}},
	})
	if got != nil {
		t.Fatalf("expected nil findings for an unregistered extension, got %+v", got)
	}
}

func TestAnalyze_NonexistentFileSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	analyzer := NewAnalyzer(testLangs())
	got := analyzer.Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{{Path: "gone.go", Extension: ".go"}},
	})
	if got != nil {
		t.Fatalf("expected nil findings for a nonexistent file, got %+v", got)
	}
}

func TestAnalyze_SkipsTestdata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixtureDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(fixtureDir, 0o750); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	path := filepath.Join(fixtureDir, "no_label.go")
	content := "package fixture\n\n" +
		"// simplification: skip the rare case because it never happens in practice.\n" +
		"func f() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	analyzer := NewAnalyzer(testLangs())
	got := analyzer.Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{{Path: "testdata/no_label.go", Extension: ".go"}},
	})
	if got != nil {
		t.Fatalf("got %+v findings for a testdata fixture, want nil", got)
	}
}

func TestBlockNamesUpgradePath_Wrapped(t *testing.T) {
	t.Parallel()

	lang := language.Lookup("go")
	testCases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"label on one line", []string{"// simplification: x", "// Upgrade path: do y"}, true},
		{
			"label wrapped by golines",
			[]string{"// simplification: x. Upgrade", "// path: do y"},
			true,
		},
		{
			"revisit-when wrapped",
			[]string{"// simplification: x, revisit", "// when y ships"},
			true,
		},
		{"no label at all", []string{"// simplification: x", "// and nothing else"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := blockNamesUpgradePath(lang, testCase.lines); got != testCase.want {
				t.Errorf(
					"blockNamesUpgradePath(%q) = %v, want %v",
					testCase.lines,
					got,
					testCase.want,
				)
			}
		})
	}
}
