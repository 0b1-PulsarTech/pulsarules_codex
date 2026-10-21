package flags

import "testing"

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "/project")
	t.Setenv("PULSARULES_SKILLS_DIR", "/skills")
	t.Setenv("PULSARULES_LOG_PATH", "/log/path")
	t.Setenv("PULSARULES_LOG_LEVEL", "debug")

	want := Environment{
		ProjectDir: "/project",
		SkillsDir:  "/skills",
		LogPath:    "/log/path",
		LogLevel:   "debug",
	}
	if got := ReadEnvironment(); got != want {
		t.Errorf("ReadEnvironment() = %+v, want %+v", got, want)
	}
}

//nolint:paralleltest // t.Setenv bars t.Parallel.
func TestReadEnvironment_Unset(t *testing.T) {
	for _, name := range []string{
		"PULSARULES_PROJECT_DIR", "PULSARULES_SKILLS_DIR",
		"PULSARULES_LOG_PATH", "PULSARULES_LOG_LEVEL",
	} {
		t.Setenv(name, "")
	}
	if got := ReadEnvironment(); got != (Environment{}) {
		t.Errorf("ReadEnvironment() = %+v, want the zero value", got)
	}
}
