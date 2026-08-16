package cliopts

import (
	"fmt"
)

type Options struct {
	Command  string
	Root     string
	LogLevel string

	Out string

	Global      bool
	Project     string
	All         bool
	Skills      string
	Target      []string
	RouterOnly  bool
	NoHooks     bool
	NoMCP       bool
	PrintHooks  bool
	HooksScope  string
	Layout      string
	Interactive bool
	GitHooks    string
	NoGitHooks  bool

	KeepSkills bool

	Preset           string
	Scope            string
	GolangciConfig   string
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

const (
	settingsProject = "settings.json"
	settingsLocal   = "settings.local.json"
)

func (opts *Options) SettingsFileName() (string, error) {
	switch opts.HooksScope {
	case "", "project":
		return settingsProject, nil
	case "local":
		return settingsLocal, nil
	default:
		return "", fmt.Errorf("invalid --hooks-scope %q (want project|local)", opts.HooksScope)
	}
}

func (opts *Options) SettingsFiles() ([]string, error) {
	switch opts.HooksScope {
	case "":
		// Uninstall has no record of which scope install used, so check both.
		return []string{settingsProject, settingsLocal}, nil
	case "project":
		return []string{settingsProject}, nil
	case "local":
		return []string{settingsLocal}, nil
	default:
		return nil, fmt.Errorf("invalid --hooks-scope %q (want project|local)", opts.HooksScope)
	}
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
