package doc

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(checkFileCases(), exemptionCases()...)
	testCases = append(testCases, deprecatedCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse(testCase.path, []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{
				Path:      testCase.path,
				Extension: ".go",
				IsTest:    testCase.path == "p_test.go",
			}
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestCheckFileHints(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		wantSymbol string
		wantHint   string
	}{
		{
			"func keeps the body hint",
			"package p\n\n// f does.\nfunc f() {}\n",
			"f carries a doc comment",
			"inside the body",
		},
		{
			"directive mixed with prose keeps the directive",
			"package p\n\n// goverter:converter\n// C converts.\ntype C interface{}\n",
			"C carries a doc comment",
			"keep the directive lines",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p.go", Extension: ".go"}
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) == 0 {
				t.Fatal("no finding")
			}
			if got[0].Message != testCase.wantSymbol ||
				!strings.Contains(got[0].Suggestion, testCase.wantHint) {
				t.Fatalf("finding = %+v", got[0])
			}
		})
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	source := []byte("package p\n\n// Foo does it.\nfunc Foo() {}\n")
	testCases := []struct {
		name      string
		generated bool
		expect    int
	}{
		{"regular file", false, 1},
		{"generated file is skipped", true, 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("p.go", source); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &pass.Context{
				ASTCache: cache,
				ChangedFiles: []pass.FileChange{
					{Path: "p.go", Extension: ".go", Generated: testCase.generated},
				},
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
	if analyzer.ID() != "doc-comment" || analyzer.Stage() != pass.StageAST {
		t.Fatalf("id = %q stage = %v", analyzer.ID(), analyzer.Stage())
	}
	if got := analyzer.Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings without an AST cache = %+v", got)
	}
}

type checkFileCase struct {
	name   string
	path   string
	source string
	expect int
}

func checkFileCases() []checkFileCase {
	return []checkFileCase{
		{"exported func doc", "p.go", "package p\n\n// Foo does it.\nfunc Foo() {}\n", 1},
		{"unexported func doc", "p.go", "package p\n\n// foo does it.\nfunc foo() {}\n", 1},
		{
			"method doc",
			"p.go",
			"package p\n\ntype S struct{}\n\n// Do does it.\nfunc (S) Do() {}\n",
			1,
		},
		{
			"multi-line group is one finding",
			"p.go",
			"package p\n\n// a\n// b\n// c\nfunc foo() {}\n",
			1,
		},
		{"type doc on GenDecl", "p.go", "package p\n\n// S is a thing.\ntype S struct{}\n", 1},
		{
			"type doc inside a grouped decl",
			"p.go",
			"package p\n\ntype (\n\t// A is a.\n\tA struct{}\n\n\tB struct{}\n)\n",
			1,
		},
		{"no doc", "p.go", "package p\n\nfunc foo() {}\n\ntype S struct{}\n", 0},
		{"var doc is ignored", "p.go", "package p\n\n// v is a var.\nvar v int\n", 0},
		{"go directive only", "p.go", "package p\n\n//go:noinline\nfunc foo() {}\n", 0},
		{
			"nolint directive only",
			"p.go",
			"package p\n\n//nolint:ireturn // reason\nfunc foo() any { return nil }\n",
			0,
		},
		{
			"prose plus directive is flagged",
			"p.go",
			"package p\n\n// foo does it.\n//nolint:ireturn // reason\n" +
				"func foo() any { return nil }\n",
			1,
		},
		{
			"test function is owned by test-comment",
			"p_test.go",
			"package p\n\nimport \"testing\"\n\n// TestFoo checks.\n" +
				"func TestFoo(t *testing.T) {}\n",
			0,
		},
		{
			"test helper is owned by test-comment",
			"p_test.go",
			"package p\n\nimport \"testing\"\n\n// setup builds.\nfunc setup(tb testing.TB) {}\n",
			0,
		},
		{
			"plain func in a test file is flagged here",
			"p_test.go",
			"package p\n\n// double doubles.\nfunc double(x int) int { return x * 2 }\n",
			1,
		},
		{
			"type in a test file is flagged here",
			"p_test.go",
			"package p\n\n// row is a case.\ntype row struct{}\n",
			1,
		},
	}
}

func exemptionCases() []checkFileCase {
	return append(exemptionDocShapeCases(), exemptionToolDirectiveCases()...)
}

func exemptionDocShapeCases() []checkFileCase {
	return []checkFileCase{
		{"const doc is not flagged", "p.go", "package p\n\n// c is a const.\nconst c = 1\n", 0},
		{
			"struct field doc is not flagged",
			"p.go",
			"package p\n\ntype S struct {\n\t// F is a field.\n\tF int\n}\n",
			0,
		},
		{
			"interface method doc is flagged",
			"p.go",
			"package p\n\ntype I interface {\n\t// M does.\n\tM()\n}\n",
			1,
		},
		{
			"deprecated-only is exempt",
			"p.go",
			"package p\n\n// Deprecated: use G.\nfunc F() {}\n",
			0,
		},
		{
			"deprecated plus prose is flagged",
			"p.go",
			"package p\n\n// F does it.\n//\n// Deprecated: use G.\nfunc F() {}\n",
			1,
		},
		{
			"goverter spaced form is a directive",
			"p.go",
			"package p\n\n// goverter:converter\n// goverter:output:file x_gen.go\n" +
				"type C interface{}\n",
			0,
		},
		{
			"pulsarules spaced form",
			"p.go",
			"package p\n\n// pulsarules:process-lifetime\nfunc f() {}\n",
			0,
		},
		{
			"directive plus prose lines is flagged",
			"p.go",
			"package p\n\n//pulsarules:process-lifetime x\n// a\n// b\nfunc f() {}\n",
			1,
		},
	}
}

func exemptionToolDirectiveCases() []checkFileCase {
	return []checkFileCase{
		{"spaced nolint is not exempt", "p.go", "package p\n\n// nolint:ireturn\n" +
			"func f() {}\n", 1},
		{"bare nolint is exempt", "p.go", "package p\n\n//nolint\nfunc f() {}\n", 0},
		{
			"lint ignore is exempt",
			"p.go",
			"package p\n\n//lint:ignore SA1019 reason\nfunc f() {}\n",
			0,
		},
		{
			"todo directive-shaped prose is flagged",
			"p.go",
			"package p\n\n//todo:remove this hack once X lands\nfunc f() {}\n",
			1,
		},
		{
			"simplification block is exempt",
			"p.go",
			"package p\n\n// simplification: caps at 7. Upgrade path: x.\nfunc f() {}\n",
			0,
		},
		{
			"TestMain is owned by test-comment",
			"p_test.go",
			"package p\n\nimport \"testing\"\n\n// TestMain boots.\n" +
				"func TestMain(m *testing.M) {}\n",
			0,
		},
		{
			"grouped type decl flags group and each spec",
			"p.go",
			"package p\n\n// group doc\ntype (\n\t// A doc\n\tA int\n\t// B doc\n\tB int\n)\n",
			3,
		},
	}
}

func TestCheckFileMessages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		wantSymbol string
		wantHint   string
	}{
		{
			"type group is labelled as a group",
			"package p\n\n// group doc\ntype (\n\tA int\n\tB int\n)\n",
			"type group carries a doc comment",
			"type name and fields",
		},
		{
			"single type uses its own name",
			"package p\n\n// S is a thing.\ntype S struct{}\n",
			"S carries a doc comment",
			"type name and fields",
		},
		{
			"interface method is qualified",
			"package p\n\ntype I interface {\n\t// M does.\n\tM()\n}\n",
			"I.M carries a doc comment",
			"method name and signature",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p.go", Extension: ".go"}
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) == 0 {
				t.Fatal("no finding")
			}
			if got[0].Message != testCase.wantSymbol ||
				!strings.Contains(got[0].Suggestion, testCase.wantHint) {
				t.Fatalf("finding = %+v", got[0])
			}
		})
	}
}

func deprecatedCases() []checkFileCase {
	return []checkFileCase{
		{
			"deprecated plus four lines is flagged",
			"p.go",
			"package p\n\n// Deprecated: use G.\n// a\n// b\n// c\nfunc F() {}\n",
			1,
		},
		{
			"deprecated spanning three lines is exempt",
			"p.go",
			"package p\n\n// Deprecated: use G,\n// because a\n// and b.\nfunc F() {}\n",
			0,
		},
	}
}
