package docshape

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Shape struct {
	Exempt, Mixed bool
	Prose         []string
}

func Classify(dialect lexer.Dialect, own, adjacent, text []string) Shape {
	hasDirective := slices.ContainsFunc(adjacent, func(line string) bool {
		return lexer.IsDirectiveLine(dialect, line)
	})
	scan := lexer.NewCommentScanner(dialect.Syntax())
	var prose, bodies []string
	for _, line := range own {
		body, inBlock := scan.Body(line)
		bodies = append(bodies, body)
		directive := lexer.IsDirectiveLine(dialect, line)
		if inBlock {
			directive = lexer.IsDirectiveBody(dialect, body)
		}
		if directive {
			hasDirective = true
			continue
		}
		if body != "" {
			prose = append(prose, body)
		}
	}
	if len(prose) == 0 {
		return Shape{Exempt: true}
	}
	if dialect.ToolchainText() {
		bodies = text
	}
	return Shape{
		Prose: prose,
		Exempt: isDeprecatedOnly(dialect, prose[0], trimBlankEdges(bodies)) ||
			strings.HasPrefix(prose[0], lexer.SimplificationMarker),
		Mixed: hasDirective,
	}
}

func trimBlankEdges(bodies []string) []string {
	for len(bodies) > 0 && bodies[0] == "" {
		bodies = bodies[1:]
	}
	for len(bodies) > 0 && bodies[len(bodies)-1] == "" {
		bodies = bodies[:len(bodies)-1]
	}
	return bodies
}

func isDeprecatedOnly(dialect lexer.Dialect, first string, bodies []string) bool {
	const maxDeprecatedLines = 3

	if dialect.ToolchainText() {
		first = ""
		if len(bodies) > 0 {
			first = bodies[0]
		}
	}
	return dialect.IsDeprecatedMarker(first) && len(bodies) <= maxDeprecatedLines &&
		!slices.Contains(bodies, "")
}
