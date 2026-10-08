package jsdoc

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/docshape"
)

func IsBlock(comment string) bool {
	return strings.HasPrefix(comment, "/**") && !strings.HasPrefix(comment, "/**/")
}

func Classify(doc string, others []string) docshape.Shape {
	own := docLines(doc)
	var prose []string
	hasDirective := false
	for _, line := range own {
		if isDirective(line) {
			hasDirective = true
		} else if line != "" {
			prose = append(prose, line)
		}
	}
	for _, other := range others {
		hasDirective = hasDirective || isDirective(commentBody(other))
	}
	return docshape.Shape{
		Exempt: len(prose) == 0 || isDeprecatedOnly(prose, own) || isSimplification(prose) ||
			isTagsOnly(prose),
		Mixed: len(prose) > 0 && hasDirective,
	}
}
