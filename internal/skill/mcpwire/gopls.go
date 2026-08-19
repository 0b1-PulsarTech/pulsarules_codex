package mcpwire

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
)

const goplsBinary = "gopls"

func GoplsOnPath() bool {
	_, err := exec.LookPath(goplsBinary)
	return err == nil
}

func GoplsInstructions(ctx context.Context) (string, error) {
	result, err := execx.Run(
		ctx,
		execx.Command{Name: goplsBinary, Args: []string{"mcp", "-instructions"}},
	)
	if err != nil {
		return "", fmt.Errorf("run gopls mcp -instructions: %w", err)
	}
	return result.Stdout, nil
}

func GenerateGoplsSkill(
	templates fs.FS,
	skillsDir, instructions string,
) (backedUp []string, err error) {
	var header []byte
	header, err = fs.ReadFile(templates, "skills/gopls-navigation.header.md")
	if err != nil {
		return nil, fmt.Errorf("read gopls-navigation header template: %w", err)
	}
	var doc strings.Builder
	doc.Write(header)
	doc.WriteString("\n## Full gopls MCP reference\n\n")
	doc.WriteString(strings.TrimSpace(instructions))
	doc.WriteString("\n")

	dir := filepath.Join(skillsDir, "gopls-navigation")
	backedUp, err = output.WriteDoc(dir, "SKILL.md", doc.String())
	if err != nil {
		return backedUp, fmt.Errorf("write gopls-navigation skill: %w", err)
	}
	return backedUp, nil
}
