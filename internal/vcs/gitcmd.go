package vcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

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

func runGit(dir string, args ...string) (string, error) {
	const gitTimeout = 2 * time.Second
	full := append([]string{"-C", dir}, args...)
	// Shell out to git: this package must match git's own status, rename and listing output.
	result, err := execx.Run(context.Background(), execx.Command{
		Name:    "git",
		Args:    full,
		Timeout: gitTimeout,
		// Git translates diagnostics per locale; LC_ALL=C keeps stderr in
		// the language isEmptyRepoError's match was written against.
		Env: []string{"LC_ALL=C"},
	})
	if err != nil {
		// Unwrap to execx.Error's own cause and stderr so gitError.Error() stays terse.
		cause, stderr := err, ""
		if execErr, ok := errors.AsType[*execx.Error](err); ok {
			cause, stderr = execErr.Err, execErr.Stderr
		}
		return "", &gitError{args: full, stderr: stderr, err: cause}
	}
	return strings.TrimRight(result.Stdout, "\n"), nil
}

func isEmptyRepoError(err error) bool {
	var gitErr *gitError
	if !errors.As(err, &gitErr) {
		return false
	}
	return strings.Contains(gitErr.stderr, "does not have any commits yet")
}
