package noemdash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cleanFile := filepath.Join(dir, "clean.go")
	if err := os.WriteFile(cleanFile, []byte("package foo\n\nfunc f() {}\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	dirtyFile := filepath.Join(dir, "dirty.go")
	if err := os.WriteFile(
		dirtyFile,
		[]byte("package foo\n\n// \u2014 em-dash comment\n"),
		0o644,
	); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	a := NewAnalyzer()
	srcs := pass.NewSourceProvider(dir)

	testCases := []struct {
		name   string
		ctx    *pass.Context
		expect int
	}{
		{
			name: "no changed files",
			ctx:  &pass.Context{ProjectDir: dir, Sources: srcs, ChangedFiles: nil},
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
		{
			name: "clean file no finding",
			ctx: &pass.Context{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []pass.FileChange{{Path: "clean.go", Extension: ".go"}},
			},
		},
		{
			name: "dirty file has finding",
			ctx: &pass.Context{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []pass.FileChange{{Path: "dirty.go", Extension: ".go"}},
			},
			expect: 1,
		},
		{
			name: "nonexistent file skipped",
			ctx: &pass.Context{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []pass.FileChange{{Path: "gone.go", Extension: ".go"}},
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
