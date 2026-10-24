package flags

import (
	"errors"
	"fmt"
	"path/filepath"
)

type Scope struct {
	Global  bool
	Project string
	Home    string
	// ConfigDir is Claude Code's own CLAUDE_CONFIG_DIR, honoured in global scope only.
	ConfigDir string
	CodexHome string
}

func (scope Scope) WithEnv(env Environment) Scope {
	scope.Home, scope.ConfigDir, scope.CodexHome = env.Home, env.ConfigDir, env.CodexHome
	return scope
}

func BaseDir(scope Scope) (string, error) {
	var dir string
	switch {
	case scope.Global && scope.Project != "":
		return "", errors.New("--global and --project are mutually exclusive")
	case scope.Global:
		if scope.Home == "" {
			return "", errors.New("resolve home: home directory is unknown")
		}
		dir = scope.Home
	case scope.Project != "":
		// A bare path: each target layers its own directory underneath.
		dir = scope.Project
	default:
		return "", errors.New("install requires --global or --project PATH")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("absolute %q: %w", dir, err)
	}
	return abs, nil
}

func UserClaudeDir(scope Scope) (string, error) {
	dir := scope.ConfigDir
	if dir == "" && scope.Home != "" {
		dir = filepath.Join(scope.Home, ".claude")
	}
	if dir == "" {
		return "", nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("absolute %q: %w", dir, err)
	}
	return abs, nil
}

func UserCodexDir(scope Scope) (string, error) {
	dir := scope.CodexHome
	if dir == "" && scope.Home != "" {
		dir = filepath.Join(scope.Home, ".codex")
	}
	if dir == "" {
		return "", nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("absolute %q: %w", dir, err)
	}
	return abs, nil
}
