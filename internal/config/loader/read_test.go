package loader

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func TestReadBounded(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		size, want int
	}{
		{"file at the cap", projectfile.MaxFileBytes, projectfile.MaxFileBytes},
		{"file twice the cap", 2 * projectfile.MaxFileBytes, projectfile.MaxFileBytes + 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "big.yaml")
			if err := os.WriteFile(path, make([]byte, testCase.size), 0o600); err != nil {
				t.Fatalf("write file: %v", err)
			}
			content, err := readBounded(path)
			if err != nil || len(content) != testCase.want {
				t.Fatalf("readBounded() = %d bytes, %v, want %d", len(content), err, testCase.want)
			}
		})
	}
}

func makeFifo(t *testing.T, path string) {
	t.Helper()
	if err := exec.CommandContext(t.Context(), "mkfifo", path).Run(); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
}

func TestReadBoundedRefusesWhatIsNotRegular(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	makeFifo(t, pipe)
	testCases := []struct{ name, path string }{
		{"a fifo", pipe},
		{"a directory", dir},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := readBounded(testCase.path)
			if !errors.Is(err, errNotRegular) || err.Error() != "read config: not a regular file" {
				t.Fatalf("readBounded() error = %v, want %v", err, errNotRegular)
			}
		})
	}
}

func TestLoadRefusesAPrivateFileThatIsNotRegular(t *testing.T) {
	t.Parallel()

	testCases := []struct{ name, kind string }{
		{"a directory", "dir"},
		{"a named pipe", "fifo"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			setup := layout{git: true, privateKind: testCase.kind}
			result := loadWithin(t, setup.build(t), "")
			assertErrorHas(t, result.err, []string{private + "read config: not a regular file"})
		})
	}
}

func TestReadFaultsNameTheStepNotThePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked.yaml")
	write(t, locked, "version: 1\n")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	testCases := []struct {
		name, path, step string
		cause            error
		needsUser        bool
	}{
		{"the stat fails", filepath.Join(dir, "gone.yaml"), "stat", syscall.ENOENT, false},
		{"the open fails", locked, "open", syscall.EACCES, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.needsUser && os.Geteuid() == 0 {
				t.Skip("root opens a mode 000 file")
			}
			_, err := readBounded(testCase.path)
			want := "read config: " + testCase.step + ": " + testCase.cause.Error()
			if err == nil || err.Error() != want || !errors.Is(err, testCase.cause) {
				t.Fatalf("readBounded() error = %v, want %q wrapping %v", err, want, testCase.cause)
			}
		})
	}
}
