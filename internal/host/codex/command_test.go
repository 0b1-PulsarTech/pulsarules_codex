package codex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type commandWorld struct {
	project   string
	home      string
	globalDir string
	nested    string
	outside   string
}

type worldSetup struct {
	withProject         bool
	globalInHome        bool
	withoutProjectBin   bool
	withoutGlobalScript bool
}

func newCommandWorld(t *testing.T, setup worldSetup) commandWorld {
	t.Helper()
	world := commandWorld{project: t.TempDir(), home: t.TempDir(), outside: t.TempDir()}
	world.nested = filepath.Join(world.project, "a", "b")
	if err := os.MkdirAll(world.nested, 0o750); err != nil {
		t.Fatal(err)
	}
	world.globalDir = filepath.Join(t.TempDir(), "codex-home")
	if setup.globalInHome {
		world.globalDir = filepath.Join(world.home, RootDir)
	}
	userSkills := filepath.Join(world.home, SkillsDir)
	installStubbed(t, world.globalDir, Location{GlobalDir: world.globalDir, UserSkills: userSkills})
	if setup.withProject {
		installStubbed(t, filepath.Join(world.project, RootDir), Location{})
	}
	if setup.withoutProjectBin {
		removeFile(t, filepath.Join(world.project, RootDir, "bin", "pulsarules_cli"))
	}
	if setup.withoutGlobalScript {
		removeFile(t, filepath.Join(world.globalDir, "hooks", "skill-router-reminder.sh"))
	}
	return world
}

func removeFile(tb testing.TB, path string) {
	tb.Helper()
	if err := os.Remove(path); err != nil {
		tb.Fatal(err)
	}
}

type commandCase struct {
	name       string
	global     bool
	setup      worldSetup
	cwd        func(commandWorld) string
	wantRecord func(commandWorld) string
}

func commandCases() []commandCase {
	return []commandCase{
		{
			name:  "project command runs the project script from a nested directory",
			setup: worldSetup{withProject: true},
			cwd:   func(world commandWorld) string { return world.nested },
			wantRecord: func(world commandWorld) string {
				return "hook\npost-edit\n" + world.project + "\n" + world.project +
					"/.agents/skills:/home/tester/.agents/skills\n" + world.project +
					"/.codex/hook-execution.log\ncodex\npayload\n"
			},
		},
		{
			name:       "project command is silent outside a checkout",
			setup:      worldSetup{withProject: true},
			cwd:        func(world commandWorld) string { return world.outside },
			wantRecord: func(commandWorld) string { return "" },
		},
		{
			name: "global command yields to the project install", global: true,
			setup:      worldSetup{withProject: true},
			cwd:        func(world commandWorld) string { return world.nested },
			wantRecord: func(commandWorld) string { return "" },
		},
		{
			name: "global command runs when the project binary is missing", global: true,
			setup: worldSetup{withProject: true, withoutProjectBin: true},
			cwd:   func(world commandWorld) string { return world.nested },
			wantRecord: func(world commandWorld) string {
				return "hook\npost-edit\n" + world.nested + "\n" + world.nested +
					"/.agents/skills:" + world.home + "/.agents/skills\n" + world.globalDir +
					"/hook-execution.log\ncodex\npayload\n"
			},
		},
		{
			name: "global command is silent when its own script is gone", global: true,
			setup:      worldSetup{withoutGlobalScript: true},
			cwd:        func(world commandWorld) string { return world.nested },
			wantRecord: func(commandWorld) string { return "" },
		},
		{
			name: "global command runs when the project has no install", global: true,
			cwd: func(world commandWorld) string { return world.nested },
			wantRecord: func(world commandWorld) string {
				return "hook\npost-edit\n" + world.nested + "\n" + world.nested +
					"/.agents/skills:" + world.home + "/.agents/skills\n" + world.globalDir +
					"/hook-execution.log\ncodex\npayload\n"
			},
		},
		{
			name: "global command does not yield to itself", global: true,
			setup: worldSetup{globalInHome: true},
			cwd:   func(world commandWorld) string { return world.home },
			wantRecord: func(world commandWorld) string {
				return "hook\npost-edit\n" + world.home + "\n" + world.home +
					"/.agents/skills:" + world.home + "/.agents/skills\n" + world.globalDir +
					"/hook-execution.log\ncodex\npayload\n"
			},
		},
	}
}

//nolint:paralleltest // an exec racing a concurrent stub write fails with ETXTBSY.
func TestCommand_Execution(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}
	testCases := commandCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			world := newCommandWorld(t, testCase.setup)
			loc := Location{}
			if testCase.global {
				loc = Location{GlobalDir: world.globalDir}
			}
			record := filepath.Join(t.TempDir(), "record")
			runShell(t, []string{"sh", "-c", loc.command("post-edit")}, testCase.cwd(world), record)
			if got, want := readRecord(t, record), testCase.wantRecord(world); got != want {
				t.Errorf("record = %q, want %q", got, want)
			}
		})
	}
}

//nolint:paralleltest // an exec racing a concurrent stub write fails with ETXTBSY.
func TestCommand_UnusablePWDEndsQuietly(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}
	world := newCommandWorld(t, worldSetup{withProject: true})
	for _, global := range []bool{false, true} {
		for _, pwd := range []string{"", ".", "a/b"} {
			t.Run(fmt.Sprintf("global %v pwd %q", global, pwd), func(t *testing.T) {
				loc := Location{}
				if global {
					loc = Location{GlobalDir: world.globalDir}
				}
				record := filepath.Join(t.TempDir(), "record")
				script := "PWD='" + pwd + "'; " + loc.command("post-edit")
				runShell(t, []string{"sh", "-c", script}, world.nested, record)
				if got := readRecord(t, record); got != "" {
					t.Errorf("record = %q, want no hook run", got)
				}
			})
		}
	}
}
