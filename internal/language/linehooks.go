package language

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func noIdentBefore(src string, at int) bool { return lexer.IdentBefore(src, at) == "" }

func wordOrNone(src string, at int) bool {
	// A quote glued to any other word is a digit separator or an apostrophe.
	ident := lexer.IdentBefore(src, at)
	return ident == "" || slices.Contains([]string{
		"r", "u8", "u", "U", "L", "R", "return", "case", "typeof", "in", "of",
		"yield", "await", "void", "delete", "throw",
	}, ident)
}

func scalarStart(src string, at int) bool {
	lineStart := strings.LastIndexByte(src[:at], '\n') + 1
	head := src[lineStart:at]
	trimmed := strings.TrimRight(head, " \t")
	if trimmed == "" {
		return true
	}
	last := trimmed[len(trimmed)-1]
	spaced := len(trimmed) < len(head)
	return strings.IndexByte("[{,=", last) >= 0 || spaced && (last == ':' || last == '-')
}

func lineOpens(src string, at int) bool {
	if at == 0 || src[at-1] == '\n' {
		return true
	}
	before := src[at-1]
	switch src[at] {
	case '#':
		return before == ' ' || before == '\t'
	case '/':
		return before != ':'
	}
	return true
}
