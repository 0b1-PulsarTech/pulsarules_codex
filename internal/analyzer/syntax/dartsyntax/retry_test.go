package dartsyntax

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func fakeHelperPrepare(
	t *testing.T, script string, removals int, calls *int,
) func(context.Context) (string, error) {
	t.Helper()
	removed := 0
	return func(context.Context) (string, error) {
		*calls++
		exe := writeFakeHelper(t, script)
		if removed < removals {
			removed++
			return exe, os.Remove(exe)
		}
		return exe, nil
	}
}

type retryCase struct {
	name         string
	script       string
	tempSubdir   string
	removals     int
	wantPrepares int
	wantErr      bool
	cancelAfter  time.Duration
	wantCanceled bool
}

func retryCases() []retryCase {
	const okScript = "echo '{\"lib/a.dart\": {}}'\n"
	const vanishOnceScript = "if [ ! -e \"$FAKE_ARGV.mark\" ]; then\n" +
		"touch \"$FAKE_ARGV.mark\"; rm \"$0\"; exit 3\nfi\n" + okScript
	return []retryCase{
		{
			name: "pruned once between prepare and run", script: okScript,
			removals: 1, wantPrepares: 2,
		},
		{
			name: "pruned on both attempts", script: okScript,
			removals: 2, wantPrepares: 2, wantErr: true,
		},
		{name: "never pruned", script: okScript, wantPrepares: 1},
		{
			name: "helper exits non-zero", script: "exit 3\n",
			wantPrepares: 1, wantErr: true,
		},
		{
			name: "exe vanished during the run", script: "rm \"$0\"; exit 3\n",
			wantPrepares: 2, wantErr: true,
		},
		{
			name:         "exe vanished during the first run only",
			script:       vanishOnceScript,
			wantPrepares: 2,
		},
		{
			name:         "exe vanished after a good run",
			script:       "rm \"$0\"\n" + okScript,
			wantPrepares: 1,
		},
		{
			name: "cancelled while the exe vanishes", script: "rm \"$0\"; sleep 5\n",
			cancelAfter: 300 * time.Millisecond, wantPrepares: 1,
			wantErr: true, wantCanceled: true,
		},
		{
			name: "request file cannot be created", script: okScript, tempSubdir: "absent",
			wantPrepares: 1, wantErr: true,
		},
	}
}

func cancelAfter(t *testing.T, delay time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if delay > 0 {
		time.AfterFunc(delay, cancel)
	}
	return ctx
}

func checkRetryError(t *testing.T, testCase retryCase, files map[string]*syntax.File, err error) {
	t.Helper()
	if got := errors.Is(err, context.Canceled); got != testCase.wantCanceled {
		t.Errorf("Is(Canceled) = %v, want %v; err %v", got, testCase.wantCanceled, err)
	}
	if testCase.wantCanceled && err != nil && strings.Contains(err.Error(), "prepare") {
		t.Errorf("error blames the retry prepare: %v", err)
	}
	if (err != nil) != testCase.wantErr || (err == nil && files["lib/a.dart"] == nil) {
		t.Errorf("Summarize = %v, %v; want error %v", files, err, testCase.wantErr)
	}
}

func TestSummarizeRetries(t *testing.T) {
	for _, testCase := range retryCases() {
		t.Run(testCase.name, func(t *testing.T) {
			argv := filepath.Join(t.TempDir(), "argv.log")
			t.Setenv("FAKE_ARGV", argv)
			if testCase.tempSubdir != "" {
				t.Setenv("TMPDIR", filepath.Join(t.TempDir(), testCase.tempSubdir))
			}
			calls := 0
			prepare := fakeHelperPrepare(t, testCase.script, testCase.removals, &calls)
			requests := []request{{Path: "lib/a.dart", Content: "void main() {}"}}
			ctx := cancelAfter(t, testCase.cancelAfter)
			files, err := summarizeWith(ctx, requests, prepare)
			checkRetryError(t, testCase, files, err)
			if calls != testCase.wantPrepares {
				t.Errorf("prepare calls = %d, want %d", calls, testCase.wantPrepares)
			}
		})
	}
}
