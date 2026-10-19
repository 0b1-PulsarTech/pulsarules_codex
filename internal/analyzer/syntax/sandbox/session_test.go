package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func request(path string) Request {
	return Request{Source: syntax.Source{Path: path, Content: []byte("x")}}
}

func checkOutcome(t *testing.T, path string, outcome Outcome, wantSkip string) {
	t.Helper()

	if wantSkip == "" {
		if outcome.File == nil || outcome.File.Path != path {
			t.Fatalf("%s: outcome %+v", path, outcome)
		}
		return
	}
	if outcome.File != nil || !strings.Contains(outcome.Skip, wantSkip) {
		t.Fatalf("%s: outcome %+v, want skip %q", path, outcome, wantSkip)
	}
}

type outcomeCase struct {
	name        string
	wall        time.Duration
	memory      uint64
	paths       []string
	wantSkips   []string
	wantStarted int
}

func outcomeCases() []outcomeCase {
	const generous = 1 << 30
	return []outcomeCase{
		{
			name: "echo", wall: time.Minute, memory: generous,
			paths: []string{"a", "b"}, wantSkips: []string{"", ""}, wantStarted: 1,
		},
		{
			name: "hang then recover", wall: 300 * time.Millisecond, memory: generous,
			paths: []string{"a", "hang", "b"}, wantSkips: []string{"", "time limit", ""},
			wantStarted: 2,
		},
		{
			name: "crash then recover", wall: time.Minute, memory: generous,
			paths: []string{"crash", "b"}, wantSkips: []string{"exit 1", ""}, wantStarted: 2,
		},
		{
			name: "memory then recover", wall: time.Minute, memory: 64 << 20,
			paths: []string{"a", "alloc", "b"}, wantSkips: []string{"", "64 MiB memory", ""},
			wantStarted: 2,
		},
	}
}

func TestSessionOutcomes(t *testing.T) {
	testCases := outcomeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			log := pidLogOf(t)
			t.Setenv(helperEnv, helperSpec("serve", log, ""))
			runner := NewRunner(testCase.memory, testCase.wall)
			session := runner.Open()
			for idx, path := range testCase.paths {
				outcome, err := session.Do(ctx, request(path))
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				checkOutcome(t, path, outcome, testCase.wantSkips[idx])
			}
			session.Close()
			assertStartedAndReaped(t, log, testCase.wantStarted)
		})
	}
}

func TestSessionBrokenChildren(t *testing.T) {
	testCases := []struct {
		name string
		mode string
		wall time.Duration
		want string
	}{
		{name: "garbage reply", mode: "garbage", want: "parser process failed"},
		{name: "partial reply", mode: "partial", want: "parser process failed"},
		{name: "reply for another request", mode: "wrongseq", want: "another file"},
		{name: "stderr tail", mode: "stderr", want: `exit 3, stderr "boom \xff\n"`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			log := pidLogOf(t)
			t.Setenv(helperEnv, helperSpec(testCase.mode, log, ""))
			runner := NewRunner(1<<30, time.Minute)
			session := runner.Open()
			outcome, err := session.Do(context.Background(), request("a"))
			session.Close()
			if err != nil {
				t.Fatalf("error %v", err)
			}
			checkOutcome(t, "a", outcome, testCase.want)
			assertStartedAndReaped(t, log, 1)
		})
	}
}

func TestSessionLaunchFailureIsSticky(t *testing.T) {
	testCases := []struct {
		name    string
		mode    string
		wall    time.Duration
		started int
	}{
		{name: "no greeting", mode: "silent", wall: 300 * time.Millisecond, started: 1},
		{name: "other protocol", mode: "badproto", started: 1},
		{name: "not a parser", mode: "plain", started: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			log := pidLogOf(t)
			t.Setenv(helperEnv, helperSpec(testCase.mode, log, ""))
			runner := NewRunner(1<<30, time.Minute)
			if testCase.wall != 0 {
				runner.wall = testCase.wall
			}
			session := runner.Open()
			for call := range 3 {
				outcome, err := session.Do(context.Background(), request("a"))
				if !errors.Is(err, errUnavailable) || outcome.File != nil || outcome.Skip != "" {
					t.Fatalf("call %d: outcome %+v, error %v", call, outcome, err)
				}
			}
			session.Close()
			assertStartedAndReaped(t, log, testCase.started)
		})
	}
}

