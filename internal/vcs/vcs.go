package vcs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrNoRepository = errors.New("vcs: not a git repository")

type Repository struct {
	root string
}

func Open(dir string) (*Repository, error) {
	root, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("open repository: %w: %w", ErrNoRepository, err)
	}
	if root == "" {
		return nil, fmt.Errorf("open repository: %w: empty toplevel", ErrNoRepository)
	}
	return &Repository{root: root}, nil
}

func (repo *Repository) Root() string {
	return repo.root
}

func (repo *Repository) CurrentBranch() (string, error) {
	branch, err := runGit(repo.root, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("read current branch: %w", err)
	}
	return branch, nil
}

func (repo *Repository) HeadAuthorEpoch() (int64, bool, error) {
	out, err := runGit(repo.root, "log", "-1", "--format=%at")
	if err != nil {
		if isEmptyRepoError(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read head author epoch: %w", err)
	}
	epoch, err := strconv.ParseInt(out, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse head author epoch %q: %w", out, err)
	}
	return epoch, true, nil
}

func (repo *Repository) RecentSubjects(limit int) ([]string, error) {
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
