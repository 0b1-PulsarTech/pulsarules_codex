package sandbox

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type feed struct {
	requests chan Request
	ended    chan int
	pending  atomic.Int32
}

type serveLimits struct {
	Ceiling uint64
	Exit    func(int)
}

func serve(in io.Reader, out io.Writer, handle func(Request) *syntax.File, limits serveLimits) int {
	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	workers.Go(func() { watch(ctx, limits) })
	defer func() {
		cancel()
		workers.Wait()
	}()
	if err := writeFrame(out, hello{Protocol: protocol}); err != nil {
		return exitProtocol
	}
	input := &feed{requests: make(chan Request), ended: make(chan int, 1)}
	go input.read(ctx, in, limits.Exit)
	for {
		select {
		case request := <-input.requests:
			file := handle(request)
			input.pending.Add(-1)
			if err := writeFrame(out, reply{Seq: request.Seq, File: *file}); err != nil {
				return exitProtocol
			}
		case code := <-input.ended:
			return code
		}
	}
}

func (input *feed) read(ctx context.Context, in io.Reader, exit func(int)) {
	reader := bufio.NewReader(in)
	for {
		var request Request
		if err := readFrame(reader, &request); err != nil {
			input.finish(err, exit)
			return
		}
		input.pending.Add(1)
		select {
		case input.requests <- request:
		case <-ctx.Done():
			return
		}
	}
}

func (input *feed) finish(err error, exit func(int)) {
	busy := input.pending.Load() > 0
	code := 0
	switch {
	case busy:
		code = exitProtocol
		exit(code)
	case !errors.Is(err, io.EOF):
		code = exitProtocol
	}
	input.ended <- code
}
