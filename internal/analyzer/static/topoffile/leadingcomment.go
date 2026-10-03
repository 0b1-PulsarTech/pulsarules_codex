package topoffile

import (
	"bytes"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

func leadingCommentLine(lang core.Language, extension string, src []byte) (int, bool) {
	// The BOM is dropped only so line 1 splits cleanly; IsGeneratedSource strips its own.
	body := bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))
	if lexical.IsGeneratedSource(body) {
		return 0, false
	}
	lines := strings.Split(string(body), "\n")
	pos, ok := firstContentLine(lines)
	if !ok {
		return 0, false
	}
	return pos + 1, isComment(lang, extension, lines[pos])
}

func isComment(lang core.Language, extension, line string) bool {
	switch strings.ToLower(extension) {
	case ".svelte", ".vue":
		// Markup has no "//" or "/* */" comment; only HTML's counts at line 1.
		return strings.HasPrefix(strings.TrimSpace(line), "<!--")
	}
	// A fresh scanner is correct: no earlier line can have left a block open.
	scanner := lexical.NewCommentScanner(lang.CommentSyntax())
	return scanner.IsCommentLine(line) && !lang.IsPackageDeclaration(line)
}
