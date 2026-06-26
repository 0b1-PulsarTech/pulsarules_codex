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

// Target installs the rendered skills for one agent layout into Context.Base.
type Target interface {
	// Name is the layout key the Registry registers the Strategy under.
	Name() string
	// Install renders the selected skills and wires the layout's config,
	// returning a Report of what it did.
	Install(ctx Context) (Report, error)
}

// Report collects a Strategy's human-facing output so the caller owns all
// stdout/stderr; the package itself stays silent and easy to test.
type Report struct {
	Notes    []string // progress lines for stdout
	Warnings []string // non-fatal warnings for stderr
}

// note appends a formatted progress line destined for stdout.
func (r *Report) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// warn appends a formatted non-fatal warning destined for stderr.
func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}
