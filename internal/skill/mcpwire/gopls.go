package mcpwire

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
)

// goplsBinary is the gopls executable name resolved on PATH.
const goplsBinary = "gopls"

// GoplsOnPath reports whether the gopls binary is resolvable on PATH.
func GoplsOnPath() bool {
	_, err := exec.LookPath(goplsBinary)
	return err == nil
}

// GoplsInstructions runs `gopls mcp -instructions` and returns its output: the
// authoritative, version-current MCP tool reference for the installed gopls.
func GoplsInstructions(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, goplsBinary, "mcp", "-instructions").Output()
	if err != nil {
		return "", fmt.Errorf("run gopls mcp -instructions: %w", err)
	}
	return string(out), nil
}

// GenerateGoplsSkill writes <skillsDir>/gopls-navigation/SKILL.md by combining the
// curated header template with the live `gopls mcp -instructions` output, so the
// skill is always current with the installed gopls. The caller skips this when
// gopls is absent.
func GenerateGoplsSkill(templates fs.FS, skillsDir, instructions string) error {
	header, err := fs.ReadFile(templates, "skills/gopls-navigation.header.md")
	if err != nil {
		return fmt.Errorf("read gopls-navigation header template: %w", err)
	}
	var doc strings.Builder
	doc.Write(header)
	doc.WriteString("\n## Full gopls MCP reference\n\n")
	doc.WriteString(strings.TrimSpace(instructions))
	doc.WriteString("\n")

	dir := filepath.Join(skillsDir, "gopls-navigation")
	if err = output.WriteDoc(dir, "SKILL.md", doc.String()); err != nil {
		return fmt.Errorf("write gopls-navigation skill: %w", err)
	}
	return nil
}
