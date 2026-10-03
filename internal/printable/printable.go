package printable

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func Path(path string) string {
	quoted := strconv.Quote(path)
	// An arrow or an edge space would blur the "old -> new" rename form.
	ambiguous := strings.Contains(path, "->") || strings.TrimSpace(path) != path
	if quoted[1:len(quoted)-1] != path || ambiguous {
		return quoted
	}
	return path
}

func Line(text string) string {
	var out strings.Builder
	for pos := 0; pos < len(text); {
		char, size := utf8.DecodeRuneInString(text[pos:])
		switch {
		case char == utf8.RuneError && size == 1:
			fmt.Fprintf(&out, `\x%02x`, text[pos])
		case strconv.IsPrint(char):
			out.WriteRune(char)
		default:
			quoted := strconv.Quote(text[pos : pos+size])
			out.WriteString(quoted[1 : len(quoted)-1])
		}
		pos += size
	}
	return out.String()
}
