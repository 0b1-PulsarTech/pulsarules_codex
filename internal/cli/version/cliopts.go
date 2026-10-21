package version

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"

type Options struct {
	Command  string
	Root     string
	LogLevel string

	Out string

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

	Preset           string
	Scope            string
	GolangciConfig   string
	Config           string
	AllFiles         bool
	IncludeGenerated bool

	Kind string

	Mode string

	CommitMsg  string
	CommitFile string

	ProjectDir string

	Write bool

	Scenario string
	Artifact string
}

func (opts *Options) applyDefaults() {
	switch opts.Command {
	case "generate":
		if opts.Out == "" {
			opts.Out = defaultOut("generated")
		}
	case "package":
		if opts.Out == "" {
			opts.Out = defaultOut("build", "standards-skills.zip")
		}
	}
}
