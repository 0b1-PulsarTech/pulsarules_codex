package version

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func (opts *Options) BaseDir() (string, error) {
	switch {
	case opts.Global && opts.Project != "":
		return "", errors.New("--global and --project are mutually exclusive")
	case opts.Global:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home: %w", err)
		}
		return home, nil
	case opts.Project != "":
		// A bare path: each target layers its own directory underneath.
		return opts.Project, nil
	default:
		return "", errors.New("install requires --global or --project PATH")
	}
}

func (opts *Options) Targets() []string {
	if len(opts.Target) == 0 {
		const defaultTarget = "claude"
		return []string{defaultTarget}
	}
	return dedupeStrings(opts.Target)
}

func (opts *Options) GitHookNames() []string {
	raw := strings.Split(opts.GitHooks, ",")
	names := make([]string, 0, len(raw))
	for _, name := range raw {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return dedupeStrings(names)
}

func dedupeStrings(raw []string) []string {
	targets := make([]string, 0, len(raw))
	for _, name := range raw {
		if !slices.Contains(targets, name) {
			targets = append(targets, name)
		}
	}
	return targets
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
