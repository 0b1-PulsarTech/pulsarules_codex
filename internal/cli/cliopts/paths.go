package cliopts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// Targets resolves the requested install targets, defaulting to claude. Order is
// preserved and duplicates are dropped; install.Run validates each name against
// the target.Registry.
func (opts *Options) Targets() []string {
	if len(opts.Target) == 0 {
		return []string{defaultTarget}
	}
	return dedupeStrings(opts.Target)
}

// ExplicitTargets returns the --target values the user actually passed,
// deduped and order-preserved, with no default applied. Uninstall must tell
// "nothing passed" apart from "claude explicitly asked for": its contract is
// to act on every target it can detect on disk (see target.Registry.
// DetectTargets), never to silently narrow to install's "claude only"
// default the way Targets does.
func (opts *Options) ExplicitTargets() []string {
	return dedupeStrings(opts.Target)
}

// GitHookNames parses --git-hooks into the ordered, deduped hook names to
// install.
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

// dedupeStrings drops duplicate names while preserving order.
func dedupeStrings(raw []string) []string {
	targets := make([]string, 0, len(raw))
	for _, name := range raw {
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
