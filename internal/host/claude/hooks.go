package claude

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const (
	RootDir      = ".claude"
	SkillsSubdir = "skills"
	binSubdir    = "bin"
	binaryName   = "pulsarules_cli"
)

const reminderScriptAsset = "skill-router-reminder.sh"

type hookAsset struct {
	templateName string
	destName     string
	mode         os.FileMode
	render       bool
}

func hookAssets() []hookAsset {
	return []hookAsset{
		{
			templateName: reminderScriptAsset + ".tmpl",
			destName:     reminderScriptAsset,
			mode:         disk.FileExec,
			render:       true,
		},
		{templateName: "README.md", destName: "README.md", mode: disk.File},
	}
}

func InstallHook(templates fs.FS, claudeDir string) (backedUp []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err = os.MkdirAll(hooksDir, disk.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets() {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.templateName)
		if readErr != nil {
			return backedUp, fmt.Errorf("read template hooks/%s: %w", asset.templateName, readErr)
		}
		if asset.render {
			var renderErr error
			assetBytes, renderErr = renderReminderScript(asset.templateName, assetBytes)
			if renderErr != nil {
				return backedUp, renderErr
			}
		}
		dst := filepath.Join(hooksDir, asset.destName)
		note, installErr := ownedfile.InstallFile(dst, assetBytes, asset.mode)
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

func installBinary(claudeDir string) error {
	dst := filepath.Join(claudeDir, binSubdir, binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

type reminderScriptValues struct {
	BinaryRelPath string
	SkillsRelPath string
	LogRelPath    string
}

func renderReminderScript(name string, body []byte) ([]byte, error) {
	tmpl, err := template.New(name).Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse hooks/%s: %w", name, err)
	}
	const logFileName = "hook-execution.log"
	// path.Join, not filepath.Join: these paths are embedded in a bash script.
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
