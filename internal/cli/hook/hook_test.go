package hook

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func TestPrintHookError_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printHookError(&out, "get templates", errors.New("open a\n[ERROR] forged"))
	want := `hook: get templates: open a\n[ERROR] forged` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

func writeHookFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), disk.File); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestHookStop_RealGovernance(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	index, assets, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeHookFile(t, dir, "violation.go", "package x\n\n// bad note \u2014 here\n")
	var out bytes.Buffer
	disp := newHookDispatcher(
		hook.Deps{Templates: templates.New(assets), Index: index, Out: &out, ProjectDir: dir},
		func() (*emoji.Catalog, error) { return testEmojiCatalog(t), nil },
	)

	_ = disp.Dispatch("stop", []byte(`{"session_id":"real-governance"}`))
	for _, want := range []string{
		"Governance checks:", "typographic-markers", "go vet", "violation.go",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stop output missing %q:\n%s", want, out.String())
		}
	}
}

func testEmojiCatalog(t *testing.T) *emoji.Catalog {
	t.Helper()

	catalog, err := emoji.NewCatalog()
	if err != nil {
		t.Fatalf("emoji.NewCatalog: %v", err)
	}
	return catalog
}

func captureStderr(t *testing.T, body func()) string {
	t.Helper()

	return captureStream(t, &os.Stderr, body)
}

func captureStream(t *testing.T, stream **os.File, body func()) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "capture")
	if err != nil {
		t.Fatalf("create capture: %v", err)
	}
	saved := *stream
	*stream = file
	defer func() { *stream = saved }()
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

func TestHookStop_CatalogFailureSkipsGovernance(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	index, assets, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeHookFile(t, dir, "violation.go", "package x\n\n// bad note \u2014 here\n")
	var out bytes.Buffer
	disp := newHookDispatcher(
		hook.Deps{Templates: templates.New(assets), Index: index, Out: &out, ProjectDir: dir},
		func() (*emoji.Catalog, error) { return nil, errors.New("catalog unavailable") },
	)

	stderr := captureStderr(t, func() {
		_ = disp.Dispatch("stop", []byte(`{"session_id":"catalog-failure"}`))
	})

	if !strings.Contains(stderr, "hook: get emoji catalog: catalog unavailable") {
		t.Errorf("stderr = %q, want the catalog failure", stderr)
	}
	if strings.Contains(out.String(), "Governance checks:") {
		t.Errorf("stop ran governance without a catalog:\n%s", out.String())
	}
}

//nolint:paralleltest // captureStderr swaps os.Stderr.
func TestCommand_Run_OtherModesDoNotNeedTheCatalog(t *testing.T) {
	svc := hookServices(t)
	stderr := captureStderr(t, func() {
		if err := New(
			flags.Environment{ProjectDir: t.TempDir()},
		).Run(svc, []string{"pre-search"}); err != nil {
			t.Errorf("Run(pre-search) = %v, want nil", err)
		}
	})

	if strings.Contains(stderr, "emoji catalog") {
		t.Errorf("pre-search resolved the emoji catalog: %q", stderr)
	}
}

//nolint:paralleltest // t.Setenv and the os.Stdin/os.Stdout swaps bar t.Parallel.
func TestCommand_Run_PassesTheCallersEnvironmentToTheDispatcher(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	project, skills := t.TempDir(), t.TempDir()
	writeHookFile(t, project, "go.mod", "module x\n")
	if err := os.MkdirAll(
		filepath.Join(skills, "gopls-navigation"),
		disk.DirPrivate,
	); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	writeHookFile(t, skills, "gopls-navigation/SKILL.md", "x")
	payload := `{"session_id":"SID","tool_name":"Grep","tool_input":{"pattern":"a.go"}}`

	testCases := []struct {
		name string
		env  flags.Environment
		want bool
	}{
		{"both given", flags.Environment{ProjectDir: project, SkillsDir: skills}, true},
		{"no skills dir", flags.Environment{ProjectDir: project}, false},
		{"no project dir", flags.Environment{SkillsDir: skills}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svc := hookServices(t)
			stdin, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatalf("create stdin: %v", err)
			}
			session := strings.Replace(payload, "SID", testCase.name, 1)
			if _, err = stdin.WriteString(session); err != nil {
				t.Fatalf("write stdin: %v", err)
			}
			if _, err = stdin.Seek(0, 0); err != nil {
				t.Fatalf("rewind stdin: %v", err)
			}
			savedStdin := os.Stdin
			os.Stdin = stdin
			defer func() { os.Stdin = savedStdin }()

			out := captureStream(t, &os.Stdout, func() {
				if err = New(testCase.env).Run(svc, []string{"pre-search"}); err != nil {
					t.Errorf("Run(pre-search) = %v, want nil", err)
				}
			})
			if got := out != ""; got != testCase.want {
				t.Errorf("pre-search emitted = %v, want %v:\n%s", got, testCase.want, out)
			}
		})
	}
}

func hookServices(t *testing.T) bootstrap.Services {
	t.Helper()

	index, assets, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return bootstrap.Services{
		Index:     index,
		Templates: templates.New(assets),
		OpenLogger: func() (bootstrap.Logging, error) {
			return bootstrap.Logging{
				Logger: slog.New(slog.DiscardHandler),
				Closer: io.NopCloser(nil),
			}, nil
		},
		LoadCatalog: func() (*emoji.Catalog, error) {
			return nil, errors.New("emoji catalog must not be resolved here")
		},
	}
}

type closeRecorder struct {
	closed *bool
}

func (rec closeRecorder) Close() error {
	*rec.closed = true
	return nil
}

//nolint:paralleltest // captureStderr swaps os.Stderr.
func TestCommand_Run_NeverFailsOnALoggerFailure(t *testing.T) {
	svc := hookServices(t)
	svc.OpenLogger = func() (bootstrap.Logging, error) {
		return bootstrap.Logging{}, errors.New("build logger: boom")
	}
	var err error
	stderr := captureStderr(t, func() {
		err = New(flags.Environment{}).Run(svc, []string{"pre-search"})
	})
	if err != nil {
		t.Errorf("Run() = %v, want nil so the turn is never blocked", err)
	}
	if want := "hook: build logger: build logger: boom\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

//nolint:paralleltest // captureStderr swaps os.Stderr.
func TestCommand_Run_ClosesTheLogger(t *testing.T) {
	var closed bool
	svc := hookServices(t)
	svc.OpenLogger = func() (bootstrap.Logging, error) {
		return bootstrap.Logging{
			Logger: slog.New(slog.DiscardHandler),
			Closer: closeRecorder{closed: &closed},
		}, nil
	}
	captureStderr(t, func() { _ = New(flags.Environment{}).Run(svc, []string{"pre-search"}) })
	if !closed {
		t.Error("Run left the logger open")
	}
}

func TestCommand_BootstrapFailed(t *testing.T) {
	t.Parallel()

	forged := fmt.Errorf("open %s: %w", "a\n[ERROR] forged", os.ErrNotExist)
	var out strings.Builder
	if got := BootstrapFailed(&out, forged); got != 0 {
		t.Errorf("BootstrapFailed() = %d, want 0 so a hook never blocks the turn", got)
	}
	text := strings.TrimSuffix(out.String(), "\n")
	if strings.Contains(text, "\n") || !strings.Contains(text, `a\n[ERROR] forged`) ||
		!strings.HasPrefix(text, "pulsarules_cli: hook: ") {
		t.Errorf("output = %q, want one escaped line", out.String())
	}
}
