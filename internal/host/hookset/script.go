package hookset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const ScriptName = "skill-router-reminder.sh"

type ScriptValues struct {
	Bin         string
	SkillsDirs  string
	Log         string
	DeferTo     string
	ProjectRoot string
}

type hookAsset struct {
	templateName string
	destName     string
	mode         os.FileMode
	render       bool
}

func hookAssets() []hookAsset {
	return []hookAsset{
		{
			templateName: ScriptName + ".tmpl",
			destName:     ScriptName,
			mode:         disk.ModeFileExec,
			render:       true,
		},
		{templateName: "README.md", destName: "README.md", mode: disk.ModeFile},
	}
}

func Install(
	templates fs.FS,
	dir string,
	values ScriptValues,
) (ledger ownedfile.Ledger, err error) {
	hooksDir := filepath.Join(dir, HooksSubdir)
	if err = os.MkdirAll(hooksDir, disk.ModeDirPrivate); err != nil {
		return ledger, fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets() {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.templateName)
		if readErr != nil {
			return ledger, fmt.Errorf("read template hooks/%s: %w", asset.templateName, readErr)
		}
		if asset.render {
			var renderErr error
			assetBytes, renderErr = renderScript(asset.templateName, assetBytes, values)
			if renderErr != nil {
				return ledger, renderErr
			}
		}
		dst := filepath.Join(hooksDir, asset.destName)
		note, installErr := ownedfile.InstallFile(dst, assetBytes, asset.mode)
		if installErr != nil {
			return ledger, fmt.Errorf("install %q: %w", dst, installErr)
		}
		ledger.AddBackup(note)
	}
	if err = selfbin.Install(filepath.Join(dir, BinSubdir)); err != nil {
		return ledger, err
	}
	return ledger, nil
}

func renderScript(name string, body []byte, values ScriptValues) ([]byte, error) {
	tmpl, err := template.New(name).Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse hooks/%s: %w", name, err)
	}
	var buf strings.Builder
	if execErr := tmpl.Execute(&buf, values); execErr != nil {
		return nil, fmt.Errorf("render hooks/%s: %w", name, execErr)
	}
	return []byte(buf.String()), nil
}
