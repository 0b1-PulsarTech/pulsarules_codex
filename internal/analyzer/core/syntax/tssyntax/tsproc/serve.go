package tsproc

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

type feed struct {
	requests chan Request
	ended    chan int
	pending  atomic.Int32
}

type Limits struct {
	Ceiling uint64
	Exit    func(int)
}

func Serve(in io.Reader, out io.Writer, handle Handler, limits Limits) int {
	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	workers.Go(func() { watch(ctx, limits) })
	defer func() {
		cancel()
		workers.Wait()
	}()
	if err := writeFrame(out, hello{Protocol: protocol}); err != nil {
		return ExitProtocol
	}
	input := &feed{requests: make(chan Request), ended: make(chan int, 1)}
	go input.read(ctx, in, limits.Exit)
	for {
		select {
		case request := <-input.requests:
			file := handle(request)
			input.pending.Add(-1)
			if err := writeFrame(out, Reply{Seq: request.Seq, File: *file}); err != nil {
				return ExitProtocol
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
		code = ExitProtocol
		exit(code)
	case !errors.Is(err, io.EOF):
		code = ExitProtocol
	}
	input.ended <- code
}
