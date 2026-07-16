package topoffile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/golang"
)

func TestTopOfFile_CleanFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "clean.go")
	if err := os.WriteFile(path, []byte("package foo\n\nimport \"fmt\"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "clean.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestTopOfFile_CommentBeforePackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(
		path,
		[]byte("// This is a package docstring\npackage foo\n"),
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
			{Path: "bad.go", Extension: ".go"},
		},
	})
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != core.SeverityError {
		t.Errorf("expected error severity, got %d", findings[0].Severity)
	}
	if findings[0].Line != 1 {
		t.Errorf("expected line 1, got %d", findings[0].Line)
	}
}

func TestTopOfFile_BlankLinesBeforePackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "blanks.go")
	if err := os.WriteFile(path, []byte("\n\npackage foo\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())
	a := NewAnalyzer(langs)
	findings := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{
			{Path: "blanks.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("blank lines before package should not trigger, got %d", len(findings))
	}
}

func TestTopOfFile_NoLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("// comment\n"), 0o644); err != nil {
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
		t.Fatalf("expected 0 findings for unregistered language, got %d", len(findings))
	}
}

func TestTopOfFile_GeneratedFileExempt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "gen.go")
	if err := os.WriteFile(
		path,
		[]byte("// Code generated DO NOT EDIT\npackage foo\n"),
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
			{Path: "gen.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("generated file header should be exempt, got %d findings", len(findings))
	}
}

func TestTopOfFile_BuildTagExempt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tag.go")
	if err := os.WriteFile(
		path,
		[]byte("//go:build integration\n// +build integration\n\npackage foo\n"),
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
			{Path: "tag.go", Extension: ".go"},
		},
	})
	if len(findings) != 0 {
		t.Fatalf("build-tag header should be exempt, got %d findings", len(findings))
	}
}
