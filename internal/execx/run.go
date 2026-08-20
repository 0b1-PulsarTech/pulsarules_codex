package execx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Result holds a completed command's captured output. Stdout is the raw
// capture (a caller that needs a specific trim, e.g. a trailing newline
// only, applies it itself); Stderr is trimmed since every caller uses it
// only for a human-readable message.
type Result struct {
	Stdout string
	Stderr string
}

// Error is returned when a command fails to start, exits non-zero, or is
// killed by its timeout. It carries stderr and the failing invocation so a
// caller (e.g. a message match against a known failure) does not have to
// re-parse the wrapped error.
type Error struct {
	Name   string
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: %v: %s", e.Name, strings.Join(e.Args, " "), e.Err, e.Stderr)
}

func (e *Error) Unwrap() error { return e.Err }

// why: a killed process can leave a child of its own holding the same
// stdout/stderr pipe open; without WaitDelay, exec.Cmd.Wait blocks on that
// pipe closing, silently defeating Command.Timeout (see os/exec's docs).
const ioWaitGrace = 5 * time.Second

// Command describes one external-process invocation.
type Command struct {
	// Name is the binary to run, resolved on PATH unless it contains a slash.
	Name string
	// Args are passed to Name, in order.
	Args []string
	// Dir is the working directory; empty keeps the caller's own.
	Dir string
	// Timeout bounds the call. Zero leaves ctx's own deadline in control,
	// for a caller that already imposes one of its own.
	Timeout time.Duration
	// Env appends extra "K=V" pairs to the inherited environment; nil keeps
	// the default environment untouched.
	Env []string
}

// Run executes cmd and returns its captured stdout and stderr. A non-zero
// exit, a start failure, or a timeout all return a *Error wrapping the
// underlying cause.
func Run(ctx context.Context, cmd Command) (Result, error) {
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	// why: cmd.Name/Args come from this codebase's own callers (git,
	// golangci-lint, gopls, emojigen's git scan), never raw user input.
	//nolint:gosec // G204: fixed subcommands built by callers, not user-controlled.
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	execCmd.WaitDelay = ioWaitGrace
	if len(cmd.Env) > 0 {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: strings.TrimSpace(stderr.String())}
	if runErr != nil {
		return result, &Error{Name: cmd.Name, Args: cmd.Args, Stderr: result.Stderr, Err: runErr}
	}
	return result, nil
}
