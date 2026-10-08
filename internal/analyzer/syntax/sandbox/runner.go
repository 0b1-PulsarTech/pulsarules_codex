package sandbox

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

var (
	errMisattributed = errors.New("reply is for another file")
	errNested        = errors.New("this process is already a parser child")
	errGreeting      = errors.New("parser sent no valid greeting")
)

const EnvMarker = "PULSARULES_TSPROC"

type Runner struct {
	Exe    string
	Args   []string
	Memory uint64
	Wall   time.Duration
}

type Outcome struct {
	File *syntax.File
	Skip string
}

func (runner Runner) spawn(ctx context.Context) (*worker, error) {
	if os.Getenv(EnvMarker) != "" {
		return nil, errNested
	}
	exe := runner.Exe
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return nil, fmt.Errorf("locate executable: %w", err)
		}
	}
	args := append(slices.Clone(runner.Args), strconv.FormatUint(runner.Memory, 10))
	workCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(workCtx, exe, args...) //nolint:gosec // Re-executes this binary.
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), EnvMarker+"=1")
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
