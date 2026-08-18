package core

import (
	"bytes"
	"strings"
)

// The Go convention for machine-written source: a line reading exactly
// "// Code generated <generator> DO NOT EDIT." placed before the package
// clause. Detecting by content rather than by path is what keeps the rule
// portable - a path list (internal/dbgen, *_mock_test.go) has to be restated
// and maintained in every consumer repository.
const (
	generatedPrefix = "// Code generated "
	generatedSuffix = " DO NOT EDIT."
)

// IsGeneratedSource reports whether src carries the Go generated-code
// marker ahead of the package clause, so analyzers can skip files no
// human can fix. It scans bytes, not go/ast, because mockgen writes
// mocks to _test.go, unparsed by the AST cache; it mirrors
// go/ast.IsGenerated - a marker inside a block comment doesn't count.
func IsGeneratedSource(src []byte) bool {
	inBlock := false
	for line := range bytes.Lines(src) {
		rest := strings.TrimSpace(string(line))
	inner:
		for {
			switch {
			case inBlock:
				closeIdx := strings.Index(rest, "*/")
				if closeIdx == -1 {
					break inner
				}
				inBlock = false
				rest = strings.TrimSpace(rest[closeIdx+len("*/"):])
			case rest == "":
				break inner
			case strings.HasPrefix(rest, "//"):
				if isGeneratedMarker(rest) {
					return true
				}
				break inner
			case strings.HasPrefix(rest, "/*"):
				inBlock = true
				rest = strings.TrimSpace(rest[len("/*"):])
			default:
				// The first line of real code (for Go, the package clause).
				// Go only honours the marker above it, and so do we.
				return false
			}
		}
	}
	return false
}

// why: this is Go's own `^// Code generated .* DO NOT EDIT\.$` without a
// regexp, and the length guard is what stops prefix and suffix from
// overlapping on a line too short to hold both.
func isGeneratedMarker(line string) bool {
	if len(line) < len(generatedPrefix)+len(generatedSuffix) {
		return false
	}
	return strings.HasPrefix(line, generatedPrefix) &&
		strings.HasSuffix(line, generatedSuffix)
}
