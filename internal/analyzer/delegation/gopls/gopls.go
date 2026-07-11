package gopls

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const binary = "gopls"

var goplsReporter = core.NewReporter("gopls", core.SeverityInfo, core.CatSyntax)

// Runner shells out to gopls for diagnostics.
type Runner struct{}

// NewRunner creates a Runner that delegates to gopls for diagnostics.
func NewRunner() *Runner {
	return &Runner{}
}

// Available reports whether gopls is on PATH.
func (r *Runner) Available() bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// Run shells out to "gopls version" for a lightweight availability check.
//
// simplification: "gopls version" is the only portable CLI query. Full
// file-level diagnostics require MCP or LSP; the upgrade path is connecting
// the gopls MCP server (via internal/skill/mcpwire) and routing its
// diagnostics through the pipeline.
func (r *Runner) Run() []core.Finding {
	if !r.Available() {
		return nil
	}

	out, err := exec.CommandContext(context.Background(), binary, "version").Output()
	if err != nil {
		return nil
	}
	version := strings.TrimSpace(string(out))

	return []core.Finding{goplsReporter.New(fmt.Sprintf("gopls available: %s", version))}
}
