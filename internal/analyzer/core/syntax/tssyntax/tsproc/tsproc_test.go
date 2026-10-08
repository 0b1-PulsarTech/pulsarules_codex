package tsproc

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func TestMain(m *testing.M) {
	if Child(os.Args, helperHandle, os.Exit) {
		return
	}
	if len(os.Args) > 1 {
		runMode(os.Args[1:])
	}
	os.Exit(m.Run())
}

func helperModes() map[string]func([]string) {
	return map[string]func([]string){
		"garbage":  func([]string) { greeting(); emit([]byte("not a frame at all")) },
		"partial":  func([]string) { greeting(); emit([]byte{0, 0, 0, 100, '{'}) },
		"plain":    func([]string) { emit([]byte("plain output\n")) },
		"badproto": func([]string) { _ = writeFrame(os.Stdout, hello{Protocol: protocol + 1}) },
		"silent":   func([]string) { _, _ = io.Copy(io.Discard, os.Stdin) },
		"wrongseq": func([]string) { wrongSeq() },
		"middle":   func([]string) { middle() },
		"recurse":  func(args []string) { recurse(args[0]) },
		"stderr":   func([]string) { greeting(); failLoudly() },
		"forkhold": func([]string) { greeting(); holdStdout() },
		"sleeper":  func([]string) { <-time.After(4 * time.Second) },
	}
}

func runMode(args []string) {
	if mode, ok := helperModes()[args[0]]; ok {
		mode(args[1:])
		os.Exit(0)
	}
}

func emit(output []byte) {
	_, _ = os.Stdout.Write(output)
}

func failLoudly() {
	_, _ = os.Stderr.Write([]byte("boom \xff\n"))
	os.Exit(3)
}

func holdStdout() {
	var request Request
	if err := readFrame(os.Stdin, &request); err != nil {
		os.Exit(2)
	}
	sleeper := exec.CommandContext(context.Background(), os.Args[0], "sleeper")
	sleeper.Stdout = os.Stdout
	if err := sleeper.Start(); err != nil {
		os.Exit(2)
	}
}

func greeting() {
	if err := writeFrame(os.Stdout, hello{Protocol: protocol}); err != nil {
		os.Exit(2)
	}
}

func recurse(report string) {
	exe, err := os.Executable()
	if err != nil {
		os.Exit(2)
	}
	args := []string{"recurse", report}
	runner := Runner{Exe: exe, Args: args, Memory: 1 << 30, Wall: time.Second}
	session := runner.Open()
	_, err = session.Do(context.Background(), Request{Path: "a"})
	session.Close()
	line := fmt.Sprintf("started=%d err=%v", session.started, err)
	if err = os.WriteFile(report, []byte(line), 0o600); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func helperHandle(request Request) *syntax.File {
	switch request.Path {
	case "hang":
		select {}
	case "hangsignal":
		fmt.Fprintf(os.Stderr, "hanging %d\n", os.Getpid())
		select {}
	case "limit":
		limit := strconv.FormatInt(debug.SetMemoryLimit(-1), 10)
		return &syntax.File{Path: request.Path, ParseError: limit}
	case "crash":
		os.Exit(1)
	case "alloc":
		blob := make([]byte, 256<<20)
		for idx := 0; idx < len(blob); idx += 4096 {
			blob[idx]++
		}
		if blob[0] == 7 {
			return &syntax.File{}
		}
		select {}
	}
	return &syntax.File{Path: request.Path, ParseError: request.Text}
}

func wrongSeq() {
	greeting()
	var request Request
	if err := readFrame(os.Stdin, &request); err != nil {
		os.Exit(2)
	}
	_ = writeFrame(os.Stdout, Reply{Seq: request.Seq + 1, File: syntax.File{Path: request.Path}})
	os.Exit(0)
}

func middle() {
	child := exec.CommandContext(context.Background(), os.Args[0], ChildArg, "1073741824")
	child.Stderr = os.Stdout
	stdin, err := child.StdinPipe()
	if err != nil || child.Start() != nil {
		os.Exit(3)
	}
	if err = writeFrame(stdin, Request{Seq: 1, Path: "hangsignal"}); err != nil {
		os.Exit(3)
	}
	reader, _, err := os.Pipe()
	if err != nil {
		os.Exit(3)
	}
	_, _ = reader.Read(make([]byte, 1))
}
