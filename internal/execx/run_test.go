package execx

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRun_Success proves stdout and stderr are captured separately and
// trimmed the way each field promises: Stdout raw, Stderr trimmed.
func TestRun_Success(t *testing.T) {
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

// TestRun_NonZeroExit proves a non-zero exit returns a *Error carrying
// stderr and wrapping the underlying *exec.ExitError, so a caller can still
// branch on the exit code via errors.As.
func TestRun_NonZeroExit(t *testing.T) {
	t.Parallel()

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
	if exitErr.ExitCode() != 3 {
		t.Errorf("ExitCode() = %d, want 3", exitErr.ExitCode())
	}
}

// TestRun_MissingBinary proves a binary that cannot even start (never on
// PATH) still returns a *Error, not a bare *exec.Error the caller has to
// know to unwrap itself.
func TestRun_MissingBinary(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Command{Name: "definitely-not-a-real-binary-xyz"})
	if err == nil {
		t.Fatal("Run: expected an error for a missing binary")
	}
	var execErr *Error
	if !errors.As(err, &execErr) {
		t.Fatalf("errors.As(%v, *Error) = false", err)
	}
}

// TestRun_Timeout proves Command.Timeout kills a command that overruns it,
// well before the process would finish on its own; the real sleep happens
// in the child process, not in this test goroutine.
func TestRun_Timeout(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, err := Run(context.Background(), Command{
		Name:    "sh",
		Args:    []string{"-c", "sleep 5"},
		Timeout: 50 * time.Millisecond,
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run: expected a timeout error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Run took %v, want it killed near the 50ms timeout", elapsed)
	}
	var execErr *Error
	if !errors.As(err, &execErr) {
		t.Fatalf("errors.As(%v, *Error) = false", err)
	}
}

// TestRun_ZeroTimeoutUsesCallerContext proves Timeout of zero leaves the
// caller's own context deadline in control instead of adding a second one.
func TestRun_ZeroTimeoutUsesCallerContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Run(ctx, Command{Name: "sh", Args: []string{"-c", "sleep 5"}})
	if err == nil {
		t.Fatal("Run: expected the caller's context deadline to kill the command")
	}
}

// TestRun_Env proves Env appends to, rather than replaces, the inherited
// environment, and that a nil Env leaves it untouched.
func TestRun_Env(t *testing.T) {
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

// TestRun_Dir proves Dir sets the child's working directory.
func TestRun_Dir(t *testing.T) {
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

// TestError_ErrorMessage pins the message shape callers may end up logging.
func TestError_ErrorMessage(t *testing.T) {
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
