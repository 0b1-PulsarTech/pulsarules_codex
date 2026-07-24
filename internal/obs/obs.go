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

var ErrInvalidLevel = errors.New("invalid log level")

type Config struct {
	Level    string
	Path     string
	MaxBytes int64
}

func New(cfg Config) (*slog.Logger, io.Closer, error) {
	if cfg.Level == "" || cfg.Path == "" {
		return slog.New(slog.DiscardHandler), noopCloser{}, nil
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, fmt.Errorf("parse log level: %w", err)
	}

	const defaultMaxBytes = 256 * 1024

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

	//nolint:gosec // path is caller-controlled.
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

func truncateToTail(path string, maxBytes int64) error {
	//nolint:gosec // path is caller-controlled.
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

	// Cut at the next newline after maxBytes/2 so the tail starts on a line boundary.
	keepFrom := max(len(raw)-int(maxBytes/2), 0)
	idx := bytes.IndexByte(raw[keepFrom:], '\n')
	if idx == -1 {
		// simplification: no newline in the kept region means its one line
		// exceeds maxBytes/2; keep the whole file rather than cut mid-line.
		// Upgrade path: a byte-budget line splitter, if records outgrow one line.
		return nil
	}
	tail := raw[keepFrom+idx+1:]
	//nolint:gosec // path is caller-controlled.
	if err = os.WriteFile(path, tail, fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write truncated log file: %w", err)
	}
	return nil
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

var _ io.Closer = noopCloser{}
