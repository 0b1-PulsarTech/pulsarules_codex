package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOpenPipesClosesStdinOnFailure(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(), "unused")
	cmd.Stdout = &bytes.Buffer{}
	stdin, stdout, err := openPipes(cmd)
	if err == nil || stdin != nil || stdout != nil {
		t.Fatalf("pipes %v %v, %v", stdin, stdout, err)
	}
	reader, ok := cmd.Stdin.(*os.File)
	if !ok {
		t.Fatalf("stdin is %T", cmd.Stdin)
	}
	defer reader.Close()
	_ = reader.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err = reader.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("the write end of stdin is still open: %v", err)
	}
	runtime.KeepAlive(cmd)
}

func TestOpenPipes(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(), "unused")
	stdin, stdout, err := openPipes(cmd)
	if err != nil || stdin == nil || stdout == nil {
		t.Fatalf("pipes %v %v, %v", stdin, stdout, err)
	}
	_ = stdin.Close()
	_ = stdout.Close()
}

func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestSpawnFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  func() context.Context
		want string
	}{
		{name: "cancelled before start", ctx: cancelled, want: "start parser"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			live, err := NewRunner(1<<30, time.Second).spawn(testCase.ctx())
			if live != nil || err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("spawn %v, %v, want %q", live, err, testCase.want)
			}
		})
	}
}

func TestNewRunner(t *testing.T) {
	t.Parallel()

	runner := NewRunner(64<<20, 3*time.Second)
	if runner.memory != 64<<20 || runner.wall != 3*time.Second {
		t.Fatalf("runner %+v", runner)
	}
}
