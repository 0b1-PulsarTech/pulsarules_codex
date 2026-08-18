package imports

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	modPath := "github.com/0b1-PulsarTech/pulsarules_codex"
	a := NewAnalyzer(modPath)

	testCases := []struct {
		name   string
		ctx    *core.AnalysisContext
		expect int
	}{
		{
			name: "no changed files",
			ctx:  &core.AnalysisContext{ASTCache: astcache.New()},
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ASTCache:     astcache.New(),
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := a.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer("github.com/0b1-PulsarTech/pulsarules_codex")

	testCases := []struct {
		name   string
		source string
		expect int
	}{
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			path := filepath.Join(tmp, "foo.go")
			if err := os.WriteFile(path, []byte(testCase.source), 0o644); err != nil {
				t.Fatal(err)
			}
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			cache := astcache.New()
			f, err := cache.Parse(path, src)
			if err != nil {
				t.Fatal(err)
			}

			fc := core.FileChange{Path: "foo.go", Extension: ".go"}
			got := a.checkFile(cache.FileSet(), fc, f)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}
