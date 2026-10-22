package version

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"

type Options struct {
	Command  string
	Root     string
	LogLevel string
	Env      flags.Environment

	Global      bool
	Project     string
	All         bool
	Skills      []string
	Target      []string
	RouterOnly  bool
	NoHooks     bool
	NoMCP       bool
	PrintHooks  bool
	HooksScope  string
	Layout      string
	Interactive bool
	GitHooks    []string
	NoGitHooks  bool
	Languages   []string
	Tuning      flags.Tuning

	KeepSkills bool

	Mode string

	ProjectDir string
}
