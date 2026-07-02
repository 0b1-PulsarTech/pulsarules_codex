package vcs

import (
	"errors"
	"fmt"
)

var ErrNoRepository = errors.New("vcs: not a git repository")

type Repository interface {
	Root() string
	WorktreeStatus() (Status, error)
}

type repository struct {
	root string
}

var _ Repository = (*repository)(nil)

//nolint:ireturn // factory constructor returns the consumer-declared interface
func Open(dir string) (Repository, error) {
	root, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("open repository: %w: %w", ErrNoRepository, err)
	}
	if root == "" {
		return nil, fmt.Errorf("open repository: %w: empty toplevel", ErrNoRepository)
	}
	return &repository{root: root}, nil
}

func (repo *repository) Root() string {
	return repo.root
}
