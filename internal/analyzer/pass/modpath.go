package pass

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNoModuleDirective = errors.New("go.mod has no module directive")

func ModulePath(projectDir string) (string, error) {
	const goModModulePrefix = "module "

	path := filepath.Join(projectDir, "go.mod")
	//nolint:gosec // G304: projectDir is the analyzed project root, not user input.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
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
