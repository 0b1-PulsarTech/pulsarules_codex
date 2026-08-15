package arch

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// errNoModuleDirective is returned by resolveModulePath when projectDir's
// go.mod exists and is readable but declares no "module" line.
var errNoModuleDirective = errors.New("go.mod has no module directive")

// goModModulePrefix is the go.mod directive line prefix that names the
// module path; the value starts right after it.
const goModModulePrefix = "module "

// why: derives the module path from the analyzed project's own go.mod
// instead of a hardcoded constant, so buildGraph's local-import filter keys
// off the target project rather than only ever matching this tool's own
// module (which made every other target's import graph come back empty).
func resolveModulePath(projectDir string) (string, error) {
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
	return "", fmt.Errorf("%s: %w", path, errNoModuleDirective)
}
