//go:build unix

package disk

import (
	"context"
	"errors"
	"os"
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
		if !errors.Is(err, errNotRegular) {
			t.Fatalf("ReadBounded() error = %v, want %v", err, errNotRegular)
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
		file, err := nonBlocking(os.OpenFile)(path)
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
			t.Fatalf("nonBlocking() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("nonBlocking() still waiting after %v for a writer", deadline)
	}
}

func TestReadFileFailsWhenTheHandleCannotBeRead(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ok")
	writeFile(t, path, 3)
	open := func(name string) (*os.File, error) { return os.OpenFile(name, os.O_WRONLY, 0) }
	_, err := readFile(path, os.Stat, open, testLimit)
	if !errors.Is(err, syscall.EBADF) {
		t.Fatalf("readFile() error = %v, want %v", err, syscall.EBADF)
	}
}

func TestReadBoundedInRefusesALinkSwappedInAfterTheStat(t *testing.T) {
	t.Parallel()

	const rounds = 20000
	base := t.TempDir()
	inside := filepath.Join(base, "project")
	if err := os.Mkdir(inside, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(base, "secret"), 3)
	writeFile(t, filepath.Join(inside, "plain"), 2)
	root, err := os.OpenRoot(inside)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	target := filepath.Join(inside, "file")
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
			_ = os.Remove(target + ".new")
			if round%2 == 0 {
				_ = os.Symlink(filepath.Join(base, "secret"), target+".new")
			} else {
				_ = os.Link(filepath.Join(inside, "plain"), target+".new")
			}
			_ = os.Rename(target+".new", target)
		}
	}()
	defer func() {
		close(stop)
		<-swapped
	}()
	for range rounds {
		content, readErr := ReadBoundedIn(root, "file", testLimit)
		if readErr == nil && len(content) == 3 {
			t.Fatal("ReadBoundedIn() read a file outside the root")
		}
	}
}
