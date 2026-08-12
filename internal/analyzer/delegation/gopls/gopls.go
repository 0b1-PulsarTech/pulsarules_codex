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

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Available() bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// Run shells out to "gopls version" for a lightweight availability check.
//
// simplification: "gopls version" is the only portable CLI query; full
// diagnostics need MCP or LSP. Upgrade path: connect the gopls MCP server
// (internal/skill/mcpwire) and route its diagnostics through the pipeline.
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
