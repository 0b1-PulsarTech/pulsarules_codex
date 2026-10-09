//go:build unix

package tssyntax

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestProviderSummarizeIgnoresFIFOConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "tsconfig.json"), 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	files := map[string]string{
		"package.json":  `{"name": "x"}`,
		"tsconfig.json": `{"compilerOptions": {"paths": {"@x/*": ["./src/*"]}}}`,
		"src/a.ts":      "import '@x/y';\n",
	}
	read := func(path string) ([]byte, bool) {
		text, ok := files[path]
		return []byte(text), ok
	}
	type result struct {
		files map[string]*syntax.File
		err   error
	}
	done := make(chan result, 1)
	go func() {
		files, err := New().Summarize(context.Background(), syntax.Corpus{
			Root:    root,
			Sources: []syntax.Source{{Path: "src/a.ts"}},
			Read:    read,
		})
		done <- result{files, err}
	}()
	var got result
	select {
	case got = <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("summarize blocked on a fifo")
	}
	if got.err != nil {
		t.Fatalf("summarize: %v", got.err)
	}
	imports := got.files["src/a.ts"].Imports
	if len(imports) != 1 || imports[0].Group != 1 {
		t.Fatalf("imports %+v, want one external import", imports)
	}
}
