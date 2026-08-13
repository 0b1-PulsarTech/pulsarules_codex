package golang

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// GoHandler implements core.Language for Go source files.
type GoHandler struct{}

var _ core.Language = GoHandler{}

func New() GoHandler { return GoHandler{} }

func (GoHandler) ID() string            { return "go" }
func (GoHandler) Extensions() []string  { return []string{".go"} }
func (GoHandler) CommentPrefix() string { return "//" }

func (GoHandler) IsCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*")
}

func (GoHandler) IsBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

func (GoHandler) IsPackageDeclaration(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "package ")
}
