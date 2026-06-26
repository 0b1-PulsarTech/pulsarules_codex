package knowledge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func Load(root string) (*Index, error) {
	if root == "" {
		return loadEmbedded()
	}
	return loadDisk(filepath.Join(root, "knowledge"))
}

func loadEmbedded() (*Index, error) {
	standards, err := fs.Sub(standardsFS, "standards")
	if err != nil {
		return nil, fmt.Errorf("sub standards: %w", err)
	}
	idx, err := readIndex(standards)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return idx, nil
}

func loadDisk(dir string) (*Index, error) {
	standards := os.DirFS(filepath.Join(dir, "standards"))
	idx, err := readIndex(standards)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return idx, nil
}
