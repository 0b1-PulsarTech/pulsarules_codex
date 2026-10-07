package dartsyntax

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func writeFakeDart(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fake dart needs a POSIX shell")
	}
	const fakeDart = `#!/bin/sh
fail() { echo "boom" >&2; exit 1; }
version() { echo "Dart SDK version: ${FAKE_VERSION:-9.9.9} (stable)"; }
[ -n "$FAKE_ARGV" ] && echo "$*" >> "$FAKE_ARGV"
[ "$1" = "--suppress-analytics" ] && shift
case "$1" in
--version)
  [ "$FAKE_FAIL" = "version" ] && fail
  [ "$FAKE_FAIL" = "noversion" ] && { echo garbage; exit 0; }
  [ "$FAKE_FAIL" = "oldsdk" ] && { echo "Dart SDK version: 3.9.0 (stable)"; exit 0; }
  [ "$FAKE_FAIL" = "stderrversion" ] && { version >&2; exit 0; }
  version ;;
pub)
  [ -n "$FAKE_LOG" ] && echo "$*" >> "$FAKE_LOG"
  [ "$FAKE_FAIL" = "hang" ] && { sleep 30 & sleep 30; }
  [ "$FAKE_FAIL" = "pub" ] && fail
  [ "$FAKE_FAIL" = "offline" ] && [ "$3" = "--offline" ] && fail
  exit 0 ;;
compile)
  [ -n "$FAKE_COMPILES" ] && echo "${FAKE_VERSION:-9.9.9}" >> "$FAKE_COMPILES"
  [ "$FAKE_FAIL" = "compile" ] && fail
  echo exe > "$5"
  if [ -n "$FAKE_EXE" ]; then cp "$FAKE_EXE" "$5"; chmod +x "$5"; fi ;;
esac
`
	path := filepath.Join(t.TempDir(), "dart")
	if err := os.WriteFile(path, []byte(fakeDart), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeLog(t *testing.T) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("FAKE_LOG", log)
	return log
}

func fakeArgv(t *testing.T) string {
	t.Helper()
	argv := filepath.Join(t.TempDir(), "argv.log")
	t.Setenv("FAKE_ARGV", argv)
	return argv
}

func currentDir(t *testing.T, base string) string {
	t.Helper()
	hash, err := helperHash()
	if err != nil {
		t.Fatal(err)
	}
	return cacheDir(base, "9.9.9", hash)
}

func readLines(path string) []string {
	calls, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(calls)), "\n")
}

func TestEnsureHelper(t *testing.T) {
	dart := writeFakeDart(t)
	testCases := []struct {
		name    string
		fail    string
		wantPub []string
	}{
		{name: "builds and reuses", wantPub: []string{"pub get --offline"}},
		{name: "version on stderr", fail: "stderrversion", wantPub: []string{"pub get --offline"}},
		{
			name: "offline falls back online", fail: "offline",
			wantPub: []string{"pub get --offline", "pub get"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			base := t.TempDir()
			t.Setenv("FAKE_FAIL", testCase.fail)
			log, argv := fakeLog(t), fakeArgv(t)
			exe, err := ensureHelper(context.Background(), base, dart)
			if err != nil {
				t.Fatalf("ensureHelper: %v", err)
			}
			assertQuiet(t, readLines(argv), len(testCase.wantPub)+2)
			if want := filepath.Join(currentDir(t, base), exeName()); exe != want {
				t.Fatalf("exe = %q, want %q", exe, want)
			}
			if got := readLines(log); !slices.Equal(got, testCase.wantPub) {
				t.Errorf("pub calls = %q, want %q", got, testCase.wantPub)
			}
			t.Setenv("FAKE_FAIL", "compile")
			again, err := ensureHelper(context.Background(), base, dart)
			if err != nil || again != exe {
				t.Errorf("reuse = %q, %v; want %q", again, err, exe)
			}
		})
	}
}

func assertQuiet(t *testing.T, calls []string, wantCalls int) {
	t.Helper()
	if len(calls) != wantCalls {
		t.Errorf("dart ran %d steps, want %d: %q", len(calls), wantCalls, calls)
	}
	for _, call := range calls {
		if !strings.HasPrefix(call, "--suppress-analytics ") {
			t.Errorf("dart step lacks --suppress-analytics: %q", call)
		}
	}
}

