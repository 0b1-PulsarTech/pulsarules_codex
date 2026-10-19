package sandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

var ErrUnavailable = errors.New("parser process unavailable")

type Session struct {
	runner  Runner
	live    *worker
	failure error
	started int
	sent    uint64
	reaped  int
}

func (runner Runner) Open() *Session { return &Session{runner: runner} }

func (session *Session) Close() { session.retire() }

func (session *Session) Do(ctx context.Context, request Request) (Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Outcome{}, fmt.Errorf("run parser: %w", err)
	}
	if session.failure != nil {
		return Outcome{}, session.failure
	}
	if session.live == nil {
		if err := session.launch(ctx); err != nil {
			session.failure = fmt.Errorf("%w: %w", ErrUnavailable, err)
			return Outcome{}, session.failure
		}
	}
	session.sent++
	request.Seq = session.sent
	file, fired, err := session.live.exchange(request, session.runner.Wall)
	if err == nil && fired {
		return session.acceptLate(file), nil
	}
	if err == nil {
		return Outcome{File: file}, nil
	}
	return session.failed(ctx, err)
}

func (session *Session) launch(ctx context.Context) error {
	live, err := session.runner.spawn(ctx)
	if err != nil {
		return err
	}
	session.live = live
	session.started++
	if err = live.greet(session.runner.Wall); err != nil {
		return fmt.Errorf("%s: %w", session.retire(), err)
	}
	return nil
}

func (session *Session) acceptLate(file *syntax.File) Outcome {
	session.retire()
	return Outcome{File: file}
}

func (session *Session) failed(ctx context.Context, cause error) (Outcome, error) {
	expired := session.live.expired.Load()
	status := session.retire()
	if err := ctx.Err(); err != nil {
		return Outcome{}, fmt.Errorf("run parser: %w", err)
	}
	switch {
	case expired:
		return Outcome{Skip: fmt.Sprintf(
			"skipped: parse exceeded the %s time limit", session.runner.Wall)}, nil
	case status.code == exitMemory:
		return Outcome{Skip: fmt.Sprintf(
			"skipped: parse exceeded the %d MiB memory limit", session.runner.Memory/MiB)}, nil
	}
	return Outcome{Skip: fmt.Sprintf(
		"skipped: parser process failed (%s): %v", status, cause)}, nil
}

func (session *Session) retire() exitStatus {
	live := session.live
	if live == nil {
		return exitStatus{}
	}
	session.live = nil
	status, waited := live.stop()
	if waited {
		session.reaped++
	}
	return status
}
