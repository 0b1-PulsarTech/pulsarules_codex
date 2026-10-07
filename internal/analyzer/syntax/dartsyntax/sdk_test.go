package dartsyntax

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func TestParseSDKVersion(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{name: "stable", output: "Dart SDK version: 3.13.2 (stable)", want: "3.13.2"},
		{name: "prerelease", output: "Dart SDK version: 3.12.0-dev.1 (dev)", want: "3.12.0-dev.1"},
		{name: "vm label", output: "Dart VM version: 3.12.0 (stable)", want: "3.12.0"},
		{name: "garbage", output: "nothing here", wantErr: true},
		{name: "analytics decoy alone", output: "Analytics version: 2.0", wantErr: true},
		{
			name:   "analytics decoy first",
			output: "Analytics version: 2.0\nDart SDK version: 3.13.2 (stable)", want: "3.13.2",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSDKVersion(testCase.output)
			if (err != nil) != testCase.wantErr || got != testCase.want {
				t.Errorf("parseSDKVersion = %q, %v; want %q, wantErr %v",
					got, err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestCheckSDK(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		version string
		wantErr string
	}{
		{name: "exact floor", version: "3.11.0"},
		{name: "newer minor", version: "3.13.2"},
		{name: "newer major", version: "4.0.0"},
		{name: "prerelease of a newer minor", version: "3.12.0-dev.1"},
		{
			name: "prerelease at floor", version: "3.11.0-123.dev",
			wantErr: "found 3.11.0-123.dev",
		},
		{name: "patch above floor", version: "3.11.1"},
		{name: "older minor", version: "3.10.9", wantErr: "found 3.10.9"},
		{name: "older major", version: "2.19.6", wantErr: "Dart SDK 3.11+ required, found 2.19.6"},
		{name: "unparsable", version: "weird", wantErr: "found weird"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := checkSDK(testCase.version)
			if testCase.wantErr == "" && err != nil {
				t.Fatalf("checkSDK = %v, want nil", err)
			}
			missing := err == nil || !strings.Contains(err.Error(), testCase.wantErr)
			if testCase.wantErr != "" && missing {
				t.Fatalf("checkSDK = %v, want containing %q", err, testCase.wantErr)
			}
		})
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
