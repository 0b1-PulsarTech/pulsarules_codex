package clean

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestCommand_RepositoryResolution(t *testing.T) {
	t.Parallel()

	wiringErr := errors.New("boom: repository wiring failed")
	testCases := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{"wiring failure is returned", wiringErr, wiringErr},
		{"no repository falls back to the walk", vcs.ErrNoRepository, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			open := func() (*vcs.Repository, error) { return nil, testCase.repoErr }

			err := boundCommand(t, "--project", t.TempDir()).Run(open)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Run() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCommand_Bind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		args        []string
		wantProject string
		wantWrite   bool
	}{
		{name: "defaults", wantProject: "."},
		{name: "project", args: []string{"--project", "p"}, wantProject: "p"},
		{
			name:        "empty project is the working directory",
			args:        []string{"--project", ""},
			wantProject: ".",
		},
		{name: "write", args: []string{"--write"}, wantProject: ".", wantWrite: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var cmd Command
			fs := flag.NewFlagSet("clean", flag.ContinueOnError)
			cmd.Bind(fs)
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cmd.ProjectDir() != testCase.wantProject || cmd.write != testCase.wantWrite {
				t.Errorf("ProjectDir() = %q write %v, want %q and %v",
					cmd.ProjectDir(), cmd.write, testCase.wantProject, testCase.wantWrite)
			}
		})
	}
}

//nolint:paralleltest // The command prints to os.Stdout, which the capture swaps.
func TestCommand_Run_OnlyWriteRewrites(t *testing.T) {
	testCases := []struct {
		name        string
		write       bool
		wantVerb    string
		wantRewrite bool
	}{
		{name: "without write only inspects", wantVerb: "found"},
		{name: "with write rewrites", write: true, wantVerb: "left", wantRewrite: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "notes.md")
			const original = "a\u200Bb\n"
			if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			open := func() (*vcs.Repository, error) { return nil, vcs.ErrNoRepository }

			out := captureStdout(t, func() {
				args := []string{"--project", dir}
				if testCase.write {
					args = append(args, "--write")
				}
				if err := boundCommand(t, args...).Run(open); err != nil {
					t.Errorf("Run() = %v, want nil", err)
				}
			})

			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if rewritten := string(body) != original; rewritten != testCase.wantRewrite {
				t.Errorf("file rewritten = %v, want %v", rewritten, testCase.wantRewrite)
			}
			if !strings.Contains(out, testCase.wantVerb) {
				t.Errorf("output = %q, want it to say %q", out, testCase.wantVerb)
			}
		})
	}
}

func boundCommand(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd := &Command{}
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	cmd.Bind(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return cmd
}

func captureStdout(t *testing.T, body func()) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "capture")
	if err != nil {
		t.Fatalf("create capture: %v", err)
	}
	saved := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = saved }()
	body()
	if _, err = file.Seek(0, 0); err != nil {
		t.Fatalf("rewind capture: %v", err)
	}
	captured, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	return string(captured)
}
