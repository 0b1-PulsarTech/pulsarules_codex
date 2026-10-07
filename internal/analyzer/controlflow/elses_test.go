package controlflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

type elseTestCase struct {
	name   string
	source string
	expect int
}

func elseCases() []elseTestCase {
	return []elseTestCase{
		{
			name:   "clean if",
			source: "package foo\nfunc f() { if true { return }; x := 1; _ = x }\n",
			expect: 0,
		},
		{
			name: "else return",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		return\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else panic",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		panic(\"x\")\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else for loop",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		for i := 0; i < 10; i++ {}\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else break",
			source: "package foo\n" +
				"func f() {\n" +
				"	for i := 0; i < 10; i++ {\n" +
				"		if true {\n" +
				"		} else {\n" +
				"			break\n" +
				"		}\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else if no finding",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else if false {\n" +
				"		return\n" +
				"	}\n" +
				"}\n",
			expect: 0,
		},
	}
}

func TestCheckElses(t *testing.T) {
	t.Parallel()

	for _, testCase := range elseCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCheckFile(t, testCase.source, testCase.expect)
		})
	}
}

func runCheckFile(t *testing.T, source string, expect int) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cache := astcache.New()
	file, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	fc := pass.FileChange{Path: "foo.go", Extension: ".go"}
	got := checkElses(cache.FileSet(), fc, file, NewAnalyzer().reporter)
	total := 0
	for _, found := range got {
		total += len(found)
	}
	if total != expect {
		t.Fatalf("got %d findings, want %d: %v", len(got), expect, got)
	}
}
