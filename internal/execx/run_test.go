package execx

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const shortTimeout = 50 * time.Millisecond

func TestRun_CapturesStdoutRawStderrTrimmed(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Command{
		Name: "sh",
		Args: []string{"-c", "printf 'out\\n'; printf '  err  \\n' >&2"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Stdout != "out\n" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "out\n")
	}
	if result.Stderr != "err" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "err")
	}
}

func TestRun_NonZeroExitWrapsExitErrorForErrorsAs(t *testing.T) {
	t.Parallel()

	const wantExitCode = 3

	_, err := Run(context.Background(), Command{
		Name: "sh",
		Args: []string{"-c", "echo boom >&2; exit 3"},
	})
	if err == nil {
		t.Fatal("Run: expected an error for a non-zero exit")
	}

	var execErr *Error
	if !errors.As(err, &execErr) {
		t.Fatalf("errors.As(%v, *Error) = false", err)
	}
	if execErr.Stderr != "boom" {
		t.Errorf("Stderr = %q, want %q", execErr.Stderr, "boom")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf(
			"errors.As(%v, *exec.ExitError) = false, want the exit error reachable through Unwrap",
			err,
		)
	}
	if exitErr.ExitCode() != wantExitCode {
		t.Errorf("ExitCode() = %d, want %d", exitErr.ExitCode(), wantExitCode)
	}
}

func TestRun_MissingBinary(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Command{Name: "definitely-not-a-real-binary-xyz"})
	if err == nil {
		t.Fatal("Run: expected an error for a missing binary")
	}
	if _, ok := errors.AsType[*Error](err); !ok {
		t.Fatalf("errors.AsType[*Error](%v) = false", err)
	}
}

func TestRun_TimeoutKillsGrandchild(t *testing.T) {
	t.Parallel()

	const overrunBudget = 2 * time.Second

	start := time.Now()
	_, err := Run(context.Background(), Command{
		Name:    "sh",
		Args:    []string{"-c", "sleep 5 & wait"},
		Timeout: shortTimeout,
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run: expected a timeout error")
	}
	if elapsed > overrunBudget {
		t.Fatalf("Run took %v, want it killed near the %v timeout", elapsed, shortTimeout)
	}
	if _, ok := errors.AsType[*Error](err); !ok {
		t.Fatalf("errors.AsType[*Error](%v) = false", err)
	}
}

func TestRun_ZeroTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	_, err := Run(ctx, Command{Name: "sh", Args: []string{"-c", "sleep 5"}})
	if err == nil {
		t.Fatal("Run: expected the caller's context deadline to kill the command")
	}
}

func TestRun_EnvAppends(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		env  []string
		want string
	}{
		{name: "extra var appended", env: []string{"EXECX_TEST_VAR=hello"}, want: "hello"},
		{name: "nil env leaves the var unset", env: nil, want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := Run(context.Background(), Command{
				Name: "sh",
				Args: []string{"-c", "printf '%s' \"$EXECX_TEST_VAR\""},
				Env:  testCase.env,
			})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if result.Stdout != testCase.want {
				t.Errorf("Stdout = %q, want %q", result.Stdout, testCase.want)
			}
		})
	}
}

func TestRun_DirSetsChildWorkingDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result, err := Run(context.Background(), Command{Name: "pwd", Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != dir {
		t.Errorf("pwd = %q, want %q", got, dir)
	}
}

func TestError_MessageShapeMatchesWhatCallersLog(t *testing.T) {
	t.Parallel()

	wrapped := errors.New("exit status 1")
	execErr := &Error{Name: "git", Args: []string{"log", "-1"}, Stderr: "no commits", Err: wrapped}

	got := execErr.Error()
	for _, want := range []string{"git", "log -1", "exit status 1", "no commits"} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, missing %q", got, want)
		}
	}
	if !errors.Is(execErr, wrapped) {
		t.Errorf("errors.Is(execErr, wrapped) = false, want true (Unwrap must reach it)")
	}
}

func TestRun_MaxOutputCapsEachStream(t *testing.T) {
	t.Parallel()

	const limit = 100

	testCases := []struct {
		name         string
		script       string
		wantExceeded bool
		wantStdout   int
	}{
		{"stdout at the limit", "head -c 100 /dev/zero", false, limit},
		{"stdout past the limit", "head -c 101 /dev/zero", true, limit},
		{"stderr past the limit", "head -c 101 /dev/zero >&2", true, 0},
		{"endless stdout is killed", "while true; do echo xxxxxxxxxx; done", true, limit},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := Run(context.Background(), Command{
				Name:      "sh",
				Args:      []string{"-c", testCase.script},
				MaxOutput: limit,
				Timeout:   time.Minute,
			})
			if got := errors.Is(err, ErrOutputExceeded); got != testCase.wantExceeded {
				t.Fatalf("errors.Is(ErrOutputExceeded) = %v, want %v (err %v)",
					got, testCase.wantExceeded, err)
			}
			if !testCase.wantExceeded && err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(result.Stdout) != testCase.wantStdout {
				t.Errorf("len(Stdout) = %d, want %d", len(result.Stdout), testCase.wantStdout)
			}
		})
	}
}

func TestRun_TimeoutReportsDeadlineExceeded(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Command{
		Name:    "sh",
		Args:    []string{"-c", "sleep 5"},
		Timeout: shortTimeout,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("errors.Is(DeadlineExceeded) = false for %v", err)
	}
}

func TestRun_ContextDoneBeforeStartReportsOnce(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		ctx    func() (context.Context, context.CancelFunc)
		ctxErr error
	}{
		{
			"deadline passed",
			func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			context.DeadlineExceeded,
		},
		{
			"cancelled before start",
			func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			context.Canceled,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := testCase.ctx()
			defer cancel()

			_, err := Run(ctx, Command{Name: "sh", Args: []string{"-c", "true"}})
			if !errors.Is(err, testCase.ctxErr) {
				t.Fatalf("errors.Is(%v) = false for %v", testCase.ctxErr, err)
			}
			if got := strings.Count(err.Error(), testCase.ctxErr.Error()); got != 1 {
				t.Errorf("%v reported %d times in %q", testCase.ctxErr, got, err)
			}
		})
	}
}

func TestCappedBuffer_ReportsExceedOnce(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		limit     int
		chunks    []string
		wantCalls int
		wantKept  string
	}{
		{"under the limit", 4, []string{"ab", "cd"}, 0, "abcd"},
		{"one byte over", 4, []string{"abcde"}, 1, "abcd"},
		{"exceeded across writes", 4, []string{"abc", "de"}, 1, "abcd"},
		{"later writes stay silent", 4, []string{"abcde", "fg", "hi"}, 1, "abcd"},
		{"unbounded when limit is zero", 0, []string{"abcdefgh"}, 0, "abcdefgh"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			capped := &cappedBuffer{limit: testCase.limit, onExceed: func() { calls++ }}
			for _, chunk := range testCase.chunks {
				written, err := capped.Write([]byte(chunk))
				if written != len(chunk) || err != nil {
					t.Fatalf("Write(%q) = %d, %v; want %d, nil", chunk, written, err, len(chunk))
				}
			}
			if calls != testCase.wantCalls {
				t.Errorf("onExceed calls = %d, want %d", calls, testCase.wantCalls)
			}
			if got := capped.buf.String(); got != testCase.wantKept {
				t.Errorf("kept = %q, want %q", got, testCase.wantKept)
			}
		})
	}
}
