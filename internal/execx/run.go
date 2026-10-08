package execx

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

var ErrOutputExceeded = errors.New("output limit exceeded")

type Result struct {
	Stdout string
	Stderr string
}

type Error struct {
	Name   string
	Args   []string
	Stderr string
	Err    error
}

func (execErr *Error) Error() string {
	return fmt.Sprintf(
		"%s %s: %v: %s",
		execErr.Name, strings.Join(execErr.Args, " "), execErr.Err, execErr.Stderr,
	)
}

func (execErr *Error) Unwrap() error { return execErr.Err }

type Command struct {
	Name      string
	Args      []string
	Dir       string
	Timeout   time.Duration
	Env       []string
	MaxOutput int
}

func Run(ctx context.Context, cmd Command) (Result, error) {
	// A zero Timeout leaves the caller's own deadline in control.
	runCtx, cancel := boundedContext(ctx, cmd.Timeout)
	defer cancel()

	// The name and args come from this codebase's own callers (git, golangci-lint,
	// gopls) or from a project's own delegated command list, run without a shell.
	//nolint:gosec // G204: no shell; callers pass fixed subcommands or the project's own list.
	execCmd := exec.CommandContext(runCtx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	const ioWaitGrace = 5 * time.Second
	// A killed process can leave a child of its own holding the same
	// stdout/stderr pipe open; without WaitDelay, exec.Cmd.Wait blocks on that
	// pipe closing, silently defeating Command.Timeout (see os/exec's docs).
	execCmd.WaitDelay = ioWaitGrace
	wireProcessGroupKill(execCmd)
	if len(cmd.Env) > 0 {
		// Extra pairs extend the inherited environment instead of replacing it.
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	stdout := &cappedBuffer{limit: cmd.MaxOutput, onExceed: cancel}
	stderr := &cappedBuffer{limit: cmd.MaxOutput, onExceed: cancel}
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr

	runErr := execCmd.Run()
	// Stdout stays raw for callers to trim; Stderr is pre-trimmed for a readable message.
	result := Result{Stdout: stdout.buf.String(), Stderr: strings.TrimSpace(stderr.buf.String())}
	if stdout.exceeded || stderr.exceeded {
		runErr = fmt.Errorf("%d bytes: %w", cmd.MaxOutput, ErrOutputExceeded)
	} else if runErr != nil && runCtx.Err() != nil && !errors.Is(runErr, runCtx.Err()) {
		runErr = fmt.Errorf("%w: %w", runCtx.Err(), runErr)
	}
	if runErr != nil {
		return result, &Error{Name: cmd.Name, Args: cmd.Args, Stderr: result.Stderr, Err: runErr}
	}
	return result, nil
}

func boundedContext(
	ctx context.Context, timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

type cappedBuffer struct {
	buf      bytes.Buffer
	limit    int
	exceeded bool
	onExceed func()
}

func (capped *cappedBuffer) Write(chunk []byte) (int, error) {
	kept := chunk
	if capped.limit > 0 && len(chunk) > capped.limit-capped.buf.Len() {
		kept = chunk[:capped.limit-capped.buf.Len()]
		if !capped.exceeded {
			capped.exceeded = true
			capped.onExceed()
		}
	}
	// Never fails past the cap: a failed copy would leave the child blocked on the pipe.
	capped.buf.Write(kept)
	return len(chunk), nil
}
