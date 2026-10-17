package gomod

import (
	"fmt"

	"golang.org/x/mod/modfile"
)

func ModulePath(src []byte) string {
	return modfile.ModulePath(src)
}

func WorkUses(src []byte) ([]string, error) {
	file, err := modfile.ParseWork("go.work", src, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.work: %w", err)
	}
	paths := make([]string, 0, len(file.Use))
	for _, use := range file.Use {
		paths = append(paths, use.Path)
	}
	return paths, nil
}
