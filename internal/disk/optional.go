package disk

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
)

var errNotRegular = errors.New("not a regular file")

type opener func(path string) (*os.File, error)

type lister func(path string) (fs.FileInfo, error)

func ReadBounded(path string, limit int64) ([]byte, error) {
	// At most limit+1 bytes, so a caller can tell a file over the limit from one at it.
	return readFile(path, os.Stat, nonBlocking(os.OpenFile), limit)
}

func ReadBoundedIn(root *os.Root, name string, limit int64) ([]byte, error) {
	return readFile(name, root.Stat, nonBlocking(root.OpenFile), limit)
}

func nonBlocking(openFile func(string, int, fs.FileMode) (*os.File, error)) opener {
	return func(path string) (*os.File, error) {
		// Non-blocking, so opening a pipe never waits for a writer.
		file, err := openFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, fmt.Errorf("open bounded: %w", err)
		}
		return file, nil
	}
}

func readFile(path string, stat lister, open opener, limit int64) ([]byte, error) {
	// Refused before opening, so no device or pipe is ever opened.
	listed, err := stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat bounded: %w", err)
	}
	if refused(listed.Mode()) {
		return nil, errNotRegular
	}
	file, err := open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	// Checked again on the handle, which the path may have been swapped for since the stat.
	target, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat bounded: %w", err)
	}
	if refused(target.Mode()) {
		return nil, errNotRegular
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read bounded: %w", err)
	}
	return content, nil
}

func refused(mode fs.FileMode) bool {
	const blocking = fs.ModeDir | fs.ModeNamedPipe | fs.ModeDevice |
		fs.ModeCharDevice | fs.ModeSocket
	// A Windows reparse point is irregular but readable, so only the blocking kinds are refused.
	return mode&blocking != 0
}
