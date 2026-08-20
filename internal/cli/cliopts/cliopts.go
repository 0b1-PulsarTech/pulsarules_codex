package cliopts

import (
	"fmt"
)

type Options struct {
	Command  string // generate | install | uninstall | list | validate | package | hook | commitlint | governance
	Root     string // --root: repository root for dev mode (empty = embedded)
	LogLevel string // --log-level: hook execution log level (empty disables logging)

	// generate / package share Out (--out).
	Out string

	// install
	Global bool
	// Project is install/uninstall's --project (install destination), kept
	// separate from ProjectDir (commitlint/clean/governance's repository to
	// analyze) since BaseDir needs it in this package. cli.resolveProjectDir
	// reads Project too for install/uninstall, so the flag still resolves to
	// one directory at the point that wires vcs.Repository.
	Project     string
	All         bool // --all: install every skill (explicit; there is no implicit default)
	Skills      string
	Target      []string // --target (repeatable): claude | opencode | agents | cursor (default claude)
	RouterOnly  bool
	NoHooks     bool   // --no-hooks: skip the Claude hook script + settings wiring (git hooks: --no-git-hooks)
	NoMCP       bool   // --no-mcp: skip writing .mcp.json + generating the gopls-navigation skill
	PrintHooks  bool   // --print-hooks: print the resolved hooks block and exit (no writes)
	HooksScope  string // --hooks-scope: project (settings.json) | local (settings.local.json)
	Layout      string // --layout: a customization profile id (e.g. monorepo | inner-modules)
	Interactive bool   // --interactive: prompt for unset customizations (e.g. the layout)
	GitHooks    string // --git-hooks: comma-separated git hooks to install (commit-msg,pre-commit,pre-push) (default: commit-msg,pre-commit)
	NoGitHooks  bool   // --no-git-hooks: skip git hooks installation
	// --typographic-severity: how hard a typographic-marker finding lands
	// (error|warning|info). Set at install, it is baked into the generated hook
	// scripts; passed to governance, it applies to that run.
	TypographicSeverity string
	// --branch-extra-types: comma-separated branch types a project allows on top
	// of the Conventional Commit set (e.g. release,hotfix). Set at install, it is
	// baked into the pre-push hook.
	BranchExtraTypes string

	// uninstall (also reuses --root, --project, --global, --target, --hooks-scope)
	KeepSkills bool // --keep-skills: uninstall keeps the rendered skill/workflow docs; hook wiring is still removed

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
	Mode string // session-start|pre-edit|post-edit|pre-search|user-prompt|stop|subagent-start|subagent-stop|session-end

	// commitlint
	CommitMsg  string // --msg: commit message to validate
	CommitFile string // --file: path to a commit-msg file (alternative to --msg)

	ProjectDir string // --project: project dir for git history lookup

	// governance (also reuses --project and --root)

	// clean
	Write bool // --write: rewrite files instead of only reporting

	// evals
	Scenario string // --scenario: grade only this scenario id (default: every one)
	Artifact string // --artifact: path to the produced code/transcript to grade
}

// settings file names for the two hook scopes.
const (
	settingsProject = "settings.json"
	settingsLocal   = "settings.local.json"
)

// defaultTarget is the install layout used when --target is omitted. The set of
// valid targets is owned by the target.Registry, which install.Run validates
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

// SettingsFiles resolves the settings files uninstall should unwire hook
// wiring from. With no --hooks-scope it checks both settings.json and
// settings.local.json, since uninstall has no record of which scope install
// used; it narrows to the named scope only when --hooks-scope is passed.
func (opts *Options) SettingsFiles() ([]string, error) {
	switch opts.HooksScope {
	case "":
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
