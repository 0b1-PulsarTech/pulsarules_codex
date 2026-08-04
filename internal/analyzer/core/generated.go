package core

import (
	"bytes"
	"strings"
)

const (
	generatedPrefix = "// Code generated "
	generatedSuffix = " DO NOT EDIT."
)

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
				return false
			}
		}
	}
	return false
}

func isGeneratedMarker(line string) bool {
	if len(line) < len(generatedPrefix)+len(generatedSuffix) {
		return false
	}
	return strings.HasPrefix(line, generatedPrefix) &&
		strings.HasSuffix(line, generatedSuffix)
}
