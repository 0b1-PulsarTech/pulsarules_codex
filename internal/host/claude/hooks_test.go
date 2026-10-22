package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestInstallHook(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir, Location{}); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	stat, err := os.Stat(script)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if stat.Mode().Perm()&0o100 == 0 {
		t.Errorf("script not executable: mode %v", stat.Mode().Perm())
	}
	if _, err = os.Stat(filepath.Join(claudeDir, "hooks", "README.md")); err != nil {
		t.Errorf("README not installed: %v", err)
	}
}

func TestInstallHook_ForeignReadmeBackup(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readme := filepath.Join(hooksDir, "README.md")
	foreign := "# My own notes about this hooks dir\n"
	if err := os.WriteFile(readme, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign readme: %v", err)
	}

	ledger, err := InstallHook(fakeTemplates(), claudeDir, Location{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	backupPath := readme + ownedfile.BackupSuffix
	wantMsg := ownedfile.BackupMessage(readme, backupPath)
	if len(ledger.BackedUp) != 1 || ledger.BackedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", ledger.BackedUp, wantMsg)
	}
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}

	secondLedger, err := InstallHook(fakeTemplates(), claudeDir, Location{})
	if err != nil {
		t.Fatalf("second InstallHook: %v", err)
	}
	if len(secondLedger.BackedUp) != 0 {
		t.Errorf(
			"second InstallHook backedUp = %v, want none (it owns the file now)",
			secondLedger.BackedUp,
		)
	}
}

func seedFile(tb testing.TB, path, content string) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		tb.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", path, err)
	}
}

func assertContent(tb testing.TB, path, want string) {
	tb.Helper()

	got, err := os.ReadFile(path) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		tb.Errorf("%q content = %q, want %q", path, got, want)
	}
}

func TestInstallHook_NextNumberedSlot(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	readme := filepath.Join(claudeDir, "hooks", "README.md")
	current := "# current notes\n"
	seedFile(t, readme, current)
	priorBackup := readme + ownedfile.BackupSuffix
	priorContent := "# an even earlier version of the user's notes\n"
	seedFile(t, priorBackup, priorContent)

	ledger, err := InstallHook(fakeTemplates(), claudeDir, Location{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	newBackup := readme + ownedfile.BackupSuffix + ".1"
	wantMsg := ownedfile.BackupMessage(readme, newBackup)
	if len(ledger.BackedUp) != 1 || ledger.BackedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", ledger.BackedUp, wantMsg)
	}
	assertContent(t, priorBackup, priorContent)
	assertContent(t, newBackup, current)
}

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
		filepath.Join(claudeDir, "hooks", reminderScriptAsset),
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

func TestRenderReminderScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		body      string
		separator rune
		want      string
		wantErr   bool
	}{
		{
			name:      "substitutes every path",
			body:      `bin={{.Bin}} skills={{.SkillsDirs}} log={{.Log}}`,
			separator: ':',
			want: `bin="$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli" ` +
				`skills="$CLAUDE_PROJECT_DIR/.claude/skills":` +
				`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills" ` +
				`log="$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
		},
		{
			name:      "a windows separator is quoted so the shell keeps one word",
			body:      `skills={{.SkillsDirs}}`,
			separator: ';',
			want: `skills="$CLAUDE_PROJECT_DIR/.claude/skills"";"` +
				`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills"`,
		},
		{
			name:      "malformed template fails to parse",
			body:      `{{.Bin`,
			separator: ':',
			wantErr:   true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, renderErr := renderReminderScript(
				"t", []byte(testCase.body), Location{}, testCase.separator,
			)
			if testCase.wantErr {
				if renderErr == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if renderErr != nil {
				t.Fatalf("renderReminderScript: %v", renderErr)
			}
			if string(got) != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
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
	script := filepath.Join(claudeDir, "hooks", reminderScriptAsset)
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
	script := filepath.Join(hooks, reminderScriptAsset)
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
