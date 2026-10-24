package flags

import (
	"os"
	"runtime"
	"testing"
)

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "/project")
	t.Setenv("PULSARULES_SKILLS_DIR", "/skills")
	t.Setenv("PULSARULES_LOG_PATH", "/log/path")
	t.Setenv("PULSARULES_LOG_LEVEL", "debug")
	t.Setenv("HOME", "/home/user")
	t.Setenv("CLAUDE_CONFIG_DIR", "/claude/config")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")

	wantUserDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	want := Environment{
		Home:          "/home/user",
		ConfigDir:     "/claude/config",
		ProjectDir:    "/project",
		SkillsDir:     "/skills",
		LogPath:       "/log/path",
		LogLevel:      "debug",
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
		"PULSARULES_LOG_PATH", "PULSARULES_LOG_LEVEL", "HOME", "CLAUDE_CONFIG_DIR",
	} {
		t.Setenv(name, "")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := ReadEnvironment(); got != (Environment{}) {
		t.Errorf("ReadEnvironment() = %+v, want the zero value", got)
	}
}
