package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func helperRunner(t *testing.T, wall time.Duration, memory uint64) Runner {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	return Runner{Exe: exe, Args: []string{ChildArg}, Memory: memory, Wall: wall}
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
	t.Parallel()

	testCases := outcomeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			session := helperRunner(t, testCase.wall, testCase.memory).Open()
			for idx, path := range testCase.paths {
				outcome, err := session.Do(ctx, Request{Path: path, Text: "x"})
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				checkOutcome(t, path, outcome, testCase.wantSkips[idx])
			}
			session.Close()
			if session.started != testCase.wantStarted || session.reaped != session.started {
				t.Fatalf("started %d reaped %d", session.started, session.reaped)
			}
		})
	}
}

func TestSessionBrokenChildren(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		exe  string
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
			t.Parallel()

			runner := helperRunner(t, time.Minute, 1<<30)
			runner.Args = []string{testCase.mode}
			session := runner.Open()
			outcome, err := session.Do(context.Background(), Request{Path: "a"})
			session.Close()
			if err != nil {
				t.Fatalf("error %v", err)
			}
			checkOutcome(t, "a", outcome, testCase.want)
			if session.reaped != session.started {
				t.Fatalf("started %d reaped %d", session.started, session.reaped)
			}
		})
	}
}

func TestSessionLaunchFailureIsSticky(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		exe     string
		mode    string
		wall    time.Duration
		started int
	}{
		{name: "no greeting", mode: "silent", wall: 300 * time.Millisecond, started: 1},
		{name: "other protocol", mode: "badproto", started: 1},
		{name: "not a parser", mode: "plain", started: 1},
		{name: "missing binary", exe: "/nonexistent/parser", mode: "x"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runner := helperRunner(t, time.Minute, 1<<30)
			if testCase.wall != 0 {
				runner.Wall = testCase.wall
			}
			runner.Args = []string{testCase.mode}
			if testCase.exe != "" {
				runner.Exe = testCase.exe
			}
			session := runner.Open()
			for call := range 3 {
				outcome, err := session.Do(context.Background(), Request{Path: "a"})
				if !errors.Is(err, ErrUnavailable) || outcome.File != nil || outcome.Skip != "" {
					t.Fatalf("call %d: outcome %+v, error %v", call, outcome, err)
				}
			}
			session.Close()
			if session.started != testCase.started || session.reaped != session.started {
				t.Fatalf("started %d reaped %d", session.started, session.reaped)
			}
		})
	}
}

func TestSessionNested(t *testing.T) {
	t.Parallel()

	report := filepath.Join(t.TempDir(), "report")
	runner := helperRunner(t, 5*time.Second, 1<<30)
	runner.Args = []string{"recurse", report}
	session := runner.Open()
	_, err := session.Do(context.Background(), Request{Path: "a"})
	session.Close()
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error %v", err)
	}
	if session.started != 1 || session.reaped != 1 {
		t.Fatalf("started %d reaped %d", session.started, session.reaped)
	}
	inner, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.HasPrefix(string(inner), "started=0 err=") ||
		!strings.Contains(string(inner), "unavailable") {
		t.Fatalf("the nested session spawned: %s", inner)
	}
}

func TestSessionHeldStdout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	runner := helperRunner(t, 300*time.Millisecond, 1<<30)
	runner.Args = []string{"forkhold"}
	session := runner.Open()
	outcome, err := session.Do(ctx, Request{Path: "a"})
	session.Close()
	if err != nil {
		t.Fatalf("a descendant holding stdout stretched the bound: %v", err)
	}
	checkOutcome(t, "a", outcome, "time limit")
}

func TestSessionChildCeiling(t *testing.T) {
	t.Parallel()

	const memory = 64 << 20
	session := helperRunner(t, time.Minute, memory).Open()
	outcome, err := session.Do(context.Background(), Request{Path: "limit"})
	session.Close()
	if err != nil || outcome.File == nil {
		t.Fatalf("outcome %+v, %v", outcome, err)
	}
	if want := strconv.Itoa(memory / 2); outcome.File.ParseError != want {
		t.Fatalf("soft limit %q, want %q", outcome.File.ParseError, want)
	}
}

func TestSessionContext(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			ctx, cancel := testCase.ctx()
			defer cancel()
			session := helperRunner(t, time.Minute, 1<<30).Open()
			_, err := session.Do(ctx, Request{Path: testCase.path})
			session.Close()
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
			if session.reaped != session.started {
				t.Fatalf("started %d reaped %d", session.started, session.reaped)
			}
		})
	}
}

func TestSessionAcceptLate(t *testing.T) {
	t.Parallel()

	session := helperRunner(t, time.Minute, 1<<30).Open()
	defer session.Close()
	ctx := context.Background()
	outcome, err := session.Do(ctx, Request{Path: "a"})
	if err != nil || outcome.File == nil || session.live == nil {
		t.Fatalf("first: %+v, %v", outcome, err)
	}
	if late := session.acceptLate(outcome.File); late.File != outcome.File || session.live != nil {
		t.Fatalf("late accept kept the worker: %+v", late)
	}
	if outcome, err = session.Do(ctx, Request{Path: "b"}); err != nil || outcome.File == nil {
		t.Fatalf("second: %+v, %v", outcome, err)
	}
	if session.started != 2 || session.reaped != 1 {
		t.Fatalf("started %d reaped %d", session.started, session.reaped)
	}
}
