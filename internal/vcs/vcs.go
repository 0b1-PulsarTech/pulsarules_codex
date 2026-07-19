package vcs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrNoRepository = errors.New("vcs: not a git repository")

type Repository interface {
	Root() string
	RecentSubjects(limit int) ([]string, error)
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

func (repo *repository) RecentSubjects(limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}

	out, err := runGit(repo.root, "log", "--format=%s", "-n", strconv.Itoa(limit))
	if err != nil {
		if isEmptyRepoError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read recent subjects: %w", err)
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
