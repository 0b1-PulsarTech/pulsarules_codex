package claude

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const (
	hooksSubdir = "hooks"
	projectDir  = `"$CLAUDE_PROJECT_DIR/` + RootDir
)

type Location struct {
	// GlobalDir is empty for a project install, whose files are found through $CLAUDE_PROJECT_DIR.
	GlobalDir string
}

func (loc Location) isGlobal() bool { return loc.GlobalDir != "" }

func (loc Location) scriptPath() string {
	if loc.isGlobal() {
		dir := filepath.ToSlash(loc.GlobalDir)
		return shellQuote(path.Join(dir, hooksSubdir, reminderScriptAsset))
	}
	return projectDir + `/` + hooksSubdir + `/` + reminderScriptAsset + `"`
}

func (loc Location) command(mode string) string {
	return "bash " + loc.scriptPath() + " " + mode
}

func (loc Location) scriptValues(listSeparator rune) reminderScriptValues {
	const logFileName = "hook-execution.log"
	if !loc.isGlobal() {
		return reminderScriptValues{
			Bin: projectDir + `/` + binSubdir + `/` + selfbin.Name + `"`,
			// The user directory follows Claude's own CLAUDE_CONFIG_DIR; a project script has no
			// other way to know where a global install put its skills.
			SkillsDirs: joinShellList(
				listSeparator,
				projectDir+`/`+SkillsSubdir+`"`,
				`"${CLAUDE_CONFIG_DIR:-$HOME/`+RootDir+`}/`+SkillsSubdir+`"`,
			),
			Log: projectDir + `/` + logFileName + `"`,
		}
	}
	dir := filepath.ToSlash(loc.GlobalDir)
	return reminderScriptValues{
		Bin: shellQuote(path.Join(dir, binSubdir, selfbin.Name)),
		SkillsDirs: joinShellList(
			listSeparator,
			projectDir+`/`+SkillsSubdir+`"`,
			shellQuote(path.Join(dir, SkillsSubdir)),
		),
		Log:     shellQuote(path.Join(dir, logFileName)),
		DeferTo: projectDir + `/` + hooksSubdir + `/` + reminderScriptAsset + `"`,
	}
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
