package knowledge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Load returns the Index and the templates filesystem. With root == "" it reads
// the embedded snapshot; otherwise it reads <root>/knowledge from disk, so a
// rule edit can be exercised without rebuilding the binary.
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
	templates, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, nil, fmt.Errorf("sub templates: %w", err)
	}
	idx, err := readIndex(standards)
	if err != nil {
		return nil, nil, fmt.Errorf("read index: %w", err)
	}
	return idx, templates, nil
}

func loadDisk(dir string) (*Index, fs.FS, error) {
	standards := os.DirFS(filepath.Join(dir, "standards"))
	templates := os.DirFS(filepath.Join(dir, "templates"))
	idx, err := readIndex(standards)
	if err != nil {
		return nil, nil, fmt.Errorf("read index: %w", err)
	}
	return idx, templates, nil
}
