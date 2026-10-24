package codex

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/dialect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const (
	RootDir     = ".codex"
	SkillsDir   = ".agents/skills"
	HooksFile   = "hooks.json"
	LogFileName = "hook-execution.log"
)

type Location struct {
	GlobalDir  string
	UserSkills string
}

func (loc Location) isGlobal() bool { return loc.GlobalDir != "" }

func (loc Location) command(mode string) string {
	const (
		scriptRel  = RootDir + "/" + hookset.HooksSubdir + "/" + hookset.ScriptName
		binRel     = RootDir + "/" + hookset.BinSubdir + "/" + selfbin.Name
		dialectEnv = "PULSARULES_HOOK_DIALECT=" + dialect.Codex
		// An empty or relative $PWD would make the walk spin on dirname, so it ends the hook.
		// The parent-walk needs no git and ends silently when no checkout carries the script.
		walkToScript = `d=$PWD; case $d in /*) ;; *) exit 0;; esac; ` +
			`while [ "$d" != / ] && [ ! -f "$d/` + scriptRel + `" ]; do d=$(dirname "$d"); done; `
		foundScript = `"$d/` + scriptRel + `"`
		foundBin    = `"$d/` + binRel + `"`
	)
	if !loc.isGlobal() {
		return walkToScript + `[ -f ` + foundScript + ` ] || exit 0; ` +
			dialectEnv + ` bash ` + foundScript + ` ` + mode
	}
	script := hookset.ShellQuote(path.Join(
		filepath.ToSlash(loc.GlobalDir), hookset.HooksSubdir, hookset.ScriptName,
	))
	// Codex runs the project hooks.json too, so a project install handles its own session.
	// Whether Codex trusts that project's hooks cannot be told from the shell, so the global
	// hook defers only to a project hook that can run: its script and its binary both exist.
	return walkToScript + `if [ -f ` + foundScript + ` ] && [ -x ` + foundBin + ` ] && [ ! ` +
		foundScript + ` -ef ` + script + ` ]; then exit 0; fi; [ -f ` + script + ` ] || exit 0; ` +
		dialectEnv + ` bash ` + script + ` ` + mode
}

func (loc Location) scriptValues(listSeparator rune) hookset.ScriptValues {
	// The script locates its install from its own path, never by walking.
	const (
		ownCodexDir = `$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)`
		ownRoot     = `$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)`
	)
	if !loc.isGlobal() {
		return hookset.ScriptValues{
			Bin: `"` + ownCodexDir + `/` + hookset.BinSubdir + `/` + selfbin.Name + `"`,
			SkillsDirs: hookset.JoinShellList(
				listSeparator, `"`+ownRoot+`/`+SkillsDir+`"`, `"$HOME/`+SkillsDir+`"`,
			),
			Log:         `"` + ownCodexDir + `/` + LogFileName + `"`,
			ProjectRoot: `"` + ownRoot + `"`,
		}
	}
	dir := filepath.ToSlash(loc.GlobalDir)
	userSkills := hookset.ShellQuote(filepath.ToSlash(loc.UserSkills))
	return hookset.ScriptValues{
		Bin: hookset.ShellQuote(path.Join(dir, hookset.BinSubdir, selfbin.Name)),
		SkillsDirs: hookset.JoinShellList(
			listSeparator, `"$PWD/`+SkillsDir+`"`, userSkills,
		),
		Log:         hookset.ShellQuote(path.Join(dir, LogFileName)),
		ProjectRoot: `"$PWD"`,
	}
}

func (loc Location) block() hookset.Block {
	hooksBlock := hookset.BuildBlock(loc.command, matcher)
	for event, groups := range hooksBlock.Hooks {
		seconds := timeoutSeconds(event)
		for _, group := range groups {
			for idx := range group.Hooks {
				if seconds != "" {
					group.Hooks[idx].Timeout = []byte(seconds)
				}
			}
		}
	}
	return hooksBlock
}

func timeoutSeconds(event string) string {
	switch event {
	case "SessionEnd":
		// Codex gives it one second by default and allows three at most.
		return "3"
	case "PreToolUse", "PostToolUse", "UserPromptSubmit":
		// Codex defaults to 600 seconds; the reminder answers at once, so only a stuck binary hits this.
		return "10"
	default:
		return ""
	}
}

func matcher(event hook.Event) string {
	const editMatcher, searchMatcher = "^(apply_patch|Edit|Write|MultiEdit)$", "Bash"
	switch {
	case event.Matcher == "":
		return ""
	case event.Mode == "pre-search":
		return searchMatcher
	default:
		return editMatcher
	}
}
