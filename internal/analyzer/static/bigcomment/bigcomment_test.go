package bigcomment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/golang"
)

func TestBigComment_SmallComment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "small.go")
	if err := os.WriteFile(
		path,
		[]byte("package foo\n\n// This is a short comment\nfunc bar() {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "small.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for small comment, got %d", len(findings))
	}
}

func TestBigComment_LargeBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "big.go")
	var b strings.Builder
	b.WriteString("package foo\n\n")
	for count := range 25 {
		b.WriteString("// comment line ")
		b.WriteByte(byte('A' + count))
		b.WriteByte('\n')
	}
	content := b.String()
	content += "func bar() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "big.go", Extension: ".go"},
		},
	})
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != core.SeverityWarning {
		t.Errorf("expected warning severity, got %d", findings[0].Severity)
	}
}

func TestBigComment_NoLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("// comment\n// comment\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "file.txt", Extension: ".txt"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("expected 0 for unregistered language, got %d", len(findings))
	}
}

func TestBigComment_GeneratedFileExempt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "gen.go")
	var b strings.Builder
	b.WriteString("// Code generated DO NOT EDIT\n")
	for range 25 {
		b.WriteString("// generated line\n")
	}
	b.WriteString("package foo\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "gen.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("generated file header should be exempt, got %d findings", len(findings))
	}
}
