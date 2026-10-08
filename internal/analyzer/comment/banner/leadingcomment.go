package banner

import (
	"bytes"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func leadingCommentLine(lang pass.Language, extension string, src []byte) (int, bool) {
	// The BOM is dropped only so line 1 splits cleanly; IsGeneratedSource strips its own.
	body := bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))
	if lexer.IsGeneratedSource(body) {
		return 0, false
	}
	lines := strings.Split(string(body), "\n")
	var skip func(string) bool
	if isScriptExtension(extension) {
		skip = isToolPragma
	}
	pos, ok := firstContentLine(lines, skip)
	if !ok {
		return 0, false
	}
	if isAnnotationExtension(extension) {
		offset, exempt := annotationBannerLine(lines[pos:])
		if exempt {
			return 0, false
		}
		if offset > 0 {
			return pos + offset + 1, true
		}
	}
	return pos + 1, isComment(lang, extension, lines[pos])
}

func isScriptExtension(extension string) bool {
	return isAnnotationExtension(extension) || extension == ".ts" || extension == ".tsx"
}

func isAnnotationExtension(extension string) bool {
	switch extension {
	case ".js", ".mjs", ".cjs", ".jsx":
		return true
	}
	return false
}

func isComment(lang pass.Language, extension, line string) bool {
	switch strings.ToLower(extension) {
	case ".svelte", ".vue":
		// Markup has no "//" or "/* */" comment; only HTML's counts at line 1.
		return strings.HasPrefix(strings.TrimSpace(line), "<!--")
	}
	// A fresh scanner is correct: no earlier line can have left a block open.
	scanner := lexer.NewCommentScanner(lang.CommentSyntax())
	return scanner.IsCommentLine(line) && !lang.IsPackageDeclaration(line)
}
