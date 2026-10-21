package version

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
)

func (opts *Options) BaseDir() (string, error) {
	var dir string
	switch {
	case opts.Global && opts.Project != "":
		return "", errors.New("--global and --project are mutually exclusive")
	case opts.Global:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home: %w", err)
		}
		dir = home
	case opts.Project != "":
		// A bare path: each target layers its own directory underneath.
		dir = opts.Project
	default:
		return "", errors.New("install requires --global or --project PATH")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("absolute %q: %w", dir, err)
	}
	return abs, nil
}

func (opts *Options) Targets() []string {
	if len(opts.Target) == 0 {
		const defaultTarget = "claude"
		return []string{defaultTarget}
	}
	return ordered.Dedupe(opts.Target)
}

func (opts *Options) ExplicitTargets() []string {
	return ordered.Dedupe(opts.Target)
}

func IsHelp(command string) bool {
	switch command {
	case "help", "-h", "--help":
		return true
	}
	return false
}

func defaultOut(parts ...string) string {
	return filepath.Join(append([]string{"."}, parts...)...)
}
