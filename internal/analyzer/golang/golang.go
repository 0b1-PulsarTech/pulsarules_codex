package golang

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type GoHandler struct{}

var _ pass.Language = GoHandler{}

func New() GoHandler { return GoHandler{} }

func (GoHandler) ID() string           { return "go" }
func (GoHandler) Extensions() []string { return []string{".go"} }

func (GoHandler) IsCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*")
}

func (GoHandler) IsPackageDeclaration(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "package ")
}
