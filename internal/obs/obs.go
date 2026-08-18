package obs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// ErrInvalidLevel indicates a Config.Level value log/slog does not recognize.
var ErrInvalidLevel = errors.New("invalid log level")

const defaultMaxBytes = 256 * 1024

// Config selects where and whether hook telemetry is written. obs owns no
// host layout of its own - a host-neutral default log path would still bake
// in someone's directory convention - so both fields must be supplied by the
// caller for logging to activate at all.
type Config struct {
	Level    string // "", debug, info, warn, error - empty disables logging entirely
	Path     string // full log file path; empty disables logging even if Level is set
	MaxBytes int64  // defaults to 256 KiB
}

// New returns a logger that writes JSON records at cfg.Level, and a Closer
// for the underlying file. Disabled by default and free: empty Level or
// Path discards everything and Closer no-ops, so nothing is created or
// written unexpectedly. A set Level and Path truncate an oversized log to
// its tail before opening for append.
func New(cfg Config) (*slog.Logger, io.Closer, error) {
	if cfg.Level == "" || cfg.Path == "" {
		return slog.New(slog.DiscardHandler), noopCloser{}, nil
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, fmt.Errorf("parse log level: %w", err)
	}

	path := cfg.Path
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	if err = os.MkdirAll(filepath.Dir(path), fsperm.DirPrivate); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	if err = truncateToTail(path, maxBytes); err != nil {
		return nil, nil, fmt.Errorf("truncate log file: %w", err)
	}

	//nolint:gosec // path is caller-controlled, under .claude by convention.
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fsperm.FilePrivate)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})
	return slog.New(handler), file, nil
}

func parseLevel(raw string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return 0, fmt.Errorf("%q: %w", raw, ErrInvalidLevel)
	}
	return level, nil
}

// truncateToTail rewrites path to keep only its tail when it exceeds
// maxBytes, cutting at the first newline at or after maxBytes/2 counted back
// from the end so the surviving content still starts at a line boundary. A
// missing file is not an error - New goes on to create it fresh.
func truncateToTail(path string, maxBytes int64) error {
	//nolint:gosec // path is caller-controlled, under .claude by convention.
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read log file: %w", err)
	}
	if int64(len(raw)) <= maxBytes {
		return nil
	}

	keepFrom := max(len(raw)-int(maxBytes/2), 0)
	idx := bytes.IndexByte(raw[keepFrom:], '\n')
	if idx == -1 {
		// simplification: the kept region has no newline (its one line is
		// larger than maxBytes/2); keep the whole file rather than cut it
		// mid-line. Upgrade path: add a byte-budget line splitter if hook
		// records stop being one compact JSON line each.
		return nil
	}
	tail := raw[keepFrom+idx+1:]
	//nolint:gosec // path is caller-controlled, under .claude by convention.
	if err = os.WriteFile(path, tail, fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write truncated log file: %w", err)
	}
	return nil
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

var _ io.Closer = noopCloser{}
