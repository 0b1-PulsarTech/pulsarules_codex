package topoffile

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

func firstContentLine(lines []string, skip func(string) bool) (int, bool) {
	for pos := 0; pos < len(lines); {
		line := lines[pos]
		if strings.TrimSpace(line) == "" || (skip != nil && skip(line)) {
			pos++
			continue
		}
		if lexical.IsContentStartLine(line) {
			return 0, false
		}
		span := lexical.ExemptHeaderSpan(lines[pos:])
		if span == 0 {
			return pos, true
		}
		pos += span
	}
	return 0, false
}
