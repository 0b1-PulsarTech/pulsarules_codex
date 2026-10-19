package banner

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func leadingCommentLine(extension string, src []byte) (int, bool) {
	lang := language.ForExtension(extension)
	lines := strings.Split(string(lexer.TrimBOM(src)), "\n")
	var skip func(string) bool
	if isScript(lang.ID) {
		skip = isToolPragma
	}
	pos, ok := firstContentLine(lines, skip)
	if !ok {
		return 0, false
	}
	if lang.ID == language.JavaScript {
		offset, exempt := annotationBannerLine(lines[pos:])
		if exempt {
			return 0, false
		}
		if offset > 0 {
			return pos + offset + 1, true
		}
	}
	return pos + 1, isComment(lang, lines[pos])
}

func isScript(id language.ID) bool {
	return id == language.JavaScript || id == language.TypeScript
}

func isComment(lang language.Language, line string) bool {
	switch lang.ID {
	case language.Svelte, language.Vue:
		// Markup has no "//" or "/* */" comment; only HTML's counts at line 1.
		return strings.HasPrefix(strings.TrimSpace(line), "<!--")
	}
	// A fresh scanner is correct: no earlier line can have left a block open.
	scanner := lexer.NewCommentScanner(lang.Comments)
	return scanner.IsCommentLine(line) && !lang.IsPackageClause(line)
}
