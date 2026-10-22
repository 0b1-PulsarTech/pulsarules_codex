package uninstall

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRun_RoundTrip(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)

	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	skillDoc := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skillDoc); statErr != nil {
		t.Fatalf("install did not write %q: %v", skillDoc, statErr)
	}
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); statErr != nil {
		t.Fatalf("install did not wire %q: %v", settingsPath, statErr)
	}

	uninstallOpts := &Command{project: projectDir}
	if err := uninstallOpts.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(skillDoc); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", skillDoc, statErr)
	}
	if _, statErr := os.Stat(settingsPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", settingsPath, statErr)
	}

	t.Run("idempotent: second uninstall is not an error", func(t *testing.T) {
		t.Parallel()
		if err := uninstallOpts.Run(wired.targets); err != nil {
			t.Fatalf("second Run: %v", err)
		}
	})
}

func TestRun_KeepSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)

	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	uninstallOpts := &Command{project: projectDir, keepSkills: true}
	if err := uninstallOpts.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}

	skillDoc := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skillDoc); statErr != nil {
		t.Errorf("--keep-skills should have kept %q, stat err = %v", skillDoc, statErr)
	}
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected hook wiring to be removed regardless, stat err = %v", statErr)
	}
}

func TestRun_RemovesLanguageConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectDir, "go.mod"),
		[]byte("module x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	wired := testWiring(t)

	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	configPath := filepath.Join(projectDir, detect.ConfigFileName)
	if _, statErr := os.Stat(configPath); statErr != nil {
		t.Fatalf("install did not write %q: %v", configPath, statErr)
	}

	uninstallOpts := &Command{project: projectDir}
	if err := uninstallOpts.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(configPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", configPath, statErr)
	}
}

func TestRun_Idempotent(t *testing.T) {
	t.Parallel()

	wired := testWiring(t)
	opts := &Command{project: t.TempDir()}
	if err := opts.Run(wired.targets); err != nil {
		t.Fatalf("Run on untouched project: %v", err)
	}
}

func TestRun_DetectsBothTargetsWhenNoneGiven(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)

	installArgs := []string{
		"--project",
		projectDir,
		"--skills",
		"go-style",
		"--target",
		"claude",
		"--target",
		"opencode",
		"--no-mcp",
	}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	claudeSkill := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	opencodeConfig := filepath.Join(projectDir, "opencode.json")
	for _, path := range []string{claudeSkill, opencodeConfig} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("install did not write %q: %v", path, statErr)
		}
	}

	uninstallOpts := &Command{project: projectDir}
	if err := uninstallOpts.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, path := range []string{claudeSkill, opencodeConfig} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed, stat err = %v", path, statErr)
		}
	}
}

func TestRun_UnwiresLocalHooksScopeByDefault(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)

	installArgs := []string{
		"--project",
		projectDir,
		"--skills",
		"go-style",
		"--no-mcp",
		"--hooks-scope",
		"local",
	}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	localSettings := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, statErr := os.Stat(localSettings); statErr != nil {
		t.Fatalf("install did not wire %q: %v", localSettings, statErr)
	}

	uninstallOpts := &Command{project: projectDir}
	if err := uninstallOpts.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(localSettings); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", localSettings, statErr)
	}
}

func TestRun_ReturnsErrorOnGitHookFailure(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)

	hookAsDir := filepath.Join(
		projectDir,
		".git",
		"hooks",
		"commit-msg",
	)
	if err := os.MkdirAll(hookAsDir, 0o750); err != nil {
		t.Fatalf("mkdir %q: %v", hookAsDir, err)
	}

	opts := &Command{project: projectDir}
	if err := opts.Run(wired.targets); err == nil {
		t.Fatal("expected a non-nil error when git hook removal fails")
	}
}

func TestRun_TargetErrorNotDoubled(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)
	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("remove settings.json: %v", err)
	}
	if err := os.MkdirAll(settingsPath, 0o750); err != nil {
		t.Fatalf("replace settings.json with a directory: %v", err)
	}

	uninstallOpts := &Command{project: projectDir}
	err := uninstallOpts.Run(wired.targets)
	if err == nil {
		t.Fatal("expected an error when settings.json cannot be read")
	}
	if strings.Count(err.Error(), `uninstall target "claude"`) > 1 {
		t.Errorf("error = %q, wraps `uninstall target %q` more than once", err.Error(), "claude")
	}
}

