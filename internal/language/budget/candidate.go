package budget

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
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
	if !lexer.IsIdentByte(last) {
		return "", last
	}
	start := len(trimmed)
	for start > 0 && lexer.IsIdentByte(trimmed[start-1]) {
		start--
	}
	return trimmed[start:], last
}

func isCall(word string, last byte) bool {
	if word != "" {
		return !lexer.StartsExpression(word)
	}
	return last == ')' || last == ']'
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
	if !lexer.IsIdentByte(rest[0]) || '0' <= rest[0] && rest[0] <= '9' {
		return false
	}
	end := 0
	for end < len(rest) && lexer.IsIdentByte(rest[end]) {
		end++
	}
	rest = strings.TrimLeft(rest[end:], blanks)
	return rest != "" && strings.IndexByte(",=):?", rest[0]) >= 0
}
