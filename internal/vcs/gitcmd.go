package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout bounds every git invocation so a hung process (e.g. a
// credential prompt on a misconfigured remote) cannot block the pipeline.
const gitTimeout = 2 * time.Second

// gitError carries stderr alongside a failed git invocation, so
// isEmptyRepoError can recognize a specific git message without
// string-matching the wrapped error's full text.
type gitError struct {
	args   []string
	stderr string
	err    error
}

func (e *gitError) Error() string {
	return fmt.Sprintf("git %s: %v: %s", strings.Join(e.args, " "), e.err, e.stderr)
}

func (e *gitError) Unwrap() error {
	return e.err
}

// runGit runs git with args in dir under a timeout and returns trimmed
// stdout. Every git call in this package goes through this one helper.
func runGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	full := append([]string{"-C", dir}, args...)
	//nolint:gosec // dir and args are fixed subcommands over an operator-supplied local path, never user input.
	cmd := exec.CommandContext(ctx, "git", full...)
	// why: isEmptyRepoError matches an English git message, and git translates
	// its diagnostics when the locale has them installed. Pinning LC_ALL keeps
	// stderr in the language that match was written against.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &gitError{args: full, stderr: strings.TrimSpace(stderr.String()), err: err}
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// isEmptyRepoError reports whether err came from a log/diff command run
// against a repository with no commits yet - a normal state, not a failure.
func isEmptyRepoError(err error) bool {
	var gitErr *gitError
	if !errors.As(err, &gitErr) {
		return false
	}
	return strings.Contains(gitErr.stderr, "does not have any commits yet")
}
