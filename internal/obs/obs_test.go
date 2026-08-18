package obs

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "hook-execution.log")

	logger, closer, err := New(Config{Path: path})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	logger.Info("should be discarded", slog.Int("n", 1))
	if err = closer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no log file to be created, stat err = %v", err)
	}
}

// TestNewDisabledOnEmptyPath proves the other half of the disabled-by-default
// contract: a Level without a Path - an older installed host wrapper that
// never learned PULSARULES_LOG_PATH, or the binary run by hand with
// --log-level set - must not fall back to a guessed location or crash.
func TestNewDisabledOnEmptyPath(t *testing.T) {
	t.Parallel()

	logger, closer, err := New(Config{Level: "debug"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	logger.Debug("should be discarded", slog.Int("n", 1))
	if err = closer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewValidLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
	}{
		{name: "debug"},
		{name: "info"},
		{name: "warn"},
		{name: "error"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "hook-execution.log")

			logger, closer, err := New(Config{Level: testCase.name, Path: path})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			// Error() always clears the configured threshold, so this
			// exercises every level without needing a name-to-Level lookup.
			logger.Error("recorded", slog.String("case", testCase.name))
			if err = closer.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			logged, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read log: %v", err)
			}
			if !bytes.Contains(logged, []byte(`"case":"`+testCase.name+`"`)) {
				t.Fatalf("log missing record: %s", logged)
			}
		})
	}
}

func TestNewInvalidLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, _, err := New(Config{Level: "bogus", Path: filepath.Join(dir, "hook-execution.log")})
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
	if !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("err = %v, want wrapping ErrInvalidLevel", err)
	}
}

func TestTruncateToTail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		lines    int
		maxBytes int64
		wantCut  bool
	}{
		{name: "under cap left untouched", lines: 5, maxBytes: 4096, wantCut: false},
		{name: "over cap truncated to tail", lines: 500, maxBytes: 512, wantCut: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "hook-execution.log")
			original := writeLines(t, path, testCase.lines)

			if err := truncateToTail(path, testCase.maxBytes); err != nil {
				t.Fatalf("truncateToTail: %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			if !testCase.wantCut {
				if !bytes.Equal(got, original) {
					t.Fatalf(
						"file under cap was rewritten: got %d bytes, want %d",
						len(got),
						len(original),
					)
				}
				return
			}

			if len(got) >= len(original) {
				t.Fatalf(
					"truncated file not smaller: got %d bytes, original %d",
					len(got),
					len(original),
				)
			}
			if !bytes.HasSuffix(original, got) {
				t.Fatal("truncated content is not a suffix of the original")
			}
			assertLineBoundary(t, got)
		})
	}
}

// TestNew_MkdirAllFailure proves the os.MkdirAll branch: a Path whose parent
// is an ordinary file (not a directory) cannot have its log directory
// created, and New surfaces that as a wrapped error rather than a logger.
func TestNew_MkdirAllFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	path := filepath.Join(blocker, "hook-execution.log")

	logger, closer, err := New(Config{Level: "info", Path: path})
	if err == nil {
		t.Fatal("New: expected an error for a path whose parent is a file")
	}
	if logger != nil || closer != nil {
		t.Fatalf("New: expected nil logger/closer on error, got %v/%v", logger, closer)
	}
	if !strings.Contains(err.Error(), "create log dir") {
		t.Fatalf("err = %v, want it to name the mkdir step", err)
	}
}

// TestNew_OpenFileFailure proves the os.OpenFile branch: the log directory
// exists and truncateToTail finds no prior file to rewrite, but the
// directory itself denies write access, so creating the log file fails.
func TestNew_OpenFileFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logDir := filepath.Join(dir, "restricted")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		t.Fatalf("seed log dir: %v", err)
	}
	if err := os.Chmod(logDir, 0o500); err != nil {
		t.Fatalf("restrict log dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(logDir, 0o750) })
	path := filepath.Join(logDir, "hook-execution.log")

	logger, closer, err := New(Config{Level: "info", Path: path})
	if err == nil {
		t.Fatal("New: expected an error for a read-only log directory")
	}
	if logger != nil || closer != nil {
		t.Fatalf("New: expected nil logger/closer on error, got %v/%v", logger, closer)
	}
	if !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("err = %v, want it to name the open step", err)
	}
}

func TestTruncateToTailMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := truncateToTail(filepath.Join(dir, "missing.log"), 1024); err != nil {
		t.Fatalf("truncateToTail on missing file: %v", err)
	}
}

// writeLines writes n compact JSON records, one per line, to path and returns
// the exact bytes written so the test can assert the truncated tail is a
// genuine suffix of it.
func writeLines(t *testing.T, path string, n int) []byte {
	t.Helper()

	var buf bytes.Buffer
	for i := range n {
		rec, err := json.Marshal(map[string]int{"n": i})
		if err != nil {
			t.Fatalf("marshal record %d: %v", i, err)
		}
		buf.Write(rec)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return buf.Bytes()
}

// assertLineBoundary fails the test if any line in log is not a complete
// JSON record, which is what a mid-line cut would produce.
func assertLineBoundary(t *testing.T, log []byte) {
	t.Helper()

	for line := range bytes.SplitSeq(bytes.TrimRight(log, "\n"), []byte("\n")) {
		var rec map[string]int
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("line is not a whole record (mid-line cut): %q: %v", line, err)
		}
	}
}
