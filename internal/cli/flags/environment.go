package flags

import (
	"fmt"
	"os"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/dialect"
)

type Environment struct {
	Home        string
	ConfigDir   string
	ProjectDir  string
	SkillsDir   string
	LogPath     string
	LogLevel    string
	HookDialect string
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
		HookDialect:   readHookDialect(),
		UserConfigDir: userConfigDir,
	}
}

func readHookDialect() string {
	raw := os.Getenv("PULSARULES_HOOK_DIALECT")
	name := strings.ToLower(strings.TrimSpace(raw))
	switch name {
	case "", dialect.Claude, dialect.Codex:
		return name
	default:
		_, _ = fmt.Fprintln(
			os.Stderr,
			"pulsarules_cli: unknown PULSARULES_HOOK_DIALECT, using claude",
		)
		return dialect.Claude
	}
}
