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

// IsGeneratedSource reports whether src carries the Go generated-code marker
// ahead of its package clause, so analyzers can skip output no human can fix.
//
// It works on bytes rather than on go/ast because the pipeline must judge
// files it never parses: the AST cache holds no test file, yet mockgen writes
// its mocks to _test.go and the file-size analyzer reads them.
//
// It mirrors go/ast.IsGenerated: a `/* ... */` block comment ahead of the
// marker does not end the scan, it is skipped through like Go itself does,
// so a license block followed by the marker still counts. A marker WRITTEN
// inside a block comment does not count - only a `//` line comment does.
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