type ensureErrorCase struct {
	name       string
	fail       string
	timeout    time.Duration
	wantErr    []string
	wantNoPub  bool
	wantMarker bool
}

func ensureErrorCases() []ensureErrorCase {
	deadline := context.DeadlineExceeded.Error()
	return []ensureErrorCase{
		{name: "version fails", fail: "version", wantErr: []string{"dart --version"}},
		{name: "version missing", fail: "noversion", wantErr: []string{"no version"}},
		{
			name: "sdk too old", fail: "oldsdk", wantNoPub: true,
			wantErr: []string{"Dart SDK 3.11+ required, found 3.9.0"},
		},
		{
			name: "both pub steps are reported", fail: "pub", wantMarker: true,
			wantErr: []string{"dart pub get offline", "dart pub get:"},
		},
		{
			name: "compile fails", fail: "compile", wantMarker: true,
			wantErr: []string{"dart compile exe"},
		},
		{
			name: "parent deadline stops a hanging step", fail: "hang",
			timeout: time.Second, wantErr: []string{deadline},
		},
		{name: "deadline passed before start", timeout: -time.Second, wantErr: []string{deadline}},
	}
}

func TestEnsureHelperErrors(t *testing.T) {
	dart := writeFakeDart(t)
	for _, testCase := range ensureErrorCases() {
		t.Run(testCase.name, func(t *testing.T) {
			base := t.TempDir()
			t.Setenv("FAKE_FAIL", testCase.fail)
			log := fakeLog(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if testCase.timeout != 0 {
				ctx, cancel = context.WithTimeout(ctx, testCase.timeout)
				defer cancel()
			}
			start := time.Now()
			_, err := ensureHelper(ctx, base, dart)
			assertEnsureFailure(t, err, testCase.wantErr, time.Since(start))
			if testCase.wantNoPub && len(readLines(log)) != 0 {
				t.Errorf("pub was invoked: %q", readLines(log))
			}
			if _, marked := loadMarker(currentDir(t, base)); marked != testCase.wantMarker {
				t.Errorf("marker = %v, want %v", marked, testCase.wantMarker)
			}
		})
	}
}

func assertEnsureFailure(t *testing.T, err error, wantErr []string, elapsed time.Duration) {
	t.Helper()
	const budget = 4 * time.Second
	if err == nil {
		t.Fatal("ensureHelper succeeded, want an error")
	}
	for _, want := range wantErr {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want containing %q", err, want)
		}
	}
	if elapsed > budget {
		t.Errorf("ensureHelper took %v, want under %v", elapsed, budget)
	}
	if count := strings.Count(err.Error(), context.DeadlineExceeded.Error()); count > 1 {
		t.Errorf("deadline reported %d times in %v", count, err)
	}
}

type rememberCase struct {
	name       string
	context    func() (context.Context, context.CancelFunc)
	wantMarker bool
}

func rememberCases() []rememberCase {
	past := time.Now().Add(-time.Hour)
	return []rememberCase{
		{
			name: "live context",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantMarker: true,
		},
		{
			name: "parent cancel",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			name: "parent deadline",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), past)
			},
		},
		{
			name: "parent deadline inside the build bound",
			context: func() (context.Context, context.CancelFunc) {
				parent, cancelParent := context.WithDeadline(context.Background(), past)
				ctx, cancel := context.WithTimeoutCause(parent, time.Hour, errBuildBound)
				return ctx, func() { cancel(); cancelParent() }
			},
		},
		{
			name: "own build bound",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithTimeoutCause(context.Background(), 0, errBuildBound)
			},
			wantMarker: true,
		},
	}
}

func TestRememberFailure(t *testing.T) {
	t.Parallel()
	for _, testCase := range rememberCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := testCase.context()
			defer cancel()
			dir := filepath.Join(t.TempDir(), "current-0123456789abcdef")
			cause := errors.New("step failed")
			if err := rememberFailure(ctx, dir, cause); !errors.Is(err, cause) {
				t.Errorf("rememberFailure = %v, want the build error", err)
			}
			if _, marked := loadMarker(dir); marked != testCase.wantMarker {
				t.Errorf("marker = %v, want %v", marked, testCase.wantMarker)
			}
		})
	}
}

