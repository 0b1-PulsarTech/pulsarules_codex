package tsdoc

import "strings"

type Shape struct{ Exempt, Mixed bool }

func IsJSDoc(comment string) bool {
	return strings.HasPrefix(comment, "/**") && !strings.HasPrefix(comment, "/**/")
}

func Classify(doc string, others []string) Shape {
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
	return Shape{
		Exempt: len(prose) == 0 || isDeprecatedOnly(prose, own) || isSimplification(prose) ||
			isTagsOnly(prose),
		Mixed: len(prose) > 0 && hasDirective,
	}
}
