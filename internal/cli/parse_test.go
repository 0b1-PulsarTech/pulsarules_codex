package cli

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func TestParse_WithoutARunner(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no arguments", args: nil, want: ""},
		{name: "help word", args: []string{"help"}, want: "help"},
		{name: "short help", args: []string{"-h"}, want: "-h"},
		{name: "long help", args: []string{"--help", "--bogus"}, want: "--help"},
		{name: "version word", args: []string{"version"}, want: "version"},
		{name: "short version", args: []string{"-v"}, want: "-v"},
		{name: "long version", args: []string{"--version", "x"}, want: "--version"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			call, err := Parse(testCase.args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if call.Name != testCase.want || call.runner != nil {
				t.Errorf("Parse() = %q with runner %v, want %q with none",
					call.Name, call.runner, testCase.want)
			}
		})
	}
}

func TestParse_Failures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		args     []string
		wantHelp bool
	}{
		{name: "unknown command", args: []string{"frobnicate"}},
		{name: "empty command word", args: []string{""}},
		{name: "unknown flag", args: []string{"generate", "--bogus"}},
		{name: "unknown flag on a legacy command", args: []string{"list", "--bogus"}},
		{name: "help flag", args: []string{"generate", "-h"}, wantHelp: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(testCase.args, flags.Environment{}, io.Discard)
			if err == nil {
				t.Fatal("Parse() error = nil, want a failure")
			}
			if got := errors.Is(err, flag.ErrHelp); got != testCase.wantHelp {
				t.Errorf(
					"errors.Is(err, flag.ErrHelp) = %v, want %v: %v",
					got,
					testCase.wantHelp,
					err,
				)
			}
		})
	}
}

func TestParse_CommonFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		args         []string
		env          flags.Environment
		wantRoot     string
		wantLogLevel string
	}{
		{name: "unset", args: []string{"generate"}},
		{
			name:         "env supplies the log level",
			args:         []string{"generate"},
			env:          flags.Environment{LogLevel: "debug"},
			wantLogLevel: "debug",
		},
		{
			name:         "flags beat the env",
			args:         []string{"generate", "--root", "r", "--log-level", "error"},
			env:          flags.Environment{LogLevel: "debug"},
			wantRoot:     "r",
			wantLogLevel: "error",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			call, err := Parse(testCase.args, testCase.env, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if call.Name != "generate" || call.runner == nil || call.env != testCase.env {
				t.Fatalf("Parse() = %+v, want the generate runner and the caller's env", call)
			}
			if call.root != testCase.wantRoot || call.logLevel != testCase.wantLogLevel {
				t.Errorf("root %q log level %q, want %q and %q",
					call.root, call.logLevel, testCase.wantRoot, testCase.wantLogLevel)
			}
		})
	}
}

func TestParse_KeepsPositionalArguments(t *testing.T) {
	t.Parallel()

	call, err := Parse(
		[]string{"generate", "--out", "x", "rest", "--out", "y"},
		flags.Environment{},
		io.Discard,
	)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(call.args) != 3 || call.args[0] != "rest" {
		t.Errorf("Args = %q, want the arguments after the first positional", call.args)
	}
}

type failingWriter struct{ err error }

func (writer failingWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestParse_HelpWritesTheUsage(t *testing.T) {
	t.Parallel()

	for _, candidate := range table() {
		t.Run(candidate.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			_, err := Parse([]string{candidate.name, "-h"}, flags.Environment{}, &out)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("Parse() error = %v, want flag.ErrHelp", err)
			}
			if want := wantUsage(t); out.String() != want {
				t.Errorf("usage = %q, want the full usage text", out.String())
			}
		})
	}
}

func TestParse_UnprintableUsageIsAnError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("closed pipe")
	_, err := Parse([]string{"list", "-h"}, flags.Environment{}, failingWriter{err: wantErr})
	if !errors.Is(err, wantErr) || errors.Is(err, flag.ErrHelp) {
		t.Errorf("Parse() error = %v, want %v and not flag.ErrHelp", err, wantErr)
	}
}
