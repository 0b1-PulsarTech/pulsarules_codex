package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type expectedFinding struct {
	AnalyzerID string `json:"analyzerId"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Substring  string `json:"messageSubstring"`
}

type goldenExpectation struct {
	Findings                 []expectedFinding `json:"findings"`
	SuppressedGenerated      int               `json:"suppressedGenerated"`
	IncludeGeneratedFindings []expectedFinding `json:"includeGeneratedFindings"`
}

func loadExpectation(t *testing.T, path string) goldenExpectation {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var expectation goldenExpectation
	if err = json.Unmarshal(raw, &expectation); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return expectation
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read dir %s: %v", src, err)
	}
	for _, entry := range entries {
		if entry.Name() == "expect.json" {
			continue
		}
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err = os.MkdirAll(dstPath, 0o750); err != nil {
				t.Fatalf("mkdir %s: %v", dstPath, err)
			}
			copyTree(t, srcPath, dstPath)
			continue
		}
		raw, readErr := os.ReadFile(srcPath)
		if readErr != nil {
			t.Fatalf("read %s: %v", srcPath, readErr)
		}
		if writeErr := os.WriteFile(dstPath, raw, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", dstPath, writeErr)
		}
	}
}

func assertFindings(t *testing.T, caseName string, got []finding.Finding, want []expectedFinding) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("case %s: got %d findings, want %d\ngot: %+v\nwant: %+v",
			caseName, len(got), len(want), got, want)
	}
	for index, wantFinding := range want {
		gotFinding := got[index]
		if gotFinding.AnalyzerID != wantFinding.AnalyzerID ||
			gotFinding.File != wantFinding.File ||
			gotFinding.Line != wantFinding.Line {
			t.Errorf(
				"case %s: finding[%d] = {%s %s:%d}, want {%s %s:%d}",
				caseName, index,
				gotFinding.AnalyzerID, gotFinding.File, gotFinding.Line,
				wantFinding.AnalyzerID, wantFinding.File, wantFinding.Line,
			)
			continue
		}
		if wantFinding.Substring != "" &&
			!strings.Contains(gotFinding.Message, wantFinding.Substring) {
			t.Errorf(
				"case %s: finding[%d] message = %q, want substring %q",
				caseName, index, gotFinding.Message, wantFinding.Substring,
			)
		}
	}
}
