package claude

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const (
	RootDir      = ".claude"
	SkillsSubdir = "skills"
	projectDir   = `"$CLAUDE_PROJECT_DIR/` + RootDir
)

type Location struct {
	// GlobalDir is empty for a project install, whose files are found through $CLAUDE_PROJECT_DIR.
	GlobalDir string
}

func (loc Location) isGlobal() bool { return loc.GlobalDir != "" }

func (loc Location) scriptPath() string {
	if loc.isGlobal() {
		dir := filepath.ToSlash(loc.GlobalDir)
		return shellQuote(path.Join(dir, hookset.HooksSubdir, hookset.ScriptName))
	}
	return projectDir + `/` + hookset.HooksSubdir + `/` + hookset.ScriptName + `"`
}

func (loc Location) command(mode string) string {
	return "bash " + loc.scriptPath() + " " + mode
}

func (loc Location) scriptValues(listSeparator rune) hookset.ScriptValues {
	const logFileName = "hook-execution.log"
	if !loc.isGlobal() {
		return hookset.ScriptValues{
			Bin: projectDir + `/` + hookset.BinSubdir + `/` + selfbin.Name + `"`,
			// The user directory follows Claude's own CLAUDE_CONFIG_DIR; a project script has no
			// other way to know where a global install put its skills.
			SkillsDirs: joinShellList(
				listSeparator,
				projectDir+`/`+SkillsSubdir+`"`,
				`"${CLAUDE_CONFIG_DIR:-$HOME/`+RootDir+`}/`+SkillsSubdir+`"`,
			),
			Log:         projectDir + `/` + logFileName + `"`,
			ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
		}
	}
	dir := filepath.ToSlash(loc.GlobalDir)
	return hookset.ScriptValues{
		Bin: shellQuote(path.Join(dir, hookset.BinSubdir, selfbin.Name)),
		SkillsDirs: joinShellList(
			listSeparator,
			projectDir+`/`+SkillsSubdir+`"`,
			shellQuote(path.Join(dir, SkillsSubdir)),
		),
		Log:         shellQuote(path.Join(dir, logFileName)),
		DeferTo:     projectDir + `/` + hookset.HooksSubdir + `/` + hookset.ScriptName + `"`,
		ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
	}
}

func (loc Location) block() hookset.Block {
	return hookset.BuildBlock(loc.command, func(event hook.Event) string { return event.Matcher })
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func joinShellList(separator rune, first, second string) string {
	// The separator is quoted wherever the shell would read it as syntax (";" ends a command).
	word := string(separator)
	if separator != ':' {
		word = `"` + word + `"`
	}
	return first + word + second
}
