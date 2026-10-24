package flags

import "os"

type Environment struct {
	Home       string
	ConfigDir  string
	ProjectDir string
	SkillsDir  string
	LogPath    string
	LogLevel   string
	// UserConfigDir is empty when the platform has none, which means no user layer.
	UserConfigDir string
}

func ReadEnvironment() Environment {
	// An unresolvable home stays empty; a global scope reports it where it is needed.
	home, _ := os.UserHomeDir()
	// An unresolvable directory is not an error: the user layer is simply absent.
	userConfigDir, _ := os.UserConfigDir()
	return Environment{
		Home:          home,
		ConfigDir:     os.Getenv("CLAUDE_CONFIG_DIR"),
		ProjectDir:    os.Getenv("PULSARULES_PROJECT_DIR"),
		SkillsDir:     os.Getenv("PULSARULES_SKILLS_DIR"),
		LogPath:       os.Getenv("PULSARULES_LOG_PATH"),
		LogLevel:      os.Getenv("PULSARULES_LOG_LEVEL"),
		UserConfigDir: userConfigDir,
	}
}
