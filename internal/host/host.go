package host

import (
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Context struct {
	Templates      fs.FS
	Index          *knowledge.Index
	Renderer       *render.Renderer
	HookInstallers *install.Registry
	Base           string
	IDs            []string
	RouterFilter   []string
	NoMCP          bool
	NoHooks        bool
	SettingsFile   string
	Languages      detect.Set
}

type UninstallContext struct {
	Base             string
	HookUninstallers *install.Registry
	SettingsFiles    []string
	KeepSkills       bool
}

type Target interface {
	Name() string
	Present(base string) bool
	Install(ctx Context) (Report, error)
	Uninstall(ctx UninstallContext) (Report, error)
}

type Report = report.Report
