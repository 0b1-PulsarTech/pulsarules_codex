//go:build unix

package disk

import (
	"context"
	"errors"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadBoundedOnAFifoFailsInsteadOfBlocking(t *testing.T) {
	t.Parallel()

	const deadline = 5 * time.Second
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := ReadBounded(path, testLimit)
		done <- err
	}()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case err := <-done:
		if !errors.Is(err, ErrNotRegular) {
			t.Fatalf("ReadBounded() error = %v, want %v", err, ErrNotRegular)
		}
	case <-ctx.Done():
		t.Fatalf("ReadBounded() still blocked on a fifo after %v", deadline)
	}
}

func TestOpenNonBlockingDoesNotWaitForAWriter(t *testing.T) {
	t.Parallel()

	const deadline = 5 * time.Second
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
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
