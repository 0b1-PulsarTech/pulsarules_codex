package selfbin

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type seedFunc func(t *testing.T) (dst string, verify func(t *testing.T))

func TestCopyTo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seed    seedFunc
		wantErr bool
	}{
		{name: "copies to a nested destination", seed: seedNested},
		{name: "mkdir fails when the parent path is a file", seed: seedBlocked, wantErr: true},
		{name: "replaces the inode so a held descriptor keeps the old body", seed: seedHeld},
		{name: "the running binary is left untouched", seed: seedRunning},
		{name: "a symlink to the running binary is left untouched", seed: seedRunningLink},
		{name: "a symlinked destination replaces the link target", seed: seedLink},
		{name: "a dangling symlink fails", seed: seedDangling, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dst, verify := testCase.seed(t)
			err := copyTo(dst)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("copyTo: %v", err)
			}
			verify(t)
		})
	}
}

func seedNested(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "nested", "bin", "copy")
	return dst, func(t *testing.T) {
		t.Helper()
		requireExecutable(t, dst)
	}
}

func seedBlocked(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	return filepath.Join(blocker, "copy"), nil
}

func seedHeld(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "hook")
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatalf("seed dst: %v", err)
	}
	held, err := os.Open(dst)
	if err != nil {
		t.Fatalf("open dst: %v", err)
	}
	t.Cleanup(func() { _ = held.Close() })
	return dst, func(t *testing.T) {
		t.Helper()
		oldBody, readErr := io.ReadAll(held)
		if readErr != nil {
			t.Fatalf("read held fd: %v", readErr)
		}
		if string(oldBody) != "old" {
			t.Errorf("held fd saw %d bytes, want the old 3-byte body", len(oldBody))
		}
		requireExecutable(t, dst)
	}
}

func seedRunning(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	before, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("stat exe: %v", err)
	}
	return exe, func(t *testing.T) {
		t.Helper()
		after, statErr := os.Stat(exe)
		if statErr != nil {
			t.Fatalf("stat exe after: %v", statErr)
		}
		if !os.SameFile(before, after) || before.Size() != after.Size() {
			t.Error("running binary was modified")
		}
	}
}

func seedRunningLink(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	exe, verify := seedRunning(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return link, verify
}

func seedLink(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return link, func(t *testing.T) {
		t.Helper()
		stat, err := os.Lstat(link)
		if err != nil || stat.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("link must survive as a symlink: %v, %v", stat, err)
		}
		body, err := os.ReadFile(target)
		if err != nil || string(body) == "old" {
			t.Errorf("target not replaced: %d bytes, err %v", len(body), err)
		}
		requireExecutable(t, target)
	}
}

func seedDangling(t *testing.T) (string, func(t *testing.T)) {
	t.Helper()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return link, nil
}

func requireExecutable(t *testing.T, path string) {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if stat.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", stat.Mode().Perm())
	}
}

func TestInstallAndRemove(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "bin")
	if err := Install(dir); err != nil {
		t.Fatalf("Install: %v", err)
	}
	requireExecutable(t, filepath.Join(dir, Name))

	if err := Remove(dir); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, Name)); !os.IsNotExist(err) {
		t.Errorf("binary still present after Remove: %v", err)
	}
	if err := Remove(dir); err != nil {
		t.Errorf("Remove of an absent binary = %v, want nil", err)
	}
}

func TestInstall_Failure(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	err := Install(blocker)
	if err == nil || !strings.Contains(err.Error(), "copy installer binary") {
		t.Fatalf("Install(file as dir) err = %v, want a copy installer binary error", err)
	}
}

func TestRemove_Failure(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Name), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	err := Remove(dir)
	if err == nil || !strings.Contains(err.Error(), "remove installer binary") {
		t.Fatalf("Remove(read-only dir) err = %v, want a remove installer binary error", err)
	}
}
