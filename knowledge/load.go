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

//go:embed templates/docs templates/hooks templates/skills
var templatesFS embed.FS

func Load(root string) (*Index, templates.Set, error) {
	if root == "" {
		return loadEmbedded()
	}
	return loadDisk(filepath.Join(root, "knowledge"))
}

func loadEmbedded() (*Index, templates.Set, error) {
	standards, err := fs.Sub(standardsFS, "standards")
	if err != nil {
		return nil, templates.Set{}, fmt.Errorf("sub standards: %w", err)
	}
	assets, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, templates.Set{}, fmt.Errorf("sub templates: %w", err)
	}
	idx, err := readIndex(standards)
	if err != nil {
		return nil, templates.Set{}, fmt.Errorf("read index: %w", err)
	}
	return idx, templates.New(assets), nil
}

func loadDisk(dir string) (*Index, templates.Set, error) {
	standards := os.DirFS(filepath.Join(dir, "standards"))
	idx, err := readIndex(standards)
	if err != nil {
		return nil, templates.Set{}, fmt.Errorf("read index: %w", err)
	}
	return idx, templates.New(os.DirFS(filepath.Join(dir, "templates"))), nil
}
