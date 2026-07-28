package cliopts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// BaseDir resolves the install base directory (the project root, or the home dir
// for --global) from the install flags.
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
		return opts.Project, nil
	default:
		return "", errors.New("install requires --global or --project PATH")
	}
}

// InstallDest resolves the Claude skills destination (<base>/.claude/skills).
func (opts *Options) InstallDest() (string, error) {
	projectDir, err := opts.BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, ".claude", "skills"), nil
}

// Targets resolves the requested install targets, defaulting to claude. Order is
// preserved and duplicates are dropped; runInstall validates each name against
// the target.Registry.
func (opts *Options) Targets() []string {
	if len(opts.Target) == 0 {
		return []string{defaultTarget}
	}
	targets := make([]string, 0, len(opts.Target))
	for _, name := range opts.Target {
		if !slices.Contains(targets, name) {
			targets = append(targets, name)
		}
	}
	return targets
}

// IsHelp reports whether the command requests the usage banner.
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
