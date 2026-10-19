package sandbox

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

var (
	errMisattributed = errors.New("reply is for another file")
	errNested        = errors.New("this process is already a parser child")
	errGreeting      = errors.New("parser sent no valid greeting")
)

type Runner struct {
	memory uint64
	wall   time.Duration
}

func NewRunner(memory uint64, wall time.Duration) Runner {
	return Runner{memory: memory, wall: wall}
}

type Outcome struct {
	File *syntax.File
	Skip string
}

func (runner Runner) spawn(ctx context.Context) (*worker, error) {
	const envMarker = "PULSARULES_TSPROC"
	if os.Getenv(envMarker) != "" {
		return nil, errNested
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate executable: %w", err)
	}
	args := []string{childArg, strconv.FormatUint(runner.memory, 10)}
	workCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(workCtx, exe, args...) //nolint:gosec // Re-executes this binary.
	cmd.WaitDelay = time.Second
	process.KillGroupOnCancel(cmd)
	cmd.Env = append(os.Environ(), envMarker+"=1")
	stderr := &tail{}
	cmd.Stderr = stderr
	stdin, stdout, err := openPipes(cmd)
	if err != nil {
		cancel()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start parser: %w", err)
	}
	return &worker{
		cmd: cmd, stdin: stdin, pipe: stdout, stdout: bufio.NewReader(stdout),
		stderr: stderr, cancel: cancel,
	}, nil
}

func openPipes(cmd *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("open parser stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, nil, fmt.Errorf("open parser stdout: %w", err)
	}
	return stdin, stdout, nil
}
