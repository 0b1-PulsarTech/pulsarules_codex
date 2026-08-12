package render

import "strings"

// clauseSections are the section keys whose lines state unconditional
// directives: requirements ("must") and prohibitions ("forbidden"). Losing a
// line from either during transclusion silently weakens the contract, which is
// the failure this file exists to catch.
var clauseSections = []string{"must", "forbidden"}

// clausesFromSections returns the load-bearing lines of a source's must and
// forbidden sections, verbatim as authored, so a caller can assert each one
// survives, unmodified, into every skill that composes the source. It takes
// the same parsed sections bodySections produces, so it scans exactly what
// the renderer transcludes rather than re-parsing the body independently.
func clausesFromSections(sections map[string]string) []string {
	var clauses []string
	for _, key := range clauseSections {
		text, ok := sections[key]
		if !ok {
			continue
		}
		for line := range strings.SplitSeq(text, "\n") {
			if isLoadBearingClause(line) {
				clauses = append(clauses, line)
			}
		}
	}
	return clauses
}

// isLoadBearingClause reports whether a line states a load-bearing directive:
// NEVER, MUST, or MANDATORY anywhere on the line, or a leading "No " once any
// single markdown list marker is stripped ("- No ...", "3. No ...").
func isLoadBearingClause(line string) bool {
	if strings.Contains(line, "NEVER") ||
		strings.Contains(line, "MUST") ||
		strings.Contains(line, "MANDATORY") {
		return true
	}
	return strings.HasPrefix(stripListMarker(line), "No ")
}

// stripListMarker removes one leading markdown list marker, if present, so a
// leading "No " reads the same whether the clause is bulleted, starred, or
// numbered.
func stripListMarker(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "- "):
		return trimmed[2:]
	case strings.HasPrefix(trimmed, "* "):
		return trimmed[2:]
	default:
		return stripNumberedMarker(trimmed)
	}
}

// why: ASCII-only digit/space match Go regexp's \d and \s, not the wider
// Unicode sets, so a non-ASCII digit like "٣." is not mistaken for a marker.
func stripNumberedMarker(trimmed string) string {
	afterDigits := strings.TrimLeftFunc(trimmed, isASCIIDigit)
	if afterDigits == trimmed || !strings.HasPrefix(afterDigits, ".") {
		return trimmed
	}
	afterDot := afterDigits[1:]
	rest := strings.TrimLeftFunc(afterDot, isASCIISpace)
	if rest == afterDot {
		return trimmed
	}
	return rest
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isASCIISpace(r rune) bool {
	switch r {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	default:
		return false
	}
}
