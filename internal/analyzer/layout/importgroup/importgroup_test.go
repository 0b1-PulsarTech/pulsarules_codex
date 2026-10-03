package importgroup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *pass.Context
		expect int
	}{
		{
			name: "no changed files",
			ctx:  &pass.Context{ASTCache: astcache.New()},
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ASTCache:     astcache.New(),
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

type checkFileTestCase struct {
	name   string
	source string
	expect int
}

func checkFileCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no imports",
			source: "package foo\n",
			expect: 0,
		},
		{
			name: "correct order std ext mod",
			source: "package foo\n" +
				"import (\n" +
				`	"fmt"` + "\n" +
				`	"os"` + "\n" +
				"\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				"\n" +
				`	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"` + "\n" +
				")\n",
			expect: 0,
		},
		{
			name: "module import before external",
			source: "package foo\n" +
				"import (\n" +
				`	"fmt"` + "\n" +
				"\n" +
				`	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"` + "\n" +
				"\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				")\n",
			expect: 1,
		},
		{
			name: "external before stdlib",
			source: "package foo\n" +
				"import (\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				"\n" +
				`	"fmt"` + "\n" +
				")\n",
			expect: 1,
		},
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	const modPath = "github.com/0b1-PulsarTech/pulsarules_codex"

	for _, testCase := range checkFileCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "foo.go")
			if err := os.WriteFile(path, []byte(testCase.source), 0o644); err != nil {
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
			got := analyzer.checkFile(cache.FileSet(), modPath, fc, file, analyzer.reporter)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

func TestAnalyze_ForeignModule(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	const modPath = "example.com/other"
	writeSource(t, filepath.Join(projectDir, "go.mod"), "module "+modPath+"\n\ngo 1.26\n")

	src := "package foo\n\nimport (\n\t\"fmt\"\n\n\t\"" + modPath +
		"/bar\"\n\n\t\"github.com/x/y\"\n)\n\nvar _ = fmt.Sprint\n"
	path := filepath.Join(projectDir, "foo.go")
	writeSource(t, path, src)

	cache := astcache.New()
	if _, err := cache.Parse(path, []byte(src)); err != nil {
		t.Fatalf("parse: %v", err)
	}

	got := NewAnalyzer().Analyze(&pass.Context{
		ProjectDir:   projectDir,
		ASTCache:     cache,
		ChangedFiles: []pass.FileChange{{Path: path, Extension: ".go"}},
	})
	if len(got) == 0 {
		t.Fatal("a local import grouped before the external one went unreported: the analyzer " +
			"did not recognise " + modPath + " as this project's own module")
	}
}

func writeSource(tb testing.TB, path, body string) {
	tb.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		tb.Fatalf("write %s: %v", path, err)
	}
}
