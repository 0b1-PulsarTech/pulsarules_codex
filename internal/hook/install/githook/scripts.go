package githook

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

type hookSpec struct {
	description string
	command     string
	governed    bool
	isFullScope bool
}

func hookSpecs() map[string]hookSpec {
	return map[string]hookSpec{
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
			isFullScope: true,
		},
	}
}

type Options struct {
	TypographicSeverity string
	BranchExtraTypes    string
}

func (opts Options) typographicFlag() string {
	if opts.TypographicSeverity == "" {
		return ""
	}
	return " --typographic-severity " + opts.TypographicSeverity
}

func (opts Options) branchFlag() string {
	if opts.BranchExtraTypes == "" {
		return ""
	}
	return " --branch-extra-types " + opts.BranchExtraTypes
}

func hookScript(name string, opts Options) (string, bool) {
	spec, ok := hookSpecs()[name]
	if !ok {
		return "", false
	}
	// Policy is baked in at install: a git hook gets no arguments from the committer.
	command := spec.command
	if spec.governed {
		command += opts.typographicFlag()
	}
	// Branch-name runs only at ScopeFull; pre-commit would carry a value no analyzer reads.
	if spec.isFullScope {
		command += opts.branchFlag()
	}
	const scriptPreamble = `PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$PROJECT_DIR" || exit 0
COMMON_DIR=$(git rev-parse --git-common-dir 2>/dev/null) || exit 0
case "$COMMON_DIR" in
[/]*) ;;
*) COMMON_DIR="$PROJECT_DIR/$COMMON_DIR" ;;
esac
BINARY="$COMMON_DIR/hooks/` + binaryName + `"
[ -x "$BINARY" ] || exit 0
`
	// The preamble resolves the installer via the shared hooks dir, so a hook fired from
	// a linked worktree finds it. --git-common-dir is relative in a normal checkout and
	// absolute in a worktree.
	return "#!/bin/sh\n" +
		"# pulsarules_codex " + name + " hook - " + spec.description + ".\n" +
		"# " + marker.Installed + "; remove or edit this file to disable.\n" +
		scriptPreamble + `exec "$BINARY" ` + command + "\n", true
}
