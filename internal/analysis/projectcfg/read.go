package projectcfg

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

var errNotRegular = errors.New("read config: not a regular file")

type opener func(path string) (*os.File, error)

func readBounded(path string) ([]byte, error) {
	return readFile(path, openNonBlocking)
}

func openNonBlocking(path string) (*os.File, error) {
	// Non-blocking, so opening a pipe never waits for a writer.
	flags := os.O_RDONLY | syscall.O_NONBLOCK
	//nolint:gosec // G304: the path is the project config file or the operator's --config.
	file, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, readFault(err)
	}
	return file, nil
}

func readFault(err error) error {
	// The display name already shows the file, and an absolute path could fill the line.
	if pathErr, isPath := errors.AsType[*fs.PathError](err); isPath {
		return fmt.Errorf("read config: %s: %w", pathErr.Op, pathErr.Err)
	}
	return fmt.Errorf("read config: %w", err)
}

func readFile(path string, open opener) ([]byte, error) {
	// Refused before opening, so no device or pipe is ever opened.
	listed, err := os.Stat(path)
	if err != nil {
		return nil, readFault(err)
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
		return nil, readFault(err)
	}
	if refused(target.Mode()) {
		return nil, errNotRegular
	}
	return readCapped(file)
}

func refused(mode fs.FileMode) bool {
	const blocking = fs.ModeDir | fs.ModeNamedPipe | fs.ModeDevice |
		fs.ModeCharDevice | fs.ModeSocket
	// A Windows reparse point is irregular but readable, so only the blocking kinds are refused.
	return mode&blocking != 0
}

func readCapped(source io.Reader) ([]byte, error) {
	// One byte past the cap is enough for Parse to reject the file without reading all of it.
	content, err := io.ReadAll(io.LimitReader(source, config.MaxFileBytes+1))
	if err != nil {
		return nil, readFault(err)
	}
	return content, nil
}
