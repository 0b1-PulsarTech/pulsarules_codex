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

func TestSpawnFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		runner Runner
		want   string
	}{
		{name: "missing binary", runner: Runner{Exe: "/nonexistent/parser"}, want: "start parser"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			live, err := testCase.runner.spawn(context.Background())
			if live != nil || err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("spawn %v, %v, want %q", live, err, testCase.want)
			}
		})
	}
}
