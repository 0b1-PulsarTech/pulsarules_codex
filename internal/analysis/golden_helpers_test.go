package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// expectedFinding is one row of a golden case's expect.json: the exact
// analyzer/file/line triple a finding must match, and a message substring
// (never the full text) so a reworded message does not break the fixture.
type expectedFinding struct {
	AnalyzerID string `json:"analyzerId"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Substring  string `json:"messageSubstring"`
}

// goldenExpectation is the schema of expect.json: the findings a default run
// must produce, how many the suppression pass must have hidden, and
// (generated-file cases only) the findings a run with IncludeGenerated set
// must produce instead, proving the hidden finding was real rather than
// never generated.
type goldenExpectation struct {
	Findings                 []expectedFinding `json:"findings"`
	SuppressedGenerated      int               `json:"suppressedGenerated"`
	IncludeGeneratedFindings []expectedFinding `json:"includeGeneratedFindings"`
}

// loadExpectation reads and parses a case's expect.json.
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

// copyTree copies every file under src into dst, preserving the relative
// tree, but skips expect.json: that file is the case's metadata, not part
// of the fixture repository the pipeline analyzes.
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

// assertFindings compares got against want exactly: same count, and each
// pair matches on AnalyzerID/File/Line exactly and on Message by substring,
// so a reworded message never breaks the fixture.
func assertFindings(t *testing.T, caseName string, got []core.Finding, want []expectedFinding) {
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
