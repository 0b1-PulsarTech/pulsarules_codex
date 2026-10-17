package pass

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

var ErrNoModuleDirective = errors.New("go.mod has no module directive")

func ModulePath(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "go.mod")
	//nolint:gosec // G304: projectDir is the analyzed project root, not user input.
	src, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	modulePath := gomod.ModulePath(src)
	if modulePath == "" {
		return "", fmt.Errorf("%s: %w", path, ErrNoModuleDirective)
	}
	return modulePath, nil
}
