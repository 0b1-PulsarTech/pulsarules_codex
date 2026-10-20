//go:build unix

package pathalias

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDiskReader_FIFO(t *testing.T) {
	t.Parallel()

	const deadline = 10 * time.Second
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "tsconfig.json"), 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	done := make(chan bool, 1)
	go func() {
		_, found := DiskReader(
			root,
			func(string) ([]byte, bool) { return nil, false },
		)(
			"tsconfig.json",
		)
		done <- found
	}()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case found := <-done:
		if found {
			t.Fatal("fifo was read")
		}
	case <-ctx.Done():
		t.Fatal("read blocked on a fifo")
	}
}

func TestDiskReader_FIFOSwappedInAfterTheStat(t *testing.T) {
	t.Parallel()

	const (
		deadline = 20 * time.Second
		rounds   = 20000
	)
	root := t.TempDir()
	target := filepath.Join(root, "tsconfig.json")
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "plain"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	stop := make(chan struct{})
	swapped := make(chan struct{})
	go func() {
		defer close(swapped)
		for round := 0; ; round++ {
			select {
			case <-stop:
				return
			default:
			}
			source := "plain"
			if round%2 == 0 {
				source = "pipe"
			}
			_ = os.Link(filepath.Join(root, source), target+".new")
			_ = os.Rename(target+".new", target)
		}
	}()
	defer func() {
		close(stop)
		<-swapped
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		read := DiskReader(root, func(string) ([]byte, bool) { return nil, false })
		for range rounds {
			read("tsconfig.json")
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("read blocked on a fifo swapped in after the stat")
	}
}
