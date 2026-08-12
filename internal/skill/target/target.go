package target

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
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

type Report struct {
	Notes    []string
	Warnings []string
}

func (r *Report) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}
