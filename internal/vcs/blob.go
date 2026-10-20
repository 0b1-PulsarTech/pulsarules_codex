package vcs

import "fmt"

func (repo *Repository) Blob(side Side, path string) (string, error) {
	ref := "HEAD:" + path
	if side == Staged {
		ref = ":" + path
	}
	out, err := runGit(repo.root, "show", ref)
	if err != nil {
		return "", fmt.Errorf("read %s blob %s: %w", side, path, err)
	}
	return out, nil
}
