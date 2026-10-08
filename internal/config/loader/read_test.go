package loader

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

type endlessSource struct {
	served int
}

func (source *endlessSource) Read(buffer []byte) (int, error) {
	if source.served > 2*config.MaxFileBytes {
		return 0, errors.New("read past twice the cap")
	}
	source.served += len(buffer)
	return len(buffer), nil
}

func TestReadCapped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		source   io.Reader
		wantSize int
	}{
		{"small file", strings.NewReader("version: 1\n"), len("version: 1\n")},
		{
			"file at the cap",
			strings.NewReader(strings.Repeat("x", config.MaxFileBytes)),
			config.MaxFileBytes,
		},
		{"endless source", &endlessSource{}, config.MaxFileBytes + 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			content, err := readCapped(testCase.source)
			if err != nil {
				t.Fatalf("readCapped() error = %v", err)
			}
			if len(content) != testCase.wantSize {
				t.Fatalf("readCapped() read %d bytes, want %d", len(content), testCase.wantSize)
			}
		})
	}
}

func TestReadBounded(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		size, want int
	}{
		{"file at the cap", config.MaxFileBytes, config.MaxFileBytes},
		{"file twice the cap", 2 * config.MaxFileBytes, config.MaxFileBytes + 1},
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

func linkTo(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
}

type openKindCase struct {
	name       string
	prepare    func(t *testing.T, dir string) string
	wantOpened int
	wantErr    error
}

func openKindCases() []openKindCase {
	fifo := func(t *testing.T, dir string) string {
		t.Helper()
		path := filepath.Join(dir, "pipe")
		makeFifo(t, path)
		return path
	}
	return []openKindCase{
		{"a fifo", fifo, 0, errNotRegular},
		{"a directory", func(*testing.T, string) string { return "." }, 0, errNotRegular},
		{
			"a symlink to a device",
			func(t *testing.T, dir string) string {
				t.Helper()
				path := filepath.Join(dir, "link")
				linkTo(t, "/dev/zero", path)
				return path
			},
			0, errNotRegular,
		},
		{
			"a symlink to a fifo",
			func(t *testing.T, dir string) string {
				t.Helper()
				path := filepath.Join(dir, "link")
				linkTo(t, fifo(t, dir), path)
				return path
			},
			0, errNotRegular,
		},
		{
			"a missing file",
			func(_ *testing.T, dir string) string { return filepath.Join(dir, "nope") },
			0, fs.ErrNotExist,
		},
		{
			"a regular file",
			func(t *testing.T, dir string) string {
				t.Helper()
				path := filepath.Join(dir, "ok.yaml")
				if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return path
			},
			1, nil,
		},
	}
}

func TestReadFileOpensOnlyRegularFiles(t *testing.T) {
	t.Parallel()

	for _, testCase := range openKindCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			path := testCase.prepare(t, t.TempDir())
			opened := 0
			_, err := readFile(path, func(name string) (*os.File, error) {
				opened++
				return openNonBlocking(name)
			})
			if opened != testCase.wantOpened || !errors.Is(err, testCase.wantErr) {
				t.Fatalf("readFile() opened %d times, error = %v; want %d, %v",
					opened, err, testCase.wantOpened, testCase.wantErr)
			}
		})
	}
}

func TestReadFileChecksTheOpenedHandle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "ok.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	testCases := []struct {
		name    string
		open    opener
		wantErr error
	}{
		{"the path was swapped for a directory", func(string) (*os.File, error) {
			return os.Open(dir)
		}, errNotRegular},
		{"the path was swapped for a device", func(string) (*os.File, error) {
			return os.Open(os.DevNull)
		}, errNotRegular},
		{"the open fails", func(string) (*os.File, error) {
			return nil, fs.ErrPermission
		}, fs.ErrPermission},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := readFile(path, testCase.open); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("readFile() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestOpenNonBlockingDoesNotWaitForAWriter(t *testing.T) {
	t.Parallel()

	const deadline = 5 * time.Second
	path := filepath.Join(t.TempDir(), "pipe")
	makeFifo(t, path)
	done := make(chan error, 1)
	go func() {
		file, err := openNonBlocking(path)
		if err == nil {
			err = file.Close()
		}
		done <- err
	}()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("openNonBlocking() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("openNonBlocking() still waiting after %v for a writer", deadline)
	}
}

func TestRefused(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode fs.FileMode
		want bool
	}{
		{"a regular file", 0o644, false},
		{"a directory", fs.ModeDir | 0o755, true},
		{"a named pipe", fs.ModeNamedPipe | 0o644, true},
		{"a device", fs.ModeDevice | 0o644, true},
		{"a character device", fs.ModeDevice | fs.ModeCharDevice | 0o644, true},
		{"a character device flag alone", fs.ModeCharDevice | 0o644, true},
		{"a socket", fs.ModeSocket | 0o644, true},
		{"a Windows reparse point", fs.ModeIrregular | 0o644, false},
		{"a symlink the OS has not followed", fs.ModeSymlink | 0o644, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := refused(testCase.mode); got != testCase.want {
				t.Errorf("refused(%v) = %v, want %v", testCase.mode, got, testCase.want)
			}
		})
	}
}

func closedHandle(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return file, nil
}

func TestReadFaultsNameTheStepNotThePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	readable := filepath.Join(dir, "readable.yaml")
	locked := filepath.Join(dir, "locked.yaml")
	for _, path := range []string{readable, locked} {
		if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	testCases := []struct {
		name, path, step string
		open             opener
		cause            error
		needsUser        bool
	}{
		{
			"the stat fails", filepath.Join(dir, "gone.yaml"), "stat", openNonBlocking,
			syscall.ENOENT, false,
		},
		{"the open fails", locked, "open", openNonBlocking, syscall.EACCES, true},
		{"the handle stat fails", readable, "stat", closedHandle, os.ErrClosed, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.needsUser && os.Geteuid() == 0 {
				t.Skip("root opens a mode 000 file")
			}
			_, err := readFile(testCase.path, testCase.open)
			want := "read config: " + testCase.step + ": " + testCase.cause.Error()
			if err == nil || err.Error() != want || !errors.Is(err, testCase.cause) {
				t.Fatalf("readFile() error = %v, want %q wrapping %v", err, want, testCase.cause)
			}
		})
	}
}