func TestEnsureHelperConcurrent(t *testing.T) {
	dart := writeFakeDart(t)
	t.Parallel()
	base := t.TempDir()
	const workers = 4
	exes := make([]string, workers)
	errs := make([]error, workers)
	var group sync.WaitGroup
	for index := range workers {
		group.Go(func() {
			exes[index], errs[index] = ensureHelper(context.Background(), base, dart)
		})
	}
	group.Wait()
	for index := range workers {
		if errs[index] != nil || exes[index] != exes[0] {
			t.Errorf("worker %d = %q, %v; want %q", index, exes[index], errs[index], exes[0])
		}
	}
	if _, err := os.Stat(exes[0]); err != nil {
		t.Errorf("exe missing: %v", err)
	}
	entries, err := os.ReadDir(cacheRoot(base))
	if err != nil || len(entries) != 1 {
		t.Errorf("cache entries = %v, %v; want only the published dir", entries, err)
	}
}

func TestEnsureHelperAlternatingSDKs(t *testing.T) {
	dart := writeFakeDart(t)
	base := t.TempDir()
	compiles := filepath.Join(t.TempDir(), "compiles.log")
	t.Setenv("FAKE_COMPILES", compiles)
	for _, version := range []string{"9.9.9", "8.8.8", "9.9.9", "8.8.8"} {
		t.Setenv("FAKE_VERSION", version)
		if _, err := ensureHelper(context.Background(), base, dart); err != nil {
			t.Fatalf("ensureHelper %s: %v", version, err)
		}
	}
	if got := readLines(compiles); !slices.Equal(got, []string{"9.9.9", "8.8.8"}) {
		t.Errorf("compiles = %q, want one per SDK", got)
	}
}

func TestTail(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 500) + "end"
	bullets := "a" + strings.Repeat("\u2022", 200)
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trims space", input: "  hi \n", want: "hi"},
		{name: "keeps the end", input: long, want: long[len(long)-400:]},
		{name: "never splits a rune", input: bullets, want: bullets[len(bullets)-399:]},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := tail([]byte(testCase.input))
			if got != testCase.want || !utf8.ValidString(got) {
				t.Errorf("tail = %q, want valid UTF-8 %q", got, testCase.want)
			}
		})
	}
}

func TestStepError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		result process.Result
		err    error
		want   string
	}{
		{
			name: "execx error unwraps to its cause",
			err: &process.Error{
				Name: "dart", Args: []string{"x"}, Stderr: "bad", Err: errors.New("boom"),
			},
			result: process.Result{Stderr: "bad"}, want: "dart run: boom: bad",
		},
		{name: "empty output has no trailing colon", want: "dart run: boom"},
		{
			name:   "whitespace-only output has no trailing colon",
			result: process.Result{Stderr: " \n", Stdout: "\t"}, want: "dart run: boom",
		},
		{
			name:   "stderr is appended",
			result: process.Result{Stderr: "bad"}, want: "dart run: boom: bad",
		},
		{
			name:   "stdout alone is appended",
			result: process.Result{Stdout: "out"}, want: "dart run: boom: out",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runErr := testCase.err
			if runErr == nil {
				runErr = errors.New("boom")
			}
			err := stepError(context.Background(), "run", testCase.result, runErr)
			if err.Error() != testCase.want {
				t.Errorf("stepError = %q, want %q", err, testCase.want)
			}
		})
	}
}

func TestEnsureHelperMarkerWindow(t *testing.T) {
	dart := writeFakeDart(t)
	t.Parallel()
	testCases := []struct {
		name    string
		age     time.Duration
		wantErr string
	}{
		{name: "fresh marker blocks a rebuild", age: time.Minute, wantErr: "not retrying"},
		{name: "expired marker rebuilds", age: failureWindow + time.Minute},
		{name: "marker dated in the future rebuilds", age: -time.Hour},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			dir := currentDir(t, base)
			writeMarker(t, dir, failureMarker{
				Key: filepath.Base(dir), At: time.Now().Add(-testCase.age), Message: "old failure",
			})
			exe, err := ensureHelper(context.Background(), base, dart)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) ||
					!strings.Contains(err.Error(), "old failure") || exeExists(dir) {
					t.Fatalf("ensureHelper = %q, %v; want the remembered failure", exe, err)
				}
				return
			}
			if err != nil || !exeExists(dir) {
				t.Fatalf("ensureHelper = %q, %v; want a rebuild", exe, err)
			}
			if _, statErr := os.Stat(markerPath(dir)); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("successful build left the marker: %v", statErr)
			}
		})
	}
}
