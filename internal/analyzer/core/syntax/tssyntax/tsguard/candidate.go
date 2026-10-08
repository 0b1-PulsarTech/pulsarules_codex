package tsguard

import (
	"strings"
	"unicode/utf8"
)

const blanks = " \t\r\n"

func speculative(text string, wordEnd, pos int) bool {
	word, last := wordBefore(text[:wordEnd])
	if word == "async" {
		return true
	}
	// The fork re-parses a parenthesis that may start arrow parameters, so nested
	// ones cost exponential or quadratic time and memory. Calls are unambiguous,
	// except after a type argument list, which it also tries as generic arrow parameters.
	if last == '>' && !strings.HasSuffix(strings.TrimRight(text[:wordEnd], blanks), "=>") {
		return true
	}
	if isCall(word, last) {
		return false
	}
	return startsParameters(text[pos+1:])
}

func wordBefore(prefix string) (string, byte) {
	trimmed := strings.TrimRight(prefix, blanks)
	if trimmed == "" {
		return "", 0
	}
	last := trimmed[len(trimmed)-1]
	if !isIdentByte(last) {
		return "", last
	}
	start := len(trimmed)
	for start > 0 && isIdentByte(trimmed[start-1]) {
		start--
	}
	return trimmed[start:], last
}

func isCall(word string, last byte) bool {
	if word != "" {
		return !startsExpression(word)
	}
	return last == ')' || last == ']'
}

func startsExpression(word string) bool {
	switch word {
	case "return", "typeof", "void", "delete", "await", "yield", "new", "throw",
		"case", "in", "of", "instanceof", "else", "do", "extends", "default":
		return true
	}
	return false
}

func startsParameters(rest string) bool {
	rest = strings.TrimLeft(rest, blanks)
	if rest == "" {
		return false
	}
	switch rest[0] {
	case '{', '[':
		return true
	case '.':
		return strings.HasPrefix(rest, "...")
	}
	if !isIdentByte(rest[0]) || '0' <= rest[0] && rest[0] <= '9' {
		return false
	}
	end := 0
	for end < len(rest) && isIdentByte(rest[end]) {
		end++
	}
	rest = strings.TrimLeft(rest[end:], blanks)
	return rest != "" && strings.IndexByte(",=):?", rest[0]) >= 0
}

func isIdentByte(char byte) bool {
	return char == '_' || char == '$' || char >= utf8.RuneSelf ||
		'0' <= char && char <= '9' || 'a' <= char && char <= 'z' || 'A' <= char && char <= 'Z'
}
