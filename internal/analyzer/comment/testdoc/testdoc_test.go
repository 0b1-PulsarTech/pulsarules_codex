package testdoc

import (
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

func testCommentCases() []testCommentCase {
	return append(testCommentKindCases(), testCommentFlagCases()...)
}

func testCommentKindCases() []testCommentCase {
	return []testCommentCase{
		{
			name: "test helper is never flagged even with a doc comment",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// setupServer builds the fixture every case shares.\n" +
				"func setupServer(t *testing.T) {\n\tt.Helper()\n}\n",
			expect: 0,
		},
		{
			name: "a fixture/table type is never flagged",
			source: "package p\n\n" +
				"// testCase is one table row.\ntype testCase struct{ name string }\n",
			expect: 0,
		},
		{
			name: "lowercase letter after the prefix is a helper, not a test",
			source: "package p\n\nimport \"testing\"\n\n" +
				"// Testify is not a recognized test function.\nfunc Testify(t *testing.T) {\n" +
				"\tt.Helper()\n}\n",
			expect: 0,
		},
		{
			name: "a method named like a test is never a runnable test",
			source: "package p\n\nimport \"testing\"\n\n" +
				"type S struct{}\n\n// TestFoo documents a method, not a test.\n" +
				"func (s S) TestFoo(t *testing.T) {}\n",
			expect: 0,
		},
	}
}

func testCommentFlagCases() []testCommentCase {
	return []testCommentCase{
		{
			name: "clean test not flagged",
			source: "package p\n\nimport \"testing\"\n\n" +
				"func TestFoo(t *testing.T) {\n\tt.Parallel()\n}\n",
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

func TestAnalyze_NoASTCache(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}
