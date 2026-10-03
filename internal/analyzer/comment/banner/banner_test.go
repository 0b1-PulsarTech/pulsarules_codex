package banner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestTopOfFile_UnlistedAndFixtures(t *testing.T) {
	t.Parallel()

	testCases := []string{"file.txt", "testdata/bad.go"}
	for _, testCase := range testCases {
		findings := analyzeSource(t, testCase, "// comment\npackage a\n", nil)
		if len(findings) != 0 {
			t.Fatalf("%s: got %d findings, want 0: %+v", testCase, len(findings), findings)
		}
	}
}

func TestTopOfFile_ExtensionOutsideDefaultsSkipped(t *testing.T) {
	t.Parallel()

	const src = "// Copyright 2026\nclass Foo {}\n"
	if findings := analyzeSource(t, "licensed.java", src, nil); len(findings) != 0 {
		t.Fatalf("expected 0 findings for .java by default, got %d: %+v", len(findings), findings)
	}
	config := &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
		"top-of-file": {
			Enabled: true,
			Params:  map[string]any{"extensions": []string{".go", ".java"}},
		},
	}}
	if findings := analyzeSource(t, "licensed.java", src, config); len(findings) != 1 {
		t.Fatalf(
			"expected 1 finding for .java once opted in, got %d: %+v", len(findings), findings,
		)
	}
}

type languageCase struct {
	name string
	file string
	src  string
	want int
	line int
}

func analyzeSource(t *testing.T, file, src string, config *pass.AnalysisConfig) []pass.Finding {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	langs := pass.NewLanguageRegistry()
	language.RegisterAll(langs)
	return NewAnalyzer(langs).Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      pass.NewSourceProvider(dir, pass.NewSkipSet(pass.DefaultSkipDirs())),
		Config:       config,
		ChangedFiles: []pass.FileChange{{Path: file, Extension: filepath.Ext(file)}},
	})
}

func TestTopOfFile_Languages(t *testing.T) {
	t.Parallel()

	var testCases []languageCase
	for _, group := range [][]languageCase{
		goCases(),
		scriptCases(),
		docblockCases(),
		dartCases(),
		markupCases(),
		sqlCases(),
		dataCases(),
	} {
		testCases = append(testCases, group...)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := analyzeSource(t, testCase.file, testCase.src, nil)
			if len(findings) != testCase.want {
				t.Fatalf("got %d findings, want %d: %+v", len(findings), testCase.want, findings)
			}
			if testCase.want == 0 {
				return
			}
			if findings[0].Line != testCase.line {
				t.Errorf("line = %d, want %d", findings[0].Line, testCase.line)
			}
			if findings[0].Severity != pass.SeverityError {
				t.Errorf("severity = %d, want error", findings[0].Severity)
			}
		})
	}
}
