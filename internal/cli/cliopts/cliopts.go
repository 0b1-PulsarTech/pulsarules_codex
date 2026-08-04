package cliopts

import (
	"fmt"
)

type Options struct {
	Command  string // generate | install | list | validate | package
	Root     string // --root: repository root for dev mode (empty = embedded)
	LogLevel string // --log-level: hook execution log level (empty disables logging)

	// generate / package share Out (--out).
	Out string

	// install
	Global      bool
	Project     string
	All         bool // --all: install every skill (explicit; there is no implicit default)
	Skills      string
	Target      []string // --target (repeatable): claude | opencode (default claude)
	RouterOnly  bool
	NoHooks     bool   // --no-hooks: install skills only, skip the hook + settings wiring
	NoMCP       bool   // --no-mcp: skip writing .mcp.json + generating the gopls-navigation skill
	PrintHooks  bool   // --print-hooks: print the resolved hooks block and exit (no writes)
	HooksScope  string // --hooks-scope: project (settings.json) | local (settings.local.json)
	Layout      string // --layout: a customization profile id (e.g. monorepo | inner-modules)
	Interactive bool   // --interactive: prompt for unset customizations (e.g. the layout)
	GitHooks    string // --git-hooks: comma-separated git hooks to install (commit-msg,pre-commit,pre-push) (default: commit-msg,pre-commit)
	NoGitHooks  bool   // --no-git-hooks: skip git hooks installation

	// governance
	Preset         string // --preset: analyzer preset (recommended|strict|minimal)
	Scope          string // --scope: analysis scope (full|commit) (default: full)
	GolangciConfig string // --golangci-config: path to .golangci.yml
	AllFiles       bool   // --all-files: analyze every source file in the tree, not just changed ones
	// --include-generated: report findings in generated files (suppressed by default)
	IncludeGenerated bool

	// list
	Kind string

	// hook
	Mode string // session-start|pre-edit|post-edit|pre-search|user-prompt|stop|subagent-stop|session-end

	// commitlint
	CommitMsg  string // --msg: commit message to validate
	CommitFile string // --file: path to a commit-msg file (alternative to --msg)
	ProjectDir string // --project: project dir for git history lookup

	// governance (also reuses --project and --root)
}

// settings file names for the two hook scopes.
const (
	settingsProject = "settings.json"
	settingsLocal   = "settings.local.json"
)

// defaultTarget is the install layout used when --target is omitted. The set of
// valid targets is owned by the target.Registry, which runInstall validates
// against, so this default and the allow-list cannot drift.
const defaultTarget = "claude"

// SettingsFileName resolves the settings file the hook is wired into from the
// chosen scope. It defaults to the shared project scope (settings.json).
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

// ParseArgs binds the CLI flags for the chosen subcommand onto opts and applies
// defaults. A parse failure wraps flag.ErrHelp, which main detects with
// errors.Is so it can print usage cleanly instead of an error.
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
