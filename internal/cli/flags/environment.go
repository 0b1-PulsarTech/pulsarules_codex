package flags

import "os"

type Environment struct {
	ProjectDir string
	SkillsDir  string
	LogPath    string
	LogLevel   string
}

func ReadEnvironment() Environment {
	return Environment{
		ProjectDir: os.Getenv("PULSARULES_PROJECT_DIR"),
		SkillsDir:  os.Getenv("PULSARULES_SKILLS_DIR"),
		LogPath:    os.Getenv("PULSARULES_LOG_PATH"),
		LogLevel:   os.Getenv("PULSARULES_LOG_LEVEL"),
	}
}
