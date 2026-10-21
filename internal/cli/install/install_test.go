package install

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func newTestCollaborators(tb testing.TB) installCollaborators {
	tb.Helper()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		tb.Fatalf("DoInjections: %v", err)
	}
	deps, err := resolveInstallCollaborators(inj)
	if err != nil {
		tb.Fatalf("resolveInstallCollaborators: %v", err)
	}
	return deps
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestNewTargetContext_LayoutThenSkillsShareOneReader(t *testing.T) {
	deps := newTestCollaborators(t)
	catalog := deps.idx.SkillsOrdered()
	opts := &version.Options{Interactive: true, Languages: []string{"go"}, NoMCP: true}
	cfg := installConfig{absBase: t.TempDir()}
	var prompts strings.Builder
	console := newPrompter(strings.NewReader("monorepo\n1,3\n"), &prompts)
	console.isTerminal = true

	var ctx host.Context
	var err error
	stdout, _ := captureStreams(t, func() {
		ctx, err = newTargetContext(opts, cfg, deps, console)
	})
	if err != nil {
		t.Fatalf("newTargetContext: %v", err)
	}
	if !strings.Contains(stdout, `applied layout profile "monorepo"`) {
		t.Errorf("stdout = %q, want the monorepo layout applied", stdout)
	}
	for _, picked := range []knowledge.Skill{catalog[0], catalog[2]} {
		if !slices.Contains(ctx.IDs, picked.ID) {
			t.Errorf("ids = %v, want picked skill %q", ctx.IDs, picked.ID)
		}
	}
	if !strings.Contains(prompts.String(), "layout> ") ||
		!strings.Contains(prompts.String(), "skills> ") {
		t.Errorf("prompts = %q, want both questions on the prompter output", prompts.String())
	}
}

//nolint:paralleltest // stubGoplsSoClaudeMCPIsWired needs t.Setenv, which bars t.Parallel.
func TestRun_MalformedMCPJSON_ErrorNotDoubled(t *testing.T) {
	stubGoplsSoClaudeMCPIsWired(t)

	projectDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectDir, ".mcp.json"), []byte("{not valid json"), 0o600,
	); err != nil {
		t.Fatalf("seed malformed .mcp.json: %v", err)
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	const forcedLanguage = "go"
	opts := &version.Options{
		Command:   "install",
		Project:   projectDir,
		Skills:    []string{"go-style"},
		Languages: []string{forcedLanguage},
	}

	err := Run(inj, opts)
	if err == nil {
		t.Fatal("expected an error from a malformed .mcp.json")
	}
	if strings.Count(err.Error(), `install target "claude"`) > 1 {
		t.Errorf("error = %q, wraps `install target %q` more than once", err.Error(), "claude")
	}
}

func stubGoplsSoClaudeMCPIsWired(tb testing.TB) {
	tb.Helper()

	dir := tb.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "gopls"), []byte("#!/bin/sh\nexit 0\n"), 0o700,
	); err != nil {
		tb.Fatalf("write gopls stub: %v", err)
	}
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func captureStreams(tb testing.TB, fn func()) (stdout, stderr string) {
	tb.Helper()
	originalOut, originalErr := os.Stdout, os.Stderr
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout, os.Stderr = outWriter, errWriter
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()

	fn()

	for _, writer := range []*os.File{outWriter, errWriter} {
		if closeErr := writer.Close(); closeErr != nil {
			tb.Fatalf("close pipe writer: %v", closeErr)
		}
	}
	var outBuf, errBuf bytes.Buffer
	if _, copyErr := io.Copy(&outBuf, outReader); copyErr != nil {
		tb.Fatalf("read stdout pipe: %v", copyErr)
	}
	if _, copyErr := io.Copy(&errBuf, errReader); copyErr != nil {
		tb.Fatalf("read stderr pipe: %v", copyErr)
	}
	return outBuf.String(), errBuf.String()
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestInstallPostTargets_EmptyGitHooks(t *testing.T) {
	dir := t.TempDir()
	opts := &version.Options{NoGitHooks: false}

	output, _ := captureStreams(t, func() {
		if err := installPostTargets(
			opts,
			installConfig{absBase: dir, gitHooks: []string{}},
		); err != nil {
			t.Fatalf("installPostTargets: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("installed git hooks:")) {
		t.Errorf("output = %q, must not print a git-hooks success line for an empty list", output)
	}
}

func injectRun(tb testing.TB, inj remy.Injector) {
	tb.Helper()
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		tb.Fatalf("DoInjections: %v", err)
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_PrintHooks(t *testing.T) {
	projectDir := t.TempDir()
	opts := &version.Options{Command: "install", Project: projectDir, PrintHooks: true}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	injectRun(t, inj)
	var runErr error
	stdout, _ := captureStreams(t, func() { runErr = Run(inj, opts) })
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	if !strings.Contains(stdout, `"hooks"`) {
		t.Errorf("stdout = %q, want the hooks block", stdout)
	}
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		t.Fatalf("read project dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("--print-hooks wrote %d entries, want none", len(entries))
	}
}

func TestRun_RejectsAMalformedTargetBeforeWriting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		flags []string
		bad   string
	}{
		{"unknown name", []string{"bogus"}, "bogus"},
		{"two names in one value", []string{"claude,opencode"}, "claude,opencode"},
		{"padded name", []string{" opencode "}, " opencode "},
		{"empty value", []string{""}, ""},
		{"blank value", []string{" "}, " "},
		{"lone comma", []string{","}, ","},
		{"trailing comma", []string{"claude,"}, "claude,"},
		{"empty repeat", []string{"claude", ""}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			args := []string{"install", "--project", projectDir, "--skills", "go-style"}
			args = append(args, "--languages", "go")
			for _, value := range testCase.flags {
				args = append(args, "--target", value)
			}
			opts, err := version.ParseArgs(args)
			if err != nil {
				t.Fatalf("ParseArgs: %v", err)
			}

			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			injectRun(t, inj)
			err = Run(inj, opts)
			want := fmt.Sprintf("invalid --target %q (want ", testCase.bad)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("err = %v, want an error containing %q", err, want)
			}
			_, statErr := os.Stat(filepath.Join(projectDir, ".claude"))
			if !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("rejected install created .claude, stat err = %v", statErr)
			}
		})
	}
}

func TestResolveInstallConfig_SettingsFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		scope    string
		want     string
		wantText string
	}{
		{name: "empty means project", want: "settings.json"},
		{name: "project", scope: "project", want: "settings.json"},
		{name: "local", scope: "local", want: "settings.local.json"},
		{
			name:     "bogus is refused",
			scope:    "bogus",
			wantText: `settings file: invalid --hooks-scope "bogus" (want project|local)`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts := &version.Options{Project: t.TempDir(), HooksScope: testCase.scope}
			cfg, err := resolveInstallConfig(opts)
			if testCase.wantText != "" {
				if err == nil || err.Error() != testCase.wantText {
					t.Fatalf("err = %v, want %q", err, testCase.wantText)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveInstallConfig: %v", err)
			}
			if cfg.settingsFile != testCase.want {
				t.Errorf("settingsFile = %q, want %q", cfg.settingsFile, testCase.want)
			}
		})
	}
}
