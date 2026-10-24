package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_GlobalCodexStaysInsideHomeAndCodexHome(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(t.TempDir(), "codex")
	claudeDir := filepath.Join(t.TempDir(), "claude")
	cmd := New(flags.Environment{Home: home, ConfigDir: claudeDir, CodexHome: codexHome})
	cmd.global = true
	cmd.target = []string{"codex"}
	cmd.skills = []string{"go-style"}
	cmd.languages = []string{"go"}
	cmd.noMCP = true

	var err error
	captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantPaths := []string{
		filepath.Join(home, ".agents", "skills", "go-style", "SKILL.md"),
		filepath.Join(codexHome, "AGENTS.md"),
		filepath.Join(codexHome, "hooks.json"),
		filepath.Join(codexHome, "hooks", "skill-router-reminder.sh"),
	}
	for _, path := range wantPaths {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("global codex install missing %q: %v", path, statErr)
		}
	}
	if _, statErr := os.Stat(claudeDir); statErr == nil {
		t.Errorf("a codex-only install touched the claude dir %q", claudeDir)
	}
	entries, readErr := os.ReadDir(home)
	if readErr != nil || len(entries) != 1 || entries[0].Name() != ".agents" {
		t.Errorf("home = %v, %v, want only .agents", entries, readErr)
	}
}

//nolint:paralleltest // captureStreams and the gopls stub bar t.Parallel.
func TestRun_ProjectCodexWritesTheProjectTree(t *testing.T) {
	stubGoplsSoClaudeMCPIsWired(t)
	home, projectDir := t.TempDir(), t.TempDir()
	cmd := New(flags.Environment{Home: home, CodexHome: filepath.Join(home, "codex")})
	cmd.project = projectDir
	cmd.target = []string{"codex"}
	cmd.skills = []string{"go-style"}
	cmd.languages = []string{"go"}

	var err error
	captureStreams(t, func() { err = testWiring(t).run(cmd) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantPaths := []string{
		filepath.Join(projectDir, ".agents", "skills", "go-style", "SKILL.md"),
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, ".codex", "hooks.json"),
		filepath.Join(projectDir, ".codex", "config.toml"),
	}
	for _, path := range wantPaths {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("project codex install missing %q: %v", path, statErr)
		}
	}
	if entries, readErr := os.ReadDir(home); readErr != nil || len(entries) != 0 {
		t.Errorf("home = %v, %v, want a project install to leave it empty", entries, readErr)
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_PrintHooksCodex(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "codex")
	testCases := []struct {
		name       string
		global     bool
		want       string
		wantAbsent string
	}{
		{"project walks up", false, "PULSARULES_HOOK_DIALECT=codex", "'" + codexHome},
		{
			"global is absolute",
			true,
			filepath.ToSlash(codexHome) + "/hooks/skill-router-reminder.sh",
			"",
		},
	}
	for _, testCase := range testCases {
		cmd := New(flags.Environment{Home: t.TempDir(), CodexHome: codexHome})
		cmd.global = testCase.global
		cmd.project = ""
		if !testCase.global {
			cmd.project = t.TempDir()
		}
		cmd.target = []string{"codex"}
		cmd.printHooks = true

		var err error
		stdout, _ := captureStreams(t, func() { err = testWiring(t).run(cmd) })
		if err != nil {
			t.Fatalf("%s: Run: %v", testCase.name, err)
		}
		if !strings.Contains(stdout, testCase.want) {
			t.Errorf("%s: stdout = %s, want %q", testCase.name, stdout, testCase.want)
		}
		if testCase.wantAbsent != "" && strings.Contains(stdout, testCase.wantAbsent) {
			t.Errorf(
				"%s: stdout names %q, want a path-free command",
				testCase.name,
				testCase.wantAbsent,
			)
		}
		if strings.Contains(stdout, ".claude") {
			t.Errorf("%s: a codex-only print named the claude dir: %s", testCase.name, stdout)
		}
	}
}
