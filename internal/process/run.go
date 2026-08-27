package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

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

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: %v: %s", e.Name, strings.Join(e.Args, " "), e.Err, e.Stderr)
}

func (e *Error) Unwrap() error { return e.Err }

type Command struct {
	Name    string
	Args    []string
	Dir     string
	Timeout time.Duration
	Env     []string
}

func Run(ctx context.Context, cmd Command) (Result, error) {
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	//nolint:gosec // G204: fixed subcommands built by callers, not user-controlled.
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	const ioWaitGrace = 5 * time.Second
	// A killed process can leave a child of its own holding the same
	// stdout/stderr pipe open; without WaitDelay, exec.Cmd.Wait blocks on that
	// pipe closing, silently defeating Command.Timeout (see os/exec's docs).
	execCmd.WaitDelay = ioWaitGrace
	KillGroupOnCancel(execCmd)
	if len(cmd.Env) > 0 {
		// Extra pairs extend the inherited environment instead of replacing it.
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	runErr := execCmd.Run()
	// Stdout stays raw for callers to trim; Stderr is pre-trimmed for a readable message.
	result := Result{Stdout: stdout.String(), Stderr: strings.TrimSpace(stderr.String())}
	if runErr != nil {
		return result, &Error{Name: cmd.Name, Args: cmd.Args, Stderr: result.Stderr, Err: runErr}
	}
	return result, nil
}
