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
	// A hung git (e.g. a credential prompt) must not block the pipeline.
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	full := append([]string{"-C", dir}, args...)
	//nolint:gosec // Fixed git subcommands over an operator-supplied local path, never user input.
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &gitError{args: full, stderr: strings.TrimSpace(stderr.String()), err: err}
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

func isEmptyRepoError(err error) bool {
	var gitErr *gitError
	if !errors.As(err, &gitErr) {
		return false
	}
	return strings.Contains(gitErr.stderr, "does not have any commits yet")
}
