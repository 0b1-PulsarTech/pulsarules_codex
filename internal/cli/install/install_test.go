package install

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type wiring struct {
	shared  host.Context
	targets *host.Registry
}

func testWiring(tb testing.TB) wiring {
	tb.Helper()

	index, assets, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("load knowledge: %v", err)
	}
	renderer, err := render.NewRenderer(assets)
	if err != nil {
		tb.Fatalf("new renderer: %v", err)
	}
	return wiring{
		shared:  host.Context{Templates: assets, Index: index, Renderer: renderer},
		targets: host.NewRegistry(),
	}
}

func (wired wiring) run(cmd *Command) error {
	return cmd.Run(wired.shared.Index, wired.shared.Templates, wired.shared.Renderer, wired.targets)
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestNewTargetContext_LayoutThenSkillsShareOneReader(t *testing.T) {
	wired := testWiring(t)
	catalog := wired.shared.Index.SkillsOrdered()
	cmd := &Command{interactive: true, languages: []string{"go"}, noMCP: true}
	cfg := installConfig{absBase: t.TempDir()}
	var prompts strings.Builder
	console := newPrompter(strings.NewReader("monorepo\n1,3\n"), &prompts)
	console.isTerminal = true

	var ctx host.Context
	var err error
	stdout, _ := captureStreams(t, func() {
		ctx, err = newTargetContext(cmd, cfg, wired.shared, console)
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

	wired := testWiring(t)
	const forcedLanguage = "go"
	cmd := &Command{
		project:   projectDir,
		skills:    []string{"go-style"},
		languages: []string{forcedLanguage},
	}

	err := wired.run(cmd)
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
	cmd := &Command{noGitHooks: false}

	output, _ := captureStreams(t, func() {
		if err := installPostTargets(
			cmd,
			installConfig{absBase: dir, gitHooks: []string{}},
		); err != nil {
			t.Fatalf("installPostTargets: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("installed git hooks:")) {
		t.Errorf("output = %q, must not print a git-hooks success line for an empty list", output)
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_PrintHooks(t *testing.T) {
	projectDir := t.TempDir()
	cmd := &Command{project: projectDir, printHooks: true}

	wired := testWiring(t)
	var runErr error
	stdout, _ := captureStreams(t, func() { runErr = wired.run(cmd) })
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
			cmd := &Command{}
			flagSet := flag.NewFlagSet("install", flag.ContinueOnError)
			cmd.Bind(flagSet)
			if err := flagSet.Parse(args[1:]); err != nil {
				t.Fatalf("Parse: %v", err)
			}

			wired := testWiring(t)
			err := wired.run(cmd)
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
			cmd := &Command{project: t.TempDir(), hooksScope: testCase.scope}
			cfg, err := resolveInstallConfig(cmd)
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

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_GlobalStaysInsideTheClaudeConfigDir(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(t.TempDir(), "claude")
	cmd := New(flags.Environment{Home: home, ConfigDir: configDir})
	cmd.global = true
	cmd.skills = []string{"go-style"}
	cmd.languages = []string{"go"}
	cmd.noMCP = true

	var err error
	stdout, _ := captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	skill := filepath.Join(configDir, "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skill); statErr != nil {
		t.Errorf("skill not written to the config dir: %v", statErr)
	}
	entries, readErr := os.ReadDir(home)
	if readErr != nil || len(entries) != 0 {
		t.Errorf("home directory = %v, %v, want it untouched", entries, readErr)
	}
	if !strings.Contains(stdout, "per repository; skipped under --global") {
		t.Errorf("stdout = %q, want the git hook skip note", stdout)
	}
	slashDir := filepath.ToSlash(configDir)
	settings := readFile(t, filepath.Join(configDir, "settings.json"))
	wantCommand := "bash '" + slashDir + "/hooks/skill-router-reminder.sh' post-edit"
	if !strings.Contains(settings, wantCommand) {
		t.Errorf("settings.json = %s, want the absolute quoted command %q", settings, wantCommand)
	}
	script := readFile(t, filepath.Join(configDir, "hooks", "skill-router-reminder.sh"))
	wantBin := "\nbin='" + slashDir + "/bin/pulsarules_cli'\n"
	if !strings.Contains(script, wantBin) {
		t.Errorf("hook script = %s, want the absolute %q", script, wantBin)
	}
}

func readFile(tb testing.TB, path string) string {
	tb.Helper()
	body, err := os.ReadFile(path) //nolint:gosec // test fixture path.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(body)
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_ProjectNeverWritesIntoTheUserDirs(t *testing.T) {
	home, configDir, projectDir := t.TempDir(), t.TempDir(), t.TempDir()
	cmd := New(flags.Environment{Home: home, ConfigDir: configDir})
	cmd.project = projectDir
	cmd.skills = []string{"go-style"}
	cmd.languages = []string{"go"}
	cmd.noMCP = true

	var err error
	captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	skill := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skill); statErr != nil {
		t.Errorf("skill not written to the project: %v", statErr)
	}
	for _, dir := range []string{home, configDir} {
		if entries, readErr := os.ReadDir(dir); readErr != nil || len(entries) != 0 {
			t.Errorf("%q = %v, %v, want a project install to leave it empty", dir, entries, readErr)
		}
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_PrintHooksGlobalIsAbsolute(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "claude")
	cmd := New(flags.Environment{Home: t.TempDir(), ConfigDir: configDir})
	cmd.global = true
	cmd.printHooks = true

	var err error
	stdout, _ := captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "bash '" + filepath.ToSlash(configDir) + "/hooks/skill-router-reminder.sh' stop"
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout = %s, want the absolute command %q", stdout, want)
	}
	if _, statErr := os.Stat(configDir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("--print-hooks created %q, stat err = %v", configDir, statErr)
	}
}

func TestRun_GlobalRefusesProjectOnlyTargetsBeforeInstalling(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cmd := New(flags.Environment{Home: home})
	cmd.global = true
	cmd.target = []string{"claude", "agents"}
	cmd.skills = []string{"go-style"}

	err := testWiring(t).run(cmd)
	if !errors.Is(err, host.ErrProjectOnly) || !strings.Contains(err.Error(), `target "agents"`) {
		t.Fatalf("Run = %v, want a refusal naming the agents target", err)
	}
	if entries, readErr := os.ReadDir(home); readErr != nil || len(entries) != 0 {
		t.Errorf(
			"home = %v, %v, want nothing installed for the claude target either",
			entries,
			readErr,
		)
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_GlobalRefusesBeforeAnyPrompt(t *testing.T) {
	home := t.TempDir()
	cmd := New(flags.Environment{Home: home})
	cmd.global = true
	cmd.interactive = true
	cmd.target = []string{"opencode"}

	var err error
	stdout, _ := captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if !errors.Is(err, host.ErrProjectOnly) || !strings.Contains(err.Error(), `target "opencode"`) {
		t.Fatalf("Run = %v, want a refusal naming the opencode target", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want no prompt before the refusal", stdout)
	}
}
