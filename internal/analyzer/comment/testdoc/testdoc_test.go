package testdoc

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

type testCommentCase struct {
	name   string
	source string
	expect int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range testCommentCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p_test.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p_test.go", Extension: ".go", IsTest: true}
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestCheckFile_ReportsTheCommentLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		source    string
		wantLines []int
	}{
		{
			name: "a body comment is reported at its own line",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestX(t *testing.T) {\n\t_ = 1\n\t// build it\n\t_ = 2\n}\n",
			wantLines: []int{7},
		},
		{
			name: "a doc comment is reported at its own line",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// TestX does it.\nfunc TestX(t *testing.T) {}\n",
			wantLines: []int{5},
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
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			var lines []int
			for _, finding := range got {
				lines = append(lines, finding.Line)
			}
			if !slices.Equal(lines, testCase.wantLines) {
				t.Fatalf("lines = %v, want %v: %+v", lines, testCase.wantLines, got)
			}
		})
	}
}

func testCommentCases() []testCommentCase {
	return append(testCommentKindCases(), testCommentFlagCases()...)
}

func testCommentKindCases() []testCommentCase {
	return []testCommentCase{
		{
			name: "helper calling t.Helper with a doc comment is flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// setupServer builds the fixture every case shares.\n" +
				"func setupServer(t *testing.T) {\n\tt.Helper()\n}\n",
			expect: 1,
		},
		{
			name: "helper taking testing.TB is flagged for a body comment",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func newFixture(tb testing.TB) {\n\t// build it\n\t_ = 1\n}\n",
			expect: 1,
		},
		{
			name: "helper taking testing.B is flagged for a doc comment",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// warm primes the cache.\nfunc warm(b *testing.B) {}\n",
			expect: 1,
		},
		{
			name: "Helper call on a non-testing receiver is not a helper",
			source: "package p\n\ntype h struct{}\n\nfunc (h) Helper() {}\n\n" +
				"// check asserts.\nfunc check(x h) {\n\tx.Helper()\n}\n",
			expect: 0,
		},
		{
			name: "helper with a Helper call on a t field is flagged",
			source: "package p\n\ntype s struct{ t T }\n\n" +
				"// check asserts.\nfunc (x s) check() {\n\tx.t.Helper()\n}\n",
			expect: 1,
		},
		{
			name: "helper method is flagged",
			source: "package p\n\nimport \"testing\"\n\ntype S struct{}\n\n" +
				"// assertOK asserts.\nfunc (s S) assertOK(t *testing.T) {\n\tt.Helper()\n}\n",
			expect: 1,
		},
		{
			name: "method named like a test with a testing param is covered",
			source: "package p\n\nimport \"testing\"\n\n" +
				"type S struct{}\n\n// TestFoo documents a method.\n" +
				"func (s S) TestFoo(t *testing.T) {}\n",
			expect: 1,
		},
		{
			name: "lowercase suffix after the prefix is still a helper",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// Testify is a helper.\nfunc Testify(t *testing.T) {\n\tt.Helper()\n}\n",
			expect: 1,
		},
		{
			name: "plain non-helper func is not covered",
			source: "package p\n\n" +
				"// double doubles.\nfunc double(x int) int {\n\t// twice\n\treturn x * 2\n}\n",
			expect: 0,
		},
		{
			name: "a fixture/table type is never flagged",
			source: "package p\n\n" +
				"// testCase is one table row.\ntype testCase struct{ name string }\n",
			expect: 0,
		},
	}
}

func testCommentFlagCases() []testCommentCase {
	cases := append(testCommentDirectiveCases(), testCommentFuncCases()...)
	return append(cases, testCommentExampleCases()...)
}

func testCommentFuncCases() []testCommentCase {
	return []testCommentCase{
		{
			name: "clean test not flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T) {\n\tt.Parallel()\n}\n",
			expect: 0,
		},
		{
			name: "comment after a test body is not a body comment",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T) {\n\tt.Parallel()\n}\n\n// trailing note\n",
			expect: 0,
		},
		{
			name: "bodyless test declaration beside a comment is not flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T)\n\n// trailing note\n",
			expect: 0,
		},
		{
			name: "doc comment on a test is flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// TestFoo checks the happy path.\nfunc TestFoo(t *testing.T) {\n" +
				"\tt.Parallel()\n}\n",
			expect: 1,
		},
		{
			name: "comment inside a test body is flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T) {\n\t// exercise the happy path\n\tt.Parallel()\n}\n",
			expect: 1,
		},
		{
			name: "nolint directive inside a test is not flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T) {\n\t//nolint:paralleltest // shared fixture\n" +
				"\tt.Log(\"x\")\n}\n",
			expect: 0,
		},
		{
			name: "nolint doc comment on a test is not flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"//nolint:paralleltest // shared fixture\nfunc TestFoo(t *testing.T) {\n" +
				"\tt.Log(\"x\")\n}\n",
			expect: 0,
		},
		{
			name: "benchmark doc comment is flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// BenchmarkFoo times the hot path.\nfunc BenchmarkFoo(b *testing.B) {\n" +
				"\tfor range b.N {\n\t}\n}\n",
			expect: 1,
		},
		{
			name: "fuzz doc comment is flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// FuzzFoo fuzzes the parser.\nfunc FuzzFoo(f *testing.F) {\n" +
				"\tf.Fuzz(func(t *testing.T, s string) {})\n}\n",
			expect: 1,
		},
		{
			name: "example doc comment is flagged",
			source: "package p\n\n" +
				"// ExampleFoo demonstrates Foo.\nfunc ExampleFoo() {\n\t_ = 1\n}\n",
			expect: 1,
		},
	}
}

