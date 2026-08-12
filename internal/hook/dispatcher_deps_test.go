package hook

import (
	"testing"
)

func TestDispatcher_ProjectDirEnvFallback(t *testing.T) {
	testCases := []struct {
		name       string
		projectDir string
		envDir     string
		want       string
	}{
		{
			name: "explicit dir wins", projectDir: "/explicit",
			envDir: "/from-env", want: "/explicit",
		},
		{
			name: "falls back to the env var", projectDir: "",
			envDir: "/from-env", want: "/from-env",
		},
		{name: "empty when neither is set", projectDir: "", envDir: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_PROJECT_DIR", testCase.envDir)
			d := NewDispatcher(Deps{ProjectDir: testCase.projectDir})
			if got := d.resolveProjectDir(); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestDispatcher_SkillsDirEnvFallback(t *testing.T) {
	testCases := []struct {
		name      string
		skillsDir string
		envDir    string
		want      string
	}{
		{
			name: "explicit dir wins", skillsDir: "/explicit/skills",
			envDir: "/from-env/skills", want: "/explicit/skills",
		},
		{
			name: "falls back to the env var", skillsDir: "",
			envDir: "/from-env/skills", want: "/from-env/skills",
		},
		{name: "empty when neither is set", skillsDir: "", envDir: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_SKILLS_DIR", testCase.envDir)
			d := NewDispatcher(Deps{SkillsDir: testCase.skillsDir})
			if got := d.resolveSkillsDir(); got != testCase.want {
				t.Errorf("resolveSkillsDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}
