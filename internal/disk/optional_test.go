package disk

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const testLimit = 16

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestReadBounded(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prepare func(t *testing.T, dir string) string
		want    int
		wantErr error
	}{
		{"a small file", func(t *testing.T, dir string) string {
			t.Helper()
			path := filepath.Join(dir, "small")
			writeFile(t, path, 3)
			return path
		}, 3, nil},
		{"a file at the limit", func(t *testing.T, dir string) string {
			t.Helper()
			path := filepath.Join(dir, "exact")
			writeFile(t, path, testLimit)
			return path
		}, testLimit, nil},
		{"a file past the limit", func(t *testing.T, dir string) string {
			t.Helper()
			path := filepath.Join(dir, "big")
			writeFile(t, path, 10*testLimit)
			return path
		}, testLimit + 1, nil},
		{"a directory", func(_ *testing.T, dir string) string { return dir }, 0, ErrNotRegular},
		{"a missing file", func(_ *testing.T, dir string) string {
			return filepath.Join(dir, "nope")
		}, 0, fs.ErrNotExist},
		{"a symlink to a device", func(t *testing.T, dir string) string {
			t.Helper()
			path := filepath.Join(dir, "link")
			if err := os.Symlink(os.DevNull, path); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			return path
		}, 0, ErrNotRegular},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			content, err := ReadBounded(testCase.prepare(t, t.TempDir()), testLimit)
			if !errors.Is(err, testCase.wantErr) || len(content) != testCase.want {
				t.Fatalf("ReadBounded() = %d bytes, %v; want %d, %v",
					len(content), err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestReadFileOpensOnlyRegularFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regular := filepath.Join(dir, "ok")
	writeFile(t, regular, 3)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(os.DevNull, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	testCases := []struct {
		name       string
		path       string
		wantOpened int
	}{
		{"a directory", dir, 0},
		{"a symlink to a device", link, 0},
		{"a missing file", filepath.Join(dir, "nope"), 0},
		{"a regular file", regular, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opened := 0
			_, _ = readFile(testCase.path, func(name string) (*os.File, error) {
				opened++
				return openNonBlocking(name)
			}, testLimit)
			if opened != testCase.wantOpened {
				t.Fatalf("readFile() opened %d times, want %d", opened, testCase.wantOpened)
			}
		})
	}
}

func TestReadFileChecksTheOpenedHandle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "ok")
	writeFile(t, path, 3)
	testCases := []struct {
		name    string
		open    opener
		wantErr error
	}{
		{"swapped for a directory", func(string) (*os.File, error) {
			return os.Open(dir)
		}, ErrNotRegular},
		{"swapped for a device", func(string) (*os.File, error) {
			return os.Open(os.DevNull)
		}, ErrNotRegular},
		{"the open fails", func(string) (*os.File, error) {
			return nil, fs.ErrPermission
		}, fs.ErrPermission},
		{"the handle is closed", func(name string) (*os.File, error) {
			file, err := os.Open(name)
			if err != nil {
				return nil, err
			}
			_ = file.Close()
			return file, nil
		}, os.ErrClosed},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := readFile(path, testCase.open, testLimit)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("readFile() error = %v, want %v", err, testCase.wantErr)
			}
		})
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
