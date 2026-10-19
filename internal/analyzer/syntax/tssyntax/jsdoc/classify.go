package jsdoc

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/docshape"
)

func IsBlock(comment string) bool {
	return strings.HasPrefix(comment, "/**") && !strings.HasPrefix(comment, "/**/")
}

func Classify(doc string, others []string) docshape.Shape {
	own := strings.Split(doc, "\n")
	shape := docshape.Classify(own, others)
	shape.Exempt = shape.Exempt || isTagsOnly(docshape.Prose(own))
	return shape
}
