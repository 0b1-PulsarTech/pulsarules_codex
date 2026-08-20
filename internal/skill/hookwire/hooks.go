package hookwire

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const (
	// RootDir is the Claude Code layout's root, relative to a project's root
	// directory. claudeTarget's own paths (internal/skill/target/claude.go)
	// and the reminder script this package renders both derive from it,
	// instead of each hand-copying ".claude".
	RootDir = ".claude"
	// SkillsSubdir is where claudeTarget renders skills, relative to RootDir.
	SkillsSubdir = "skills"
	// binSubdir is where InstallHook copies the installer binary, relative
	// to RootDir.
	binSubdir = "bin"
	// binaryName is where the orchestrator hook expects the installer binary.
	binaryName = "pulsarules_cli"
	// logFileName is where the reminder script points PULSARULES_LOG_PATH,
	// relative to RootDir.
	logFileName = "hook-execution.log"
)

const reminderScriptAsset = "skill-router-reminder.sh"

// hookAssets are the files installed from templates/hooks into
// <claudeDir>/hooks. reminderScriptAsset is rendered through text/template
// (its two Claude-layout paths come from RootDir/SkillsSubdir/binSubdir
// rather than being baked into the template source); README.md is copied
// verbatim. The script is made executable; the README carries its WHY.
var hookAssets = []struct {
	templateName string
	destName     string
	mode         os.FileMode
	render       bool
}{
	{
		templateName: reminderScriptAsset + ".tmpl",
		destName:     reminderScriptAsset,
		mode:         fsperm.FileExec,
		render:       true,
	},
	{templateName: "README.md", destName: "README.md", mode: fsperm.File},
}

// InstallHook copies the hook script (executable) and its README from the
// embedded templates into <claudeDir>/hooks, then installs the binary the
// orchestrator script forwards to. An asset not written by an earlier
// InstallHook is renamed to a numbered ".pulsarules-backup" slot rather
// than destroyed; backedUp reports each rename as a printable message.
func InstallHook(templates fs.FS, claudeDir string) (backedUp []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err = os.MkdirAll(hooksDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.templateName)
		if readErr != nil {
			return backedUp, fmt.Errorf("read template hooks/%s: %w", asset.templateName, readErr)
		}
		if asset.render {
			assetBytes, readErr = renderReminderScript(asset.templateName, assetBytes)
			if readErr != nil {
				return backedUp, readErr
			}
		}
		dst := filepath.Join(hooksDir, asset.destName)
		note, installErr := marker.InstallFile(dst, assetBytes, asset.mode)
		if installErr != nil {
			return backedUp, fmt.Errorf("install %q: %w", dst, installErr)
		}
		if note != "" {
			backedUp = append(backedUp, note)
		}
	}
	if err = installBinary(claudeDir); err != nil {
		return backedUp, err
	}
	return backedUp, nil
}

// installBinary copies the running installer binary into <claudeDir>/bin so the
// orchestrator hook can invoke it. The hook script guards on the binary's
// presence, so a copy failure degrades to a no-op hook rather than a hard error.
func installBinary(claudeDir string) error {
	dst := filepath.Join(claudeDir, binSubdir, binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

// reminderScriptValues are the Claude-layout paths the reminder script needs,
// relative to $CLAUDE_PROJECT_DIR - values generic Go must never hardcode
// (see dispatcher_deps.go's resolveSkillsDir and cli.go's resolveLogPath),
// drawn from this package's own Root/Skills/bin/binaryName/logFileName
// constants instead of being baked into the template source.
type reminderScriptValues struct {
	BinaryRelPath string
	SkillsRelPath string
	LogRelPath    string
}

// renderReminderScript executes the named template against
// reminderScriptValues, the same convention renderPostEditChecklist uses
// for the post-edit checklist. It uses path.Join, not filepath.Join, since
// the result is embedded in a bash script rather than used as an OS path,
// so paths must stay forward-slashed regardless of host OS.
func renderReminderScript(name string, body []byte) ([]byte, error) {
	tmpl, err := template.New(name).Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse hooks/%s: %w", name, err)
	}
	values := reminderScriptValues{
		BinaryRelPath: path.Join(RootDir, binSubdir, binaryName),
		SkillsRelPath: path.Join(RootDir, SkillsSubdir),
		LogRelPath:    path.Join(RootDir, logFileName),
	}
	var buf strings.Builder
	if execErr := tmpl.Execute(&buf, values); execErr != nil {
		return nil, fmt.Errorf("render hooks/%s: %w", name, execErr)
	}
	return []byte(buf.String()), nil
}
