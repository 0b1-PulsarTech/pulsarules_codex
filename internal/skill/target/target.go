package target

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Context carries the resolved inputs an install Strategy needs. It is a small
// facade over the CLI options so this package never imports the cmd package.
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

// UninstallContext carries the resolved inputs an uninstall Strategy needs,
// the reversal counterpart of Context. SettingsFiles carries every hook
// settings file to unwire (typically both settings.json and
// settings.local.json), since uninstall cannot recover which --hooks-scope
// install used.
type UninstallContext struct {
	Base             string
	HookUninstallers *install.Registry
	SettingsFiles    []string
	KeepSkills       bool
}

// Target installs the rendered skills for one agent layout into Context.Base.
type Target interface {
	// Name is the layout key the Registry registers the Strategy under.
	Name() string
	// Present reports whether base holds anything this layout's Install could
	// have written, so uninstall's target auto-detection can ask each layout
	// instead of the caller guessing which files belong to it.
	Present(base string) bool
	// Install renders the selected skills and wires the layout's config,
	// returning a Report of what it did.
	Install(ctx Context) (Report, error)
	// Uninstall removes what Install wrote from ctx.Base, returning a Report
	// of what it did. It is idempotent: a project Install never touched, or a
	// second run, is not an error.
	Uninstall(ctx UninstallContext) (Report, error)
}

// Report collects a Strategy's human-facing output so the caller owns all
// stdout/stderr; the package itself stays silent and easy to test.
type Report struct {
	Notes    []string // progress lines for stdout
	Warnings []string // non-fatal warnings for stderr
}

func (r *Report) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}
