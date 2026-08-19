package githook

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// scriptPreamble resolves the installer binary the way Install placed it, via the
// repository's shared hooks dir, so a hook fired from a linked worktree finds it
// instead of exiting as a silent no-op. --git-common-dir is relative in a normal
// checkout and absolute in a worktree, hence the case that joins only relative.
const scriptPreamble = `PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$PROJECT_DIR" || exit 0
COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) || exit 0
case "$COMMON_DIR" in
/*) ;;
*) COMMON_DIR="$PROJECT_DIR/$COMMON_DIR" ;;
esac
BINARY="$COMMON_DIR/hooks/` + binaryName + `"
[ -x "$BINARY" ] || exit 0
`

// hookSpec describes one git hook: the WHY a reader of .git/hooks sees, the
// installer invocation it execs, and whether governance policy applies to it.
type hookSpec struct {
	description string
	command     string
	governed    bool
}

// hookSpecs is the whole supported set, keyed by git hook name.
var hookSpecs = map[string]hookSpec{
	"commit-msg": {
		description: "validates commit message format",
		command:     `commitlint --project "$PROJECT_DIR" --file "$1"`,
	},
	"pre-commit": {
		description: "runs governance checks on staged changes",
		command:     `governance --project "$PROJECT_DIR" --scope commit`,
		governed:    true,
	},
	"pre-push": {
		description: "runs governance checks before pushing",
		command:     `governance --project "$PROJECT_DIR"`,
		governed:    true,
	},
}

// Options carries the policy chosen at install time, baked into the scripts.
// why: a git hook receives no arguments from the person committing, so a flag not
// written into the script at install can never reach the gate - this is what
// makes a configured severity apply to pre-commit, not just a hand-typed run.
type Options struct {
	// TypographicSeverity, when set, spells how hard a typographic-marker
	// finding lands. Empty leaves the analyzer's own default in place.
	TypographicSeverity string
	// BranchExtraTypes, when set, lists the branch types a project allows on top
	// of the Conventional Commit set. Empty allows only the commit types.
	BranchExtraTypes string
}

// governanceFlags renders opts as flags appended to a governance invocation.
func (o Options) governanceFlags() string {
	var flags string
	if o.TypographicSeverity != "" {
		flags += " --typographic-severity " + o.TypographicSeverity
	}
	if o.BranchExtraTypes != "" {
		flags += " --branch-extra-types " + o.BranchExtraTypes
	}
	return flags
}

// hookScript renders the full shell script for one git hook, reporting false
// for a name no hook is defined for.
func hookScript(name string, opts Options) (string, bool) {
	spec, ok := hookSpecs[name]
	if !ok {
		return "", false
	}
	command := spec.command
	if spec.governed {
		command += opts.governanceFlags()
	}
	return "#!/bin/sh\n" +
		"# pulsarules_codex " + name + " hook - " + spec.description + ".\n" +
		"# " + marker.Installed + "; remove or edit this file to disable.\n" +
		scriptPreamble + `exec "$BINARY" ` + command + "\n", true
}
