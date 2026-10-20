package vcs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func runGit(dir string, args ...string) (string, error) {
	const gitTimeout = 2 * time.Second
	// A hung git (e.g. a credential prompt) must not block the pipeline.
	return runGitWithin(gitTimeout, dir, args...)
}

func runGitWithin(timeout time.Duration, dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	// Shell out to git: this package must match git's own status, rename and listing output.
	result, err := process.Run(context.Background(), process.Command{
		Name:    "git",
		Args:    full,
		Timeout: timeout,
		// Git translates diagnostics per locale; LC_ALL=C keeps stderr in
		// the language isEmptyRepoError's match was written against.
		Env: []string{"LC_ALL=C"},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(result.Stdout, "\n"), nil
}

func isEmptyRepoError(err error) bool {
	execErr, ok := errors.AsType[*process.Error](err)
	return ok && strings.Contains(execErr.Stderr, "does not have any commits yet")
}
