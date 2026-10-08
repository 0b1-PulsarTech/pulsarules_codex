package tsproc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type worker struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	pipe    io.Closer
	stdout  *bufio.Reader
	stderr  *tail
	cancel  context.CancelFunc
	expired atomic.Bool
}

type exitStatus struct {
	code   int
	stderr string
}

func (status exitStatus) String() string {
	if status.stderr == "" {
		return fmt.Sprintf("exit %d", status.code)
	}
	return fmt.Sprintf("exit %d, stderr %s", status.code, status.stderr)
}

func (live *worker) greet(wall time.Duration) error {
	fired, err := live.within(wall, func() error {
		var greeting hello
		if err := readFrame(live.stdout, &greeting); err != nil {
			return fmt.Errorf("%w: %w", errGreeting, err)
		}
		if greeting.Protocol != protocol {
			return errGreeting
		}
		return nil
	})
	if err == nil && fired {
		return fmt.Errorf("%w: greeted after the %s time limit", errGreeting, wall)
	}
	return err
}

func (live *worker) within(wall time.Duration, run func() error) (bool, error) {
	timer := time.AfterFunc(wall, func() {
		live.expired.Store(true)
		live.cancel()
		_ = live.pipe.Close()
	})
	err := run()
	return !timer.Stop(), err
}

func (live *worker) exchange(request Request, wall time.Duration) (*syntax.File, bool, error) {
	var file *syntax.File
	fired, err := live.within(wall, func() (err error) {
		file, err = live.roundTrip(request)
		return err
	})
	return file, fired, err
}

func (live *worker) roundTrip(request Request) (*syntax.File, error) {
	if err := writeFrame(live.stdin, request); err != nil {
		return nil, err
	}
	var reply Reply
	if err := readFrame(live.stdout, &reply); err != nil {
		return nil, err
	}
	if reply.Seq != request.Seq {
		return nil, errMisattributed
	}
	return &reply.File, nil
}

func (live *worker) stop() (status exitStatus, waited bool) {
	_ = live.stdin.Close()
	live.cancel()
	_ = live.cmd.Wait()
	status.stderr = live.stderr.String()
	if live.cmd.ProcessState == nil {
		status.code = -1
		return status, false
	}
	status.code = live.cmd.ProcessState.ExitCode()
	return status, true
}
