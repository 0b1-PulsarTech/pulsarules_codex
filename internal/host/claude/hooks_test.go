package claude

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestInstallHook_RealTemplate(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err = InstallHook(templates, claudeDir, Location{}); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	got, err := os.ReadFile(
		filepath.Join(claudeDir, "hooks", hookset.ScriptName),
	) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read rendered script: %v", err)
	}
	script := string(got)
	if strings.Contains(script, "{{") {
		t.Errorf("rendered script still carries a template placeholder:\n%s", script)
	}
	for _, want := range []string{
		`bin="$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"`,
		`export PULSARULES_SKILLS_DIR="$CLAUDE_PROJECT_DIR/.claude/skills":` +
			`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills"`,
		`export PULSARULES_LOG_PATH="$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("rendered script missing %q:\n%s", want, script)
		}
	}
}

type scriptRun struct {
	loc        Location
	projectDir string
	configDir  string
	hasBinary  bool
}

func runInstalledScript(t *testing.T, run scriptRun, claudeDir string) string {
	t.Helper()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	if _, err = InstallHook(templates, claudeDir, run.loc); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	record := filepath.Join(t.TempDir(), "record")
	stub := "#!/bin/sh\nprintf '%s\\n' \"$@\" \"$PULSARULES_PROJECT_DIR\" " +
		"\"$PULSARULES_SKILLS_DIR\" \"$PULSARULES_LOG_PATH\" > \"$RECORD\"\n"
	binary := filepath.Join(claudeDir, "bin", "pulsarules_cli")
	if !run.hasBinary {
		err = os.Remove(binary)
	} else {
		err = os.WriteFile(binary, []byte(stub), 0o700) //nolint:gosec // test stub.
	}
	if err != nil {
		t.Fatalf("prepare binary: %v", err)
	}
	script := filepath.Join(claudeDir, "hooks", hookset.ScriptName)
	cmd := exec.CommandContext(t.Context(), "bash", script, "post-edit")
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=/home/tester", "RECORD=" + record,
		"CLAUDE_PROJECT_DIR=" + run.projectDir, "CLAUDE_CONFIG_DIR=" + run.configDir,
	}
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("script failed: %v\n%s", runErr, out)
	}
	got, _ := os.ReadFile(record) //nolint:gosec // test fixture.
	return string(got)
}

func wantRecord(projectDir, skills, log string) string {
	return "hook\npost-edit\n" + projectDir + "\n" + skills + "\n" + log + "\n"
}

func seedProjectHook(t *testing.T, projectDir string, mode os.FileMode) {
	t.Helper()

	hooks := filepath.Join(projectDir, ".claude", "hooks")
	if err := os.MkdirAll(hooks, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte("#!/bin/sh\n")
	script := filepath.Join(hooks, hookset.ScriptName)
	if err := os.WriteFile(script, body, mode); err != nil { //nolint:gosec // test fixture.
		t.Fatalf("write project hook: %v", err)
	}
	if err := os.Chmod(script, mode); err != nil { //nolint:gosec // test fixture.
		t.Fatalf("chmod project hook: %v", err)
	}
}

//nolint:paralleltest // an exec racing a concurrent stub write fails with ETXTBSY.
func TestInstalledScript_Global(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	testCases := []struct {
		name          string
		projectIsHome bool
		projectMode   os.FileMode
		hasBinary     bool
		wantStub      bool
	}{
		{name: "runs the global binary", hasBinary: true, wantStub: true},
		{name: "defers to the project script", projectMode: 0o700, hasBinary: true},
		{
			name:        "defers to a project script that is not executable",
			projectMode: 0o644, hasBinary: true,
		},
		{name: "does not defer to itself", projectIsHome: true, hasBinary: true, wantStub: true},
		{name: "missing binary exits quietly"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			projectDir, claudeDir := t.TempDir(), filepath.Join(t.TempDir(), "claude")
			if testCase.projectIsHome {
				claudeDir = filepath.Join(projectDir, ".claude")
			}
			if testCase.projectMode != 0 {
				seedProjectHook(t, projectDir, testCase.projectMode)
			}
			run := scriptRun{
				loc: Location{GlobalDir: claudeDir}, projectDir: projectDir,
				hasBinary: testCase.hasBinary,
			}
			want := ""
			if testCase.wantStub {
				want = wantRecord(projectDir, projectDir+"/.claude/skills:"+claudeDir+"/skills",
					claudeDir+"/hook-execution.log")
			}
			if got := runInstalledScript(t, run, claudeDir); got != want {
				t.Errorf("stub record = %q, want %q", got, want)
			}
		})
	}
}

//nolint:paralleltest // an exec racing a concurrent stub write fails with ETXTBSY.
func TestInstalledScript_Project(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	testCases := []struct {
		name       string
		configDir  string
		wantGlobal string
	}{
		{name: "user skills dir follows HOME", wantGlobal: "/home/tester/.claude/skills"},
		{
			name:       "user skills dir follows CLAUDE_CONFIG_DIR",
			configDir:  "/cfg",
			wantGlobal: "/cfg/skills",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			projectDir := t.TempDir()
			run := scriptRun{projectDir: projectDir, configDir: testCase.configDir, hasBinary: true}
			want := wantRecord(projectDir, projectDir+"/.claude/skills:"+testCase.wantGlobal,
				projectDir+"/.claude/hook-execution.log")
			if got := runInstalledScript(
				t,
				run,
				filepath.Join(projectDir, ".claude"),
			); got != want {
				t.Errorf("stub record = %q, want %q", got, want)
			}
		})
	}
}

func TestRenderHooksBlock_MatchesGolden(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile("testdata/hooks_block.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got, err := RenderHooksBlock(Location{})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("hooks block differs from the golden:\n%s", got)
	}
}

func TestRenderHooksBlock_GlobalNamesTheInstalledScript(t *testing.T) {
	t.Parallel()

	raw, err := RenderHooksBlock(Location{GlobalDir: "/home/user/.claude"})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var block hookset.Block
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for event, groups := range block.Hooks {
		for _, group := range groups {
			command := group.Hooks[0].Command
			const prefix = `bash '/home/user/.claude/hooks/skill-router-reminder.sh' `
			if !strings.HasPrefix(command, prefix) {
				t.Errorf("%s command = %q, want prefix %q", event, command, prefix)
			}
			if strings.Contains(command, "CLAUDE_PROJECT_DIR") {
				t.Errorf("%s command = %q, must not depend on the project dir", event, command)
			}
		}
	}
}

func TestInstalledScript_MatchesGolden(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	testCases := []struct {
		name   string
		loc    Location
		golden string
	}{
		{name: "project", loc: Location{}, golden: "testdata/project_script.golden.sh"},
		{
			name:   "global",
			loc:    Location{GlobalDir: "/home/user/.claude"},
			golden: "testdata/global_script.golden.sh",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			claudeDir := filepath.Join(t.TempDir(), ".claude")
			if _, installErr := hookset.Install(
				templates, claudeDir, testCase.loc.scriptValues(':'),
			); installErr != nil {
				t.Fatalf("Install: %v", installErr)
			}
			got, readErr := os.ReadFile( //nolint:gosec // test fixture.
				filepath.Join(claudeDir, hookset.HooksSubdir, hookset.ScriptName),
			)
			if readErr != nil {
				t.Fatalf("read rendered script: %v", readErr)
			}
			want, goldenErr := os.ReadFile(testCase.golden)
			if goldenErr != nil {
				t.Fatalf("read golden: %v", goldenErr)
			}
			if string(got) != string(want) {
				t.Errorf("script differs from the golden:\n%s", got)
			}
		})
	}
}
