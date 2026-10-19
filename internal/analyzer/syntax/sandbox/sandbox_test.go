package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const helperEnv = "TSPROC_HELPER"

func TestMain(m *testing.M) {
	const helperFields = 3
	fields := strings.SplitN(os.Getenv(helperEnv), "\n", helperFields)
	_ = os.Unsetenv(helperEnv)
	if len(fields) == helperFields {
		mode, pidLog, extra := fields[0], fields[1], fields[2]
		if pidLog != "" {
			logPid(pidLog)
		}
		if mode != "serve" {
			runMode(mode, extra)
		}
	}
	if Child(os.Args, helperHandle, os.Exit) {
		return
	}
	os.Exit(m.Run())
}

func helperSpec(mode, pidLog, extra string) string {
	return strings.Join([]string{mode, pidLog, extra}, "\n")
}

func pidLogOf(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "pids")
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
		"orphan":   func(args []string) { greeting(); spawnSleeper(args[0]) },
		"stderr":   func([]string) { greeting(); failLoudly() },
		"forkhold": func([]string) { greeting(); holdStdout() },
		"sleeper":  func([]string) { <-time.After(4 * time.Second) },
	}
}

func runMode(name, extra string) {
	mode, ok := helperModes()[name]
	if !ok {
		os.Exit(2)
	}
	mode([]string{extra})
	os.Exit(0)
}

func logPid(path string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(2)
	}
	_, writeErr := fmt.Fprintln(file, os.Getpid())
	if writeErr != nil || file.Close() != nil {
		os.Exit(2)
	}
}

func emit(output []byte) {
	_, _ = os.Stdout.Write(output)
}

func failLoudly() {
	_, _ = os.Stderr.Write([]byte("boom \xff\n"))
	os.Exit(3)
}

func sleeperCommand() *exec.Cmd {
	sleeper := exec.CommandContext(context.Background(), os.Args[0])
	sleeper.Env = append(os.Environ(), helperEnv+"=sleeper\n\n")
	return sleeper
}

func holdStdout() {
	var request Request
	if err := readFrame(os.Stdin, &request); err != nil {
		os.Exit(2)
	}
	sleeper := sleeperCommand()
	sleeper.Stdout = os.Stdout
	if err := sleeper.Start(); err != nil {
		os.Exit(2)
	}
}

func spawnSleeper(pidFile string) {
	var request Request
	if err := readFrame(os.Stdin, &request); err != nil {
		os.Exit(2)
	}
	sleeper := sleeperCommand()
	if err := sleeper.Start(); err != nil {
		os.Exit(2)
	}
	sleeperPid := strconv.Itoa(sleeper.Process.Pid)
	if err := os.WriteFile(pidFile, []byte(sleeperPid), 0o600); err != nil {
		os.Exit(2)
	}
	<-time.After(time.Minute)
}

func greeting() {
	if err := writeFrame(os.Stdout, hello{Protocol: protocol}); err != nil {
		os.Exit(2)
	}
}

func recurse(report string) {
	session := NewRunner(1<<30, time.Second).Open()
	_, err := session.Do(context.Background(), Request{Source: syntax.Source{Path: "a"}})
	session.Close()
	if err = os.WriteFile(report, []byte(fmt.Sprintf("err=%v", err)), 0o600); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func helperHandle(request Request) *syntax.File {
	switch request.Source.Path {
	case "hang":
		select {}
	case "hangsignal":
		fmt.Fprintf(os.Stderr, "hanging %d\n", os.Getpid())
		select {}
	case "limit":
		limit := strconv.FormatInt(debug.SetMemoryLimit(-1), 10)
		return &syntax.File{Path: request.Source.Path, ParseError: limit}
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
	return &syntax.File{Path: request.Source.Path, ParseError: string(request.Source.Content)}
}

func wrongSeq() {
	greeting()
	var request Request
	if err := readFrame(os.Stdin, &request); err != nil {
		os.Exit(2)
	}
	_ = writeFrame(
		os.Stdout,
		reply{Seq: request.Seq + 1, File: syntax.File{Path: request.Source.Path}},
	)
	os.Exit(0)
}

func middle() {
	child := exec.CommandContext(context.Background(), os.Args[0], childArg, "1073741824")
	child.Stderr = os.Stdout
	stdin, err := child.StdinPipe()
	if err != nil || child.Start() != nil {
		os.Exit(3)
	}
	if err = writeFrame(
		stdin,
		Request{Seq: 1, Source: syntax.Source{Path: "hangsignal"}},
	); err != nil {
		os.Exit(3)
	}
	reader, _, err := os.Pipe()
	if err != nil {
		os.Exit(3)
	}
	_, _ = reader.Read(make([]byte, 1))
}

func loggedPids(t *testing.T, log string) []int {
	t.Helper()

	raw, err := os.ReadFile(log)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("pid log: %v", err)
	}
	var pids []int
	for field := range strings.FieldsSeq(string(raw)) {
		pid, convErr := strconv.Atoi(field)
		if convErr != nil {
			t.Fatalf("pid %q: %v", field, convErr)
		}
		pids = append(pids, pid)
	}
	return pids
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	return err == nil && proc.Signal(syscall.Signal(0)) == nil
}

func assertStartedAndReaped(t *testing.T, log string, wantStarted int) {
	t.Helper()

	pids := loggedPids(t, log)
	if len(pids) != wantStarted {
		t.Fatalf("started %d parsers %v, want %d", len(pids), pids, wantStarted)
	}
	for _, pid := range pids {
		if processAlive(pid) {
			t.Fatalf("parser %d outlived the session", pid)
		}
	}
}
