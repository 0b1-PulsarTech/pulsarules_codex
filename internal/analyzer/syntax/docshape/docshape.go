package docshape

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Shape struct{ Exempt, Mixed bool }

func Classify(own, adjacent []string) Shape {
	bodies := trimBlankEdges(bodiesOf(own))
	prose := Prose(own)
	hasDirective := slices.ContainsFunc(own, lexer.IsDirectiveLine) ||
		slices.ContainsFunc(adjacent, lexer.IsDirectiveLine)
	return Shape{
		Exempt: len(prose) == 0 || isDeprecatedOnly(prose, bodies) ||
			strings.HasPrefix(prose[0], lexer.SimplificationMarker),
		Mixed: len(prose) > 0 && hasDirective,
	}
}

func Prose(own []string) []string {
	var prose []string
	for _, line := range own {
		if body := lexer.CommentBody(line, lexer.AnyCommentSyntax()); body != "" &&
			!lexer.IsDirectiveLine(line) {
			prose = append(prose, body)
		}
	}
	return prose
}

func bodiesOf(own []string) []string {
	bodies := make([]string, len(own))
	for idx, line := range own {
		bodies[idx] = lexer.CommentBody(line, lexer.AnyCommentSyntax())
	}
	return bodies
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

func isDeprecatedOnly(prose, bodies []string) bool {
	const maxDeprecatedLines = 3

	marked := strings.HasPrefix(prose[0], "@deprecated") ||
		strings.HasPrefix(prose[0], "Deprecated:")
	return marked && len(bodies) <= maxDeprecatedLines && !slices.Contains(bodies, "")
}
