package vcs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNoRepository is returned by Open when dir is not inside a git
// repository, so callers can tell that condition apart from a real failure
// and stay quiet rather than nagging on every turn.
var ErrNoRepository = errors.New("vcs: not a git repository")

// Repository is the read-only git access the governance pipeline needs.
type Repository interface {
	// Root returns the worktree's root directory.
	Root() string
	// HeadSubject returns the subject line of HEAD's commit message, or
	// ("", nil) when the repository has no commits yet.
	HeadSubject() (string, error)
	// HeadAuthorEpoch returns HEAD's author-date Unix epoch, or (0, false,
	// nil) when the repository has no commits yet.
	HeadAuthorEpoch() (int64, bool, error)
	// RecentSubjects returns up to limit recent commit subjects, newest
	// first, or (nil, nil) when the repository has no commits yet.
	RecentSubjects(limit int) ([]string, error)
	// CurrentBranch returns the checked-out branch name, or "" on a detached
	// HEAD - a state that names no branch at all.
	CurrentBranch() (string, error)
	// WorktreeStatus reports the paths that differ from HEAD.
	WorktreeStatus() (Status, error)
	// StagedRenames pairs staged deletions with staged additions that look
	// like the same file moved, scored at or above minScore.
	StagedRenames(minScore int) ([]Rename, error)
	// StagedDiff returns the unified diff body for a single staged path (git
	// diff --cached -- path).
	StagedDiff(path string) (string, error)
	// StagedRenameDiff returns the unified diff body for a staged rename
	// pair, scored at or above minScore, so a caller sees the delta within
	// the move rather than the new path's whole content.
	StagedRenameDiff(oldPath, newPath string, minScore int) (string, error)
}

// why: every method shells out to git rather than reading .git's object
// store directly, so it sees exactly what the git binary sees. A go-git
// backend was rejected: it missed pack files not named pack-*, and
// `git maintenance` had written loose-<hash>.pack objects; it reported 219
// changed files where git reported 19, diverging only against a real repo.
type repository struct {
	root string
}

var _ Repository = (*repository)(nil)

// Open opens the git repository containing dir, walking up through parent
// directories (including resolving a linked worktree to the main
// repository's root). Returns ErrNoRepository when dir is not inside one.
//
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

func (r *repository) Root() string {
	return r.root
}

// CurrentBranch returns the checked-out branch name.
//
// why: symbolic-ref FAILS on a detached HEAD rather than inventing a name, and
// that failure is the answer - "" means no branch, not an error to report.
func (r *repository) CurrentBranch() (string, error) {
	branch, err := runGit(r.root, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", nil //nolint:nilerr // a detached HEAD names no branch, not a failure.
	}
	return branch, nil
}

// HeadSubject returns the subject line of HEAD's commit message, or
// ("", nil) when the repository has no commits yet.
func (r *repository) HeadSubject() (string, error) {
	subject, err := runGit(r.root, "log", "-1", "--format=%s")
	if err != nil {
		if isEmptyRepoError(err) {
			return "", nil
		}
		return "", fmt.Errorf("read head subject: %w", err)
	}
	return subject, nil
}

// HeadAuthorEpoch returns HEAD's author-date Unix epoch, or (0, false, nil)
// when the repository has no commits yet.
func (r *repository) HeadAuthorEpoch() (int64, bool, error) {
	out, err := runGit(r.root, "log", "-1", "--format=%at")
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

// RecentSubjects returns up to limit recent commit subjects, newest first,
// or (nil, nil) when the repository has no commits yet.
func (r *repository) RecentSubjects(limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}

	out, err := runGit(r.root, "log", "--format=%s", "-n", strconv.Itoa(limit))
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