func TestSessionNested(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report")
	log := pidLogOf(t)
	t.Setenv(helperEnv, helperSpec("recurse", log, report))
	runner := NewRunner(1<<30, 5*time.Second)
	session := runner.Open()
	_, err := session.Do(context.Background(), request("a"))
	session.Close()
	if !errors.Is(err, errUnavailable) {
		t.Fatalf("error %v", err)
	}
	assertStartedAndReaped(t, log, 1)
	inner, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(string(inner), errNested.Error()) {
		t.Fatalf("the nested session spawned: %s", inner)
	}
}

func TestSessionHeldStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	t.Setenv(helperEnv, helperSpec("forkhold", pidLogOf(t), ""))
	runner := NewRunner(1<<30, 300*time.Millisecond)
	session := runner.Open()
	outcome, err := session.Do(ctx, request("a"))
	session.Close()
	if err != nil {
		t.Fatalf("a descendant holding stdout stretched the bound: %v", err)
	}
	checkOutcome(t, "a", outcome, "time limit")
}

func TestSessionChildCeiling(t *testing.T) {
	const memory = 64 << 20
	t.Setenv(helperEnv, helperSpec("serve", pidLogOf(t), ""))
	runner := NewRunner(memory, time.Minute)
	session := runner.Open()
	outcome, err := session.Do(context.Background(), request("limit"))
	session.Close()
	if err != nil || outcome.File == nil {
		t.Fatalf("outcome %+v, %v", outcome, err)
	}
	if want := strconv.Itoa(memory / 2); outcome.File.ParseError != want {
		t.Fatalf("soft limit %q, want %q", outcome.File.ParseError, want)
	}
}

func TestSessionContext(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     func() (context.Context, context.CancelFunc)
		path    string
		wantErr error
	}{
		{
			name: "cancelled first", path: "a", wantErr: context.Canceled,
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			name: "deadline mid-flight", path: "hang", wantErr: context.DeadlineExceeded,
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 300*time.Millisecond)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := testCase.ctx()
			defer cancel()
			log := pidLogOf(t)
			t.Setenv(helperEnv, helperSpec("serve", log, ""))
			runner := NewRunner(1<<30, time.Minute)
			session := runner.Open()
			_, err := session.Do(ctx, request(testCase.path))
			session.Close()
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
			if got := len(loggedPids(t, log)); got > 1 {
				t.Fatalf("started %d parsers for one call", got)
			}
			for _, pid := range loggedPids(t, log) {
				if processAlive(pid) {
					t.Fatalf("parser %d outlived the session", pid)
				}
			}
		})
	}
}

func TestSessionAcceptLate(t *testing.T) {
	log := pidLogOf(t)
	t.Setenv(helperEnv, helperSpec("serve", log, ""))
	runner := NewRunner(1<<30, time.Minute)
	session := runner.Open()
	defer session.Close()
	outcome, err := session.Do(context.Background(), request("a"))
	if err != nil || outcome.File == nil || session.live == nil {
		t.Fatalf("first: %+v, %v", outcome, err)
	}
	if late := session.acceptLate(outcome.File); late.File != outcome.File || session.live != nil {
		t.Fatalf("late accept kept the worker: %+v", late)
	}
	assertRelaunched(t, session, log)
}

func assertRelaunched(t *testing.T, session *Session, log string) {
	t.Helper()

	outcome, err := session.Do(context.Background(), request("b"))
	if err != nil || outcome.File == nil {
		t.Fatalf("second: %+v, %v", outcome, err)
	}
	pids := loggedPids(t, log)
	if len(pids) != 2 || processAlive(pids[0]) || !processAlive(pids[1]) {
		t.Fatalf("parsers %v: want the first reaped and the second alive", pids)
	}
}

func isGone(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return !processAlive(pid)
	}
	// A killed orphan stays a zombie until init reaps it, which a container may not do.
	return strings.Contains(string(stat), ") Z ")
}

func TestSessionKillsDescendants(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups are unix-only")
	}
	sleeperFile := filepath.Join(t.TempDir(), "sleeper")
	t.Setenv(helperEnv, helperSpec("orphan", pidLogOf(t), sleeperFile))
	runner := NewRunner(1<<30, 300*time.Millisecond)
	session := runner.Open()
	outcome, err := session.Do(context.Background(), request("a"))
	session.Close()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	checkOutcome(t, "a", outcome, "time limit")
	raw, err := os.ReadFile(sleeperFile)
	if err != nil {
		t.Fatalf("sleeper pid: %v", err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Fatalf("sleeper pid %q: %v", raw, err)
	}
	if !goneWithin(pid, 2*time.Second) {
		t.Fatalf("descendant %d survived the session", pid)
	}
}

func goneWithin(pid int, limit time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for !isGone(pid) {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
	return true
}
