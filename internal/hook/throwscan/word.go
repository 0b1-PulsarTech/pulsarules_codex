package throwscan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func standsAlone(src string, start, end int) bool {
	if start > 0 {
		before, _ := utf8.DecodeLastRuneInString(src[:start])
		if isIdentRune(before) || before == '.' {
			return false
		}
	}
	after, _ := utf8.DecodeRuneInString(src[end:])
	next, _ := utf8.DecodeRuneInString(strings.TrimLeftFunc(src[end:], unicode.IsSpace))
	if strings.Contains(src[start:end], ".") {
		return next == '('
	}
	// A key named throw ({ throw: 1 }) is data, not a statement.
	return !isIdentRune(after) && next != ':'
}

func isIdentRune(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '$'
}
