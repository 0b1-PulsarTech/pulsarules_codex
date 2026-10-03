package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedTree(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "testdata"), 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		"notes.md":            "a\u200Bb and an em\u2014dash\n",
		"code.go":             "package a // zero\u200Bwidth\n",
		"skip.txt":            "a\u200Bb\n",
		"testdata/fixture.md": "a\u200Bb\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
			tb.Fatalf("seed %q: %v", rel, err)
		}
	}
	return root
}

func TestSweep_ReportOnlyWritesNothing(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	before, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var out strings.Builder
	if sweepErr := sweep(&out, root, report); sweepErr != nil {
		t.Fatalf("sweep: %v", sweepErr)
	}
	after, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("report-only rewrote the file")
	}
	if !strings.Contains(out.String(), "re-run with --write") {
		t.Errorf("output = %q, want it to name the flag", out.String())
	}
}

func TestSweep_Write(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	var out strings.Builder
	if err := sweep(&out, root, rewrite); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	testCases := []struct {
		name      string
		rel       string
		wantClean bool
	}{
		{"markdown carrier removed", "notes.md", true},
		{"go carrier removed", "code.go", true},
		{"unlisted extension untouched", "skip.txt", false},
		{"fixture untouched", "testdata/fixture.md", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body, err := os.ReadFile(filepath.Join(root, testCase.rel))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			clean := !strings.ContainsRune(string(body), '\u200B')
			if clean != testCase.wantClean {
				t.Errorf("%q clean = %v, want %v", testCase.rel, clean, testCase.wantClean)
			}
		})
	}

	t.Run("em dash survives, reported not rewritten", func(t *testing.T) {
		t.Parallel()

		notes, err := os.ReadFile(filepath.Join(root, "notes.md"))
		if err != nil {
			t.Fatalf("read notes: %v", err)
		}
		if !strings.ContainsRune(string(notes), '\u2014') {
			t.Error("the em dash was rewritten; only carriers may be removed")
		}
	})
}

func TestSweep_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	var first, second strings.Builder
	if err := sweep(&first, root, rewrite); err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if err := sweep(&second, root, rewrite); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if !strings.Contains(second.String(), "removed 0 carrier(s)") {
		t.Errorf("second sweep acted again: %q", second.String())
	}
}

func TestSweep_QuotesForgedPath(t *testing.T) {
	t.Parallel()

	name := "a\n[ERROR] x\nb.md"
	testCases := []struct {
		name string
		how  mode
	}{
		{"report", report},
		{"rewrite", rewrite},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			body := []byte("a\u200Bb\n")
			if err := os.WriteFile(filepath.Join(root, name), body, 0o600); err != nil {
				t.Skipf("filesystem refuses the name: %v", err)
			}
			var out strings.Builder
			if err := sweep(&out, root, testCase.how); err != nil {
				t.Fatalf("sweep: %v", err)
			}
			for line := range strings.SplitSeq(out.String(), "\n") {
				if strings.HasPrefix(line, "[ERROR]") {
					t.Errorf("forged line in output: %q", out.String())
				}
			}
			if !strings.Contains(out.String(), `\n[ERROR] x\nb.md"`) {
				t.Errorf("path not quoted: %q", out.String())
			}
		})
	}
}
