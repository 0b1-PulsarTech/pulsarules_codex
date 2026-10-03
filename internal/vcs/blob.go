package vcs

import "fmt"

type BlobReader interface {
	HeadBlob(path string) (string, error)
	StagedBlob(path string) (string, error)
}

func (repo *repository) HeadBlob(path string) (string, error) {
	out, err := runGit(repo.root, "show", "HEAD:"+path)
	if err != nil {
		return "", fmt.Errorf("read HEAD blob %s: %w", path, err)
	}
	return out, nil
}

func (repo *repository) StagedBlob(path string) (string, error) {
	out, err := runGit(repo.root, "show", ":"+path)
	if err != nil {
		return "", fmt.Errorf("read staged blob %s: %w", path, err)
	}
	return out, nil
}
