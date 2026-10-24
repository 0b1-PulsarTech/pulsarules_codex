package flags

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "/project")
	t.Setenv("PULSARULES_SKILLS_DIR", "/skills")
	t.Setenv("PULSARULES_LOG_PATH", "/log/path")
	t.Setenv("PULSARULES_LOG_LEVEL", "debug")
	t.Setenv("PULSARULES_HOOK_DIALECT", "codex")
	t.Setenv("HOME", "/home/user")
	t.Setenv("CLAUDE_CONFIG_DIR", "/claude/config")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv("CODEX_HOME", "/codex/home")

	wantUserDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	want := Environment{
		Home:          "/home/user",
		ConfigDir:     "/claude/config",
		CodexHome:     "/codex/home",
		ProjectDir:    "/project",
		SkillsDir:     "/skills",
		LogPath:       "/log/path",
		LogLevel:      "debug",
		HookDialect:   "codex",
		UserConfigDir: wantUserDir,
	}
	if got := ReadEnvironment(); got != want {
		t.Errorf("ReadEnvironment() = %+v, want %+v", got, want)
	}
}

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment_Unset(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only Linux resolves the user config dir from XDG_CONFIG_HOME and HOME alone")
	}
	for _, name := range []string{
		"PULSARULES_PROJECT_DIR", "PULSARULES_SKILLS_DIR",
		"PULSARULES_LOG_PATH", "PULSARULES_LOG_LEVEL", "PULSARULES_HOOK_DIALECT", "HOME", "CLAUDE_CONFIG_DIR",
		"CODEX_HOME",
	} {
		t.Setenv(name, "")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := ReadEnvironment(); got != (Environment{}) {
		t.Errorf("ReadEnvironment() = %+v, want the zero value", got)
	}
}

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment_HookDialect(t *testing.T) {
	testCases := []struct {
		name        string
		raw         string
		want        string
		wantNotices int
	}{
		{"unset", "", "", 0},
		{"codex", "codex", "codex", 0},
		{"upper case and padded", "  CoDeX\n", "codex", 0},
		{"claude", "claude", "claude", 0},
		{"unknown falls back to claude with one notice", "gemini", "claude", 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_HOOK_DIALECT", testCase.raw)
			notice := captureStderr(t, func() {
				if got := ReadEnvironment().HookDialect; got != testCase.want {
					t.Errorf("HookDialect = %q, want %q", got, testCase.want)
				}
			})
			if got := strings.Count(notice, "\n"); got != testCase.wantNotices {
				t.Errorf("stderr = %q, want %d notice lines", notice, testCase.wantNotices)
			}
		})
	}
}

func captureStderr(t *testing.T, run func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatalf("create stderr capture: %v", err)
	}
	original := os.Stderr
	os.Stderr = file
	run()
	os.Stderr = original
	captured, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("read stderr capture: %v", err)
	}
	return string(captured)
}
