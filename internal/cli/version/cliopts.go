package version

type Options struct {
	Command  string
	Root     string
	LogLevel string

	Out string

	Preset         string
	Scope          string
	GolangciConfig string
	AllFiles       bool

	Kind string

	Mode string

	CommitMsg  string
	CommitFile string

	ProjectDir string
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
