package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type failAfter struct{ ok int }

func (writer *failAfter) Write(chunk []byte) (int, error) {
	if writer.ok == 0 {
		return 0, io.ErrClosedPipe
	}
	writer.ok--
	return len(chunk), nil
}

type gate struct{ release chan struct{} }

func (hold gate) Read([]byte) (int, error) {
	<-hold.release
	return 0, io.EOF
}

func echo(request Request) *syntax.File { return &syntax.File{Path: request.Source.Path} }

func seedFrames(t *testing.T, paths ...string) []byte {
	t.Helper()

	var input bytes.Buffer
	for _, path := range paths {
		if err := writeFrame(&input, Request{Source: syntax.Source{Path: path}}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return input.Bytes()
}

func readServed(t *testing.T, reader io.Reader, replies int) {
	t.Helper()

	var greeting hello
	if err := readFrame(reader, &greeting); err != nil || greeting.Protocol != protocol {
		t.Fatalf("greeting %+v: %v", greeting, err)
	}
	for range replies {
		var got reply
		if err := readFrame(reader, &got); err != nil {
			t.Fatalf("reply: %v", err)
		}
	}
}

func TestServe(t *testing.T) {
	t.Parallel()

	whole := seedFrames(t, "a.ts", "b.ts")
	testCases := []struct {
		name  string
		input []byte
		out   io.Writer
		want  int
		reply int
	}{
		{name: "clean end", input: whole, want: 0, reply: 2},
		{name: "end before any request", want: 0},
		{name: "truncated", input: whole[:len(whole)-3], want: exitProtocol, reply: 1},
		{name: "garbage", input: []byte("zzzzzzzz"), want: exitProtocol},
		{name: "dead output", input: whole, out: failWriter{}, want: exitProtocol},
		{name: "dead output at the greeting", out: failWriter{}, want: exitProtocol},
		{name: "dead after hello", input: whole, out: &failAfter{ok: 1}, want: exitProtocol},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			release := make(chan struct{})
			closeInput := sync.OnceFunc(func() { close(release) })
			defer closeInput()
			reader, writer := io.Pipe()
			var sink io.Writer = writer
			if testCase.out != nil {
				sink = testCase.out
			}
			done := make(chan int, 1)
			go func() {
				input := io.MultiReader(bytes.NewReader(testCase.input), gate{release})
				done <- serve(input, sink, echo, serveLimits{Ceiling: 1 << 40, Exit: func(int) {}})
			}()
			if testCase.out == nil {
				readServed(t, reader, testCase.reply)
				closeInput()
			}
			if code := <-done; code != testCase.want {
				t.Fatalf("code %d, want %d", code, testCase.want)
			}
		})
	}
}

func TestServeOrphan(t *testing.T) {
	t.Parallel()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	middle := exec.CommandContext(ctx, os.Args[0])
	middle.Env = append(os.Environ(), helperEnv+"=middle\n\n")
	middle.Stdout = writer
	if err = middle.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	_ = writer.Close()
	_ = reader.SetReadDeadline(time.Now().Add(10 * time.Second))
	lines := bufio.NewReader(reader)
	line, err := lines.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "hanging ") {
		_ = middle.Process.Kill()
		t.Fatalf("child never reached the handler: %q %v", line, err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "hanging ")))
	defer func() {
		if proc, findErr := os.FindProcess(pid); findErr == nil {
			_ = proc.Kill()
		}
	}()
	_ = middle.Process.Kill()
	_ = middle.Wait()
	if _, err = io.Copy(io.Discard, lines); err != nil {
		t.Fatalf("the orphaned child outlived its parent: %v", err)
	}
}