func testCommentExampleCases() []testCommentCase {
	return []testCommentCase{
		{
			name: "example output comment is not flagged",
			source: "package p\n\nimport \"fmt\"\n\n" +
				"func ExampleFoo() {\n\tfmt.Println(\"hi\")\n\t// Output: hi\n}\n",
			expect: 0,
		},
		{
			name: "example unordered output comment is not flagged",
			source: "package p\n\nimport \"fmt\"\n\n" +
				"func ExampleFoo() {\n\tfmt.Println(\"hi\")\n\t// Unordered output: hi\n}\n",
			expect: 0,
		},
	}
}

func testCommentDirectiveCases() []testCommentCase {
	const header = "package p\n\nimport \"testing\"\n\n"
	return []testCommentCase{
		{
			name:   "go:noinline on a benchmark helper is exempt",
			source: header + "//go:noinline\nfunc sink(b *testing.B) {}\n",
			expect: 0,
		},
		{
			name: "lint:ignore in a test body is exempt",
			source: header + "func TestA(t *testing.T) {\n\t//lint:ignore SA1019 legacy\n" +
				"\t_ = 1\n}\n",
			expect: 0,
		},
		{
			name:   "TestMain body comment is flagged",
			source: header + "func TestMain(m *testing.M) {\n\t// boot db\n\tm.Run()\n}\n",
			expect: 1,
		},
		{
			name:   "aliased testing import helper is flagged",
			source: "package p\n\nimport tst \"testing\"\n\n// h helps.\nfunc h(t *tst.T) {}\n",
			expect: 1,
		},
		{
			name: "require.TestingT helper is flagged",
			source: "package p\n\nimport \"github.com/stretchr/testify/require\"\n\n" +
				"// h helps.\nfunc h(t require.TestingT) { _ = t }\n",
			expect: 1,
		},
		{
			name:   "method without a testing param or Helper call is not covered",
			source: "package p\n\ntype S struct{}\n\n// build makes it.\nfunc (S) build() {}\n",
			expect: 0,
		},
		{
			name:   "spaced nolint on a test is not exempt",
			source: header + "// nolint:paralleltest\nfunc TestA(t *testing.T) {}\n",
			expect: 1,
		},
	}
}

func TestAnalyze_SkipsNonTestFiles(t *testing.T) {
	t.Parallel()

	cache := astcache.New()
	src := []byte("package p\n\nimport \"testing\"\n\n" +
		"// TestFoo checks the happy path.\nfunc TestFoo(t *testing.T) {}\n")
	if _, err := cache.Parse("p.go", src); err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := &pass.Context{
		ASTCache:     cache,
		ChangedFiles: []pass.FileChange{{Path: "p.go", Extension: ".go", IsTest: false}},
	}
	if got := NewAnalyzer().Analyze(ctx); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}

func TestAnalyze_GeneratedTestFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		generated bool
		want      int
	}{
		{name: "a hand-written test file is checked", generated: false, want: 1},
		{name: "a generated test file is skipped", generated: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			src := []byte("package p\n\nimport \"testing\"\n\n" +
				"// TestFoo checks the happy path.\nfunc TestFoo(t *testing.T) {}\n")
			if _, err := cache.Parse("p_test.go", src); err != nil {
				t.Fatalf("parse: %v", err)
			}
			ctx := &pass.Context{
				ASTCache: cache,
				ChangedFiles: []pass.FileChange{{
					Path:      "p_test.go",
					Extension: ".go",
					IsTest:    true,
					Generated: testCase.generated,
				}},
			}
			if got := NewAnalyzer().Analyze(ctx); len(got) != testCase.want {
				t.Errorf("findings = %+v, want %d", got, testCase.want)
			}
		})
	}
}

func TestAnalyze_NoASTCache(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}

func TestCheckFileMessages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		source   string
		wantHint string
	}{
		{
			"helper doc comment",
			"package p\n\nimport \"testing\"\n\n// warm primes.\nfunc warm(b *testing.B) {}\n",
			"the name carries what it does",
		},
		{
			"test body comment",
			"package p\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {\n\t// a\n\t_ = 1\n}\n",
			"the name carries what it does",
		},
		{
			"fixture helper body comment",
			"package p\n\nimport \"testing\"\n\nfunc seed(t *testing.T) {\n\tt.Helper()\n" +
				"\t// a\n}\n",
			"the name carries what it does",
		},
		{
			"doc directive mixed with prose",
			"package p\n\nimport \"testing\"\n\n//nolint:x // why\n// seeds.\n" +
				"func seed(t *testing.T) {}\n",
			"keep the directive lines",
		},
		{
			"body directive mixed with prose",
			"package p\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {\n" +
				"\t//nolint:x // why\n\t// seeds.\n}\n",
			"keep the directive lines",
		},
		{
			"blank comment line is not a directive",
			"package p\n\nimport \"testing\"\n\n// seeds.\n//\nfunc seed(t *testing.T) {}\n",
			"the name carries what it does",
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
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) != 1 || !strings.Contains(got[0].Suggestion, testCase.wantHint) ||
				!strings.Contains(got[0].Message, "test functions and helpers carry no comments") {
				t.Fatalf("finding = %+v", got)
			}
		})
	}
}
