package historynarration

import (
	"slices"
	"strings"
	"unicode/utf8"
)

func (quoting rules) quoteLen(line string, at int) int {
	quote := line[at]
	if quoting.open == openNone || quote != '"' && quote != '\'' {
		return 0
	}
	if quote == '\'' && quoting.open == openRust {
		return rustCharLen(line, at)
	}
	if !quoting.opens(line, at) {
		return 0
	}
	end, closed := quoting.stringEnd(line, at)
	if !closed && quote == '\'' {
		return 0
	}
	return end - at
}

func (quoting rules) opens(line string, at int) bool {
	switch quoting.open {
	case openScalar:
		return scalarStart(line[:at])
	case openCLike:
		return line[at] == '"' || cLikeOpens(identBefore(line, at))
	case openAlways, openRust:
		return true
	}
	return false
}

func rustCharLen(line string, at int) int {
	const escapedWidth = len("'\\x'")
	rest := line[at+1:]
	if strings.HasPrefix(rest, "\\") && len(rest) > 1 {
		closer := strings.IndexByte(rest[2:], '\'')
		if closer < 0 {
			return 0
		}
		return closer + escapedWidth
	}
	char, size := utf8.DecodeRuneInString(rest)
	if char == '\'' || char == utf8.RuneError || len(rest) <= size || rest[size] != '\'' {
		return 0
	}
	return size + 2
}

func scalarStart(head string) bool {
	trimmed := strings.TrimRight(head, " \t")
	if trimmed == "" {
		return true
	}
	last := trimmed[len(trimmed)-1]
	spaced := len(trimmed) < len(head)
	return strings.IndexByte("[{,=", last) >= 0 || spaced && (last == ':' || last == '-')
}

func cLikeOpens(ident string) bool {
	return ident == "" || slices.Contains([]string{
		"r", "u8", "u", "U", "L", "R", "return", "case", "typeof", "in", "of",
		"yield", "await", "void", "delete", "throw",
	}, ident)
}
