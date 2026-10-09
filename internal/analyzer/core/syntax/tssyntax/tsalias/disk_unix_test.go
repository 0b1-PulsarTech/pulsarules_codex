//go:build unix

package tsalias

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDiskReader_FIFO(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "tsconfig.json"), 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	done := make(chan bool)
	go func() {
		_, found := DiskReader(
			root,
			func(string) ([]byte, bool) { return nil, false },
		)(
			"tsconfig.json",
		)
		done <- found
	}()
	select {
	case found := <-done:
		if found {
			t.Fatal("fifo was read")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("read blocked on a fifo")
	}
}
