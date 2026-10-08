package bodyheader

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

type headerCase struct {
	name     string
	source   string
	minLines int
	expect   int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(baseCases(), shapeCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p.go", Extension: ".go"}
			settings := check{
				fset:     cache.FileSet(),
				reporter: NewAnalyzer().reporter(),
				minLines: testCase.minLines,
			}
			got := settings.file(fc, file)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	source := []byte("package p\n\nfunc f() {\n\t// a\n\t// b\n\t_ = 1\n}\n")
	testCases := []struct {
		name   string
		change pass.FileChange
		expect int
	}{
		{"regular file", pass.FileChange{Path: "p.go", Extension: ".go"}, 1},
		{
			"plain func in a test file is checked",
			pass.FileChange{Path: "p.go", Extension: ".go", IsTest: true},
			1,
		},
		{
			"generated file is skipped",
			pass.FileChange{Path: "p.go", Extension: ".go", Generated: true},
			0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("p.go", source); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &pass.Context{
				ASTCache:     cache,
				ChangedFiles: []pass.FileChange{testCase.change},
			}
			if got := NewAnalyzer().Analyze(ctx); len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	if analyzer.ID() != "body-header-comment" || analyzer.Stage() != finding.StageAST {
		t.Fatalf("id = %q stage = %v", analyzer.ID(), analyzer.Stage())
	}
	if got := analyzer.Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings without an AST cache = %+v", got)
	}
}

func TestCheckFileInTestFile(t *testing.T) {
	t.Parallel()

	const header = "package p\n\nimport \"testing\"\n\n"
	testCases := []struct {
		name   string
		source string
		expect int
	}{
		{
			"test func is owned by test-comment",
			header + "func TestA(t *testing.T) {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			0,
		},
		{
			"helper is owned by test-comment",
			header + "func setup(tb testing.TB) {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			0,
		},
		{
			"literal inside a test func is owned by test-comment",
			header + "func TestA(t *testing.T) {\n\tf := func() {\n\t\t// a\n\t\t// b\n" +
				"\t\t_ = 1\n\t}\n\t_ = f\n}\n",
			0,
		},
		{
			"plain func in a test file is checked",
			header + "func plain() {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			1,
		},
		{
			"package-level literal in a test file is checked",
			header + "var g = func() {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p_test.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p_test.go", Extension: ".go", IsTest: true}
			settings := check{
				fset: cache.FileSet(), reporter: NewAnalyzer().reporter(), minLines: 2,
			}
			got := settings.file(fc, file)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func baseCases() []headerCase {
	return []headerCase{
		{"two-line header", "package p\n\nfunc f() {\n\t// a\n\t// b\n\t_ = 1\n}\n", 2, 1},
		{"one-line header", "package p\n\nfunc f() {\n\t// a\n\t_ = 1\n}\n", 2, 0},
		{"no comment", "package p\n\nfunc f() {\n\t_ = 1\n}\n", 2, 0},
		{
			"comment between statements",
			"package p\n\nfunc f() {\n\t_ = 1\n\t// a\n\t// b\n\t_ = 2\n}\n",
			2,
			0,
		},
		{"empty body", "package p\n\nfunc f() {\n\t// a\n\t// b\n}\n", 2, 1},
		{
			"method",
			"package p\n\ntype S struct{}\n\nfunc (S) f() {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			2,
			1,
		},
		{"block comment header", "package p\n\nfunc f() {\n\t/* a\n\tb */\n\t_ = 1\n}\n", 2, 1},
		{
			"function literal header",
			"package p\n\nvar g = func() {\n\t// a\n\t// b\n\t_ = 1\n}\n",
			2,
			1,
		},
		{
			"nolint directive group",
			"package p\n\nfunc f() {\n\t//nolint:a // x\n\t//nolint:b // y\n\t_ = 1\n}\n",
			2,
			0,
		},
		{
			"one prose line plus directive is not a block",
			"package p\n\nfunc f() {\n\t// why\n\t//nolint:a // y\n\t_ = 1\n}\n",
			2,
			0,
		},
		{"min_lines raised", "package p\n\nfunc f() {\n\t// a\n\t// b\n\t_ = 1\n}\n", 3, 0},
		{"trailing comment on the brace line", "package p\n\nfunc f() { // a\n\t_ = 1\n}\n", 2, 0},
		{"bodyless declaration", "package p\n\nfunc f()\n", 2, 0},
	}
}

func shapeCases() []headerCase {
	return []headerCase{
		{
			"nolint line plus one prose line is one prose line",
			"package p\n\nfunc f() {\n\t//nolint:gosec // reason\n\t// why\n\t_ = 1\n}\n",
			2,
			0,
		},
		{
			"go generate plus one prose line",
			"package p\n\nfunc f() {\n\t//go:generate echo hi\n\t// why\n\t_ = 1\n}\n",
			2,
			0,
		},
		{
			"two prose lines plus a directive",
			"package p\n\nfunc f() {\n\t//nolint:gosec // reason\n\t// a\n\t// b\n\t_ = 1\n}\n",
			2,
			1,
		},
		{
			"blank-separated singles are summed",
			"package p\n\nfunc f() {\n\t// a\n\n\t// b\n\t_ = 1\n}\n",
			2,
			1,
		},
		{
			"brace trailing plus next line",
			"package p\n\nfunc f() { // a\n\t// b\n\t_ = 1\n}\n",
			2,
			1,
		},
		{"one-line block comment", "package p\n\nfunc f() {\n\t/* a */\n\t_ = 1\n}\n", 2, 0},
		{
			"three-line block comment",
			"package p\n\nfunc f() {\n\t/* a\n\tb\n\tc */\n\t_ = 1\n}\n",
			2,
			1,
		},
		{
			"literal as first statement",
			"package p\n\nfunc f() {\n\tg := func() {\n\t\t// a\n\t\t// b\n\t\t_ = 1\n\t}\n" +
				"\t_ = g\n}\n",
			2,
			1,
		},
		{
			"simplification block is exempt",
			"package p\n\nfunc f() {\n\t// simplification: caps at 7.\n" +
				"\t// Upgrade path: compare runs.\n\t_ = 1\n}\n",
			2,
			0,
		},
		{
			"pulsarules marker with its reason is exempt",
			"package p\n\nfunc f() {\n\t// pulsarules:process-lifetime the ticker\n" +
				"\t// lives for the process\n\tfor {}\n}\n",
			2,
			0,
		},
		{
			"const explanation at the body top stays flagged",
			"package p\n\nfunc f() int {\n\t// maxN bounds the scan; larger inputs\n" +
				"\t// are truncated by the caller.\n\tconst maxN = 7\n\treturn maxN\n}\n",
			2,
			1,
		},
	}
}