func captureStdout(tb testing.TB, fn func()) string {
	tb.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()

	if closeErr := writer.Close(); closeErr != nil {
		tb.Fatalf("close pipe writer: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, reader); copyErr != nil {
		tb.Fatalf("read pipe: %v", copyErr)
	}
	return buf.String()
}

//nolint:paralleltest // captureStdout swaps os.Stdout.
func TestRemoveLanguageConfig(t *testing.T) {
	testCases := []struct {
		name       string
		seed       func(tb testing.TB, configPath string)
		wantStdout string
		wantErr    bool
	}{
		{
			name: "present file is removed and reported",
			seed: func(tb testing.TB, configPath string) {
				tb.Helper()
				if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
					tb.Fatalf("seed config: %v", err)
				}
			},
			wantStdout: "removed " + detect.ConfigFileName + "\n",
		},
		{
			name: "absent file prints nothing",
			seed: func(testing.TB, string) {},
		},
		{
			name: "unremovable config is an error",
			seed: func(tb testing.TB, configPath string) {
				tb.Helper()
				if err := os.MkdirAll(filepath.Join(configPath, "child"), 0o750); err != nil {
					tb.Fatalf("seed directory: %v", err)
				}
			},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			testCase.seed(t, filepath.Join(dir, detect.ConfigFileName))

			var err error
			stdout := captureStdout(t, func() { err = removeLanguageConfig(dir) })
			if (err != nil) != testCase.wantErr {
				t.Errorf("removeLanguageConfig err = %v, wantErr %v", err, testCase.wantErr)
			}
			if stdout != testCase.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, testCase.wantStdout)
			}
		})
	}
}

func forgedLines(text string) []string {
	var found []string
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "[ERROR]") {
			found = append(found, line)
		}
	}
	return found
}

func TestRun_HostileTreeStaysOnOneLine(t *testing.T) {
	t.Parallel()

	const forgedLine = "[ERROR] forged"

	projectDir := t.TempDir()
	skillDir := filepath.Join(projectDir, ".claude", "skills", "x\n"+forgedLine+" by repo")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Skipf("filesystem refuses the name: %v", err)
	}
	marked := []byte("<!-- " + ownedfile.Marker + " -->\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), marked, 0o600); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	settings := `{"hooks": {"PreToolUse\n` + forgedLine + ` via key": 5}}`
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settings), 0o600); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	report, _ := testWiring(t).targets.Uninstall("claude", host.UninstallContext{
		Base:          projectDir,
		SettingsFiles: []string{"settings.json"},
	})

	var stdout, stderr strings.Builder
	report.Print(&stdout, &stderr)
	if len(report.Notes) == 0 || len(report.Warnings) == 0 {
		t.Fatalf("want notes and warnings, got %+v", report)
	}
	streams := map[string]string{"stdout": stdout.String(), "stderr": stderr.String()}
	for name, text := range streams {
		if forged := forgedLines(text); len(forged) > 0 {
			t.Errorf("%s has forged lines %q in %q", name, forged, text)
		}
	}
}

func TestUninstallContext_SettingsFiles(t *testing.T) {
	t.Parallel()

	opts := &Command{project: t.TempDir(), hooksScope: "local"}
	ctx, err := uninstallContext(opts)
	if err != nil || !slices.Equal(ctx.SettingsFiles, []string{"settings.local.json"}) {
		t.Fatalf("uninstallContext = %v, %v, want the local file", ctx.SettingsFiles, err)
	}

	opts.hooksScope = "bogus"
	_, err = uninstallContext(opts)
	const want = `settings files: invalid --hooks-scope "bogus" (want project|local)`
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func runInstall(tb testing.TB, wired wiring, args []string) error {
	tb.Helper()

	cmd := &install.Command{}
	flagSet := flag.NewFlagSet("install", flag.ContinueOnError)
	cmd.Bind(flagSet)
	if err := flagSet.Parse(args); err != nil {
		tb.Fatalf("Parse(%q): %v", args, err)
	}
	return cmd.Run(wired.index, wired.templates, wired.renderer, wired.targets)
}

type wiring struct {
	index     *knowledge.Index
	templates fs.FS
	renderer  *render.Renderer
	targets   *host.Registry
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
	return wiring{index, assets, renderer, host.NewRegistry()}
}

func TestRun_GlobalLeavesGitAndLanguageConfigAlone(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	wired := testWiring(t)
	languageConfig := filepath.Join(home, detect.ConfigFileName)
	foreignHook := filepath.Join(home, ".git", "hooks", "pre-commit")
	for path, body := range map[string]string{languageConfig: `{}`, foreignHook: "#!/bin/sh\n"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("seed %q: %v", path, err)
		}
	}

	cmd := New(flags.Environment{Home: home})
	cmd.global = true
	if err := cmd.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, kept := range []string{languageConfig, foreignHook} {
		if _, statErr := os.Stat(kept); statErr != nil {
			t.Errorf("%q was removed by a global uninstall: %v", kept, statErr)
		}
	}
}

func TestResolveTargets_GlobalDefaultsToClaude(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "AGENTS.md"), []byte("mine"), 0o600); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}
	got, err := resolveTargets(&Command{global: true}, host.NewRegistry(), home)
	if err != nil || !slices.Equal(got, []string{"claude"}) {
		t.Errorf("resolveTargets = %v, %v, want [claude]", got, err)
	}
}
