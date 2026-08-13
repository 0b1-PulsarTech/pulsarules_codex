package cliopts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/slicesx"
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
	return slicesx.Dedupe(opts.Target)
}

// ExplicitTargets returns the --target values the user actually passed,
// deduped and order-preserved, with no default applied. Uninstall must tell
// "nothing passed" apart from "claude explicitly asked for": its contract
// acts on every target it can detect on disk (target.Registry.DetectTargets),
// never narrowing to Targets' "claude only" default.
func (opts *Options) ExplicitTargets() []string {
	return slicesx.Dedupe(opts.Target)
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
	return slicesx.Dedupe(names)
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
