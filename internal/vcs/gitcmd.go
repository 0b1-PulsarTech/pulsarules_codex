package vcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
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
	full := append([]string{"-C", dir}, args...)
	result, err := execx.Run(context.Background(), execx.Command{
		Name:    "git",
		Args:    full,
		Timeout: gitTimeout,
		// why: isEmptyRepoError matches an English git message, and git
		// translates its diagnostics when the locale has them installed.
		// Pinning LC_ALL keeps stderr in the language that match was
		// written against.
		Env: []string{"LC_ALL=C"},
	})
	if err != nil {
		// why: unwrap to execx.Error's own cause/stderr instead of execx.Error
		// itself, so gitError.Error() keeps the pre-migration message shape.
		cause, stderr := err, ""
		var execErr *execx.Error
		if errors.As(err, &execErr) {
			cause, stderr = execErr.Err, execErr.Stderr
		}
		return "", &gitError{args: full, stderr: stderr, err: cause}
	}
	return strings.TrimRight(result.Stdout, "\n"), nil
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
