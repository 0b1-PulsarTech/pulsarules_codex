package sandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

var errUnavailable = errors.New("parser process unavailable")

type Session struct {
	runner  Runner
	live    *worker
	failure error
	sent    uint64
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
			session.failure = fmt.Errorf("%w: %w", errUnavailable, err)
			return Outcome{}, session.failure
		}
	}
	session.sent++
	request.Seq = session.sent
	file, fired, err := session.live.exchange(request, session.runner.wall)
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
	if err = live.greet(session.runner.wall); err != nil {
		return fmt.Errorf("%s: %w", session.retire(), err)
	}
	return nil
}

func (session *Session) acceptLate(file *syntax.File) Outcome {
	session.retire()
	return Outcome{File: file}
}

func (session *Session) failed(ctx context.Context, cause error) (Outcome, error) {
	const mib = 1 << 20
	expired := session.live.expired.Load()
	status := session.retire()
	if err := ctx.Err(); err != nil {
		return Outcome{}, fmt.Errorf("run parser: %w", err)
	}
	switch {
	case expired:
		return Outcome{Skip: fmt.Sprintf(
			"skipped: parse exceeded the %s time limit", session.runner.wall)}, nil
	case status.code == exitMemory:
		return Outcome{Skip: fmt.Sprintf(
			"skipped: parse exceeded the %d MiB memory limit", session.runner.memory/mib)}, nil
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
	return live.stop()
}
