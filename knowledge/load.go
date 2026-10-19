package knowledge

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

//go:embed standards
var standardsFS embed.FS

func Load(root string) (*Index, fs.FS, error) {
	if root == "" {
		return loadEmbedded()
	}
	return loadDisk(filepath.Join(root, "knowledge"))
}

func loadEmbedded() (*Index, fs.FS, error) {
	standards, err := fs.Sub(standardsFS, "standards")
	if err != nil {
		return nil, nil, fmt.Errorf("sub standards: %w", err)
	}
	idx, err := readIndex(standards)
	if err != nil {
		return nil, nil, fmt.Errorf("read index: %w", err)
	}
	return idx, templates.Embedded().FS(), nil
}

func loadDisk(dir string) (*Index, fs.FS, error) {
	standards := os.DirFS(filepath.Join(dir, "standards"))
	idx, err := readIndex(standards)
	if err != nil {
		return nil, nil, fmt.Errorf("read index: %w", err)
	}
	return idx, templates.Dir(filepath.Join(dir, "templates")).FS(), nil
}
