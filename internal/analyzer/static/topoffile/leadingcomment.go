package topoffile

import (
	"bytes"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func leadingCommentLine(lang core.Language, extension string, src []byte) (int, bool) {
	const utf8BOM = "\xef\xbb\xbf"
	lines := strings.Split(string(bytes.TrimPrefix(src, []byte(utf8BOM))), "\n")
	pos := 0
	for pos < len(lines) {
		line := lines[pos]
		if strings.TrimSpace(line) == "" {
			pos++
			continue
		}
		if core.IsGeneratedMarkerLine(line) {
			return 0, false
		}
		if core.IsContentStartLine(line) {
			return 0, false
		}
		if span := core.ExemptHeaderSpan(lines[pos:]); span > 0 {
			pos += span
			continue
		}
		return pos + 1, isComment(lang, extension, line)
	}
	return 0, false
}

func isComment(lang core.Language, extension, line string) bool {
	switch strings.ToLower(extension) {
	case ".svelte", ".vue":
		// Markup has no "//" or "/* */" comment; only HTML's counts at line 1.
		return strings.HasPrefix(strings.TrimSpace(line), "<!--")
	}
	// A fresh scanner is correct: no earlier line can have left a block open.
	scanner := core.NewCommentScanner(lang.CommentSyntax())
	return scanner.IsCommentLine(line) && !lang.IsPackageDeclaration(line)
}
