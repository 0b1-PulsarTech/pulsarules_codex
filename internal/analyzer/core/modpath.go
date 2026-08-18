package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoModuleDirective is returned by ModulePath when projectDir's
// go.mod exists and is readable but declares no "module" line.
var ErrNoModuleDirective = errors.New("go.mod has no module directive")

// goModModulePrefix is the go.mod directive line prefix that names the
// module path; the value starts right after it.
const goModModulePrefix = "module "

// ModulePath returns the module path declared by projectDir's go.mod.
//
// why: reads the project's own go.mod instead of a hardcoded constant, so
// buildGraph's local-import filter keys off the target project - a
// hardcoded module made every other target's import graph come back empty.
func ModulePath(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "go.mod")
	//nolint:gosec // G304: projectDir is the analyzed project root, not user input.
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), goModModulePrefix)
		if !ok {
			continue
		}
		if mod := strings.TrimSpace(rest); mod != "" {
			return mod, nil
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return "", fmt.Errorf("scan go.mod: %w", scanErr)
	}
	return "", fmt.Errorf("%s: %w", path, ErrNoModuleDirective)
}
