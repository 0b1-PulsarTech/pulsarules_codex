package lexer

import (
	"strings"
	"unicode/utf8"
)

func (scan *literalScan) opening(rest string) (best *StringDelim, bestWidth int) {
	for idx := range scan.syntax.Delims {
		delim := &scan.syntax.Delims[idx]
		width := scan.openWidth(delim, rest)
		if width > 0 && outranks(delim, width, best, bestWidth) {
			best, bestWidth = delim, width
		}
	}
	if best != nil && best.Hashes {
		hashed := *best
		hashed.Close += strings.Repeat("#", bestWidth-len(best.Open))
		best = &hashed
	}
	return best, bestWidth
}

func (scan *literalScan) openWidth(delim *StringDelim, rest string) int {
	width := openerLen(delim, rest)
	if width == 0 || delim.Tag != "" && !scan.tagBefore(delim.Tag) {
		return 0
	}
	if delim.Regex && !scan.regexAllowed() || !scan.admits(delim, rest[width:]) {
		return 0
	}
	return width
}

func outranks(delim *StringDelim, width int, best *StringDelim, bestWidth int) bool {
	if best == nil || width > bestWidth {
		return true
	}
	return width == bestWidth && delim.Tag != "" && best.Tag == ""
}

func (scan *literalScan) tagBefore(tag string) bool {
	// The tag is a whole word right before the opener, blanks and comments aside.
	head := scan.src[:scan.lastEnd]
	if !strings.HasSuffix(head, tag) {
		return false
	}
	before := len(head) - len(tag)
	return before == 0 || !IsIdentByte(scan.src[before-1])
}

type StringDelim struct {
	Open      string
	Close     string
	Escapes   bool
	Multiline bool
	Raw       bool
	Regex     bool
	Continues bool
	Interp    string
	Tag       string
	// Hashes is Rust's r#"..."#, where a bare quote must not end the string.
	Hashes bool
	// Doubled is the C# verbatim string, which escapes a quote by repeating it.
	Doubled bool
	// Whole stops an apostrophe in prose from swallowing the rest of the file.
	Whole bool
	// Char keeps a Rust lifetime such as 'a from being read as a character literal.
	Char bool
	// Opens rejects a quote glued to a word, which is a digit separator or an apostrophe.
	Opens func(src string, at int) bool
}

func IdentBefore(src string, at int) string {
	start := at
	for start > 0 && isPrefixByte(src[start-1]) {
		start--
	}
	return src[start:at]
}

func openerLen(delim *StringDelim, rest string) int {
	if !delim.Hashes {
		if strings.HasPrefix(rest, delim.Open) {
			return len(delim.Open)
		}
		return 0
	}
	prefix := delim.Open[:len(delim.Open)-1]
	if !strings.HasPrefix(rest, prefix) {
		return 0
	}
	tail := rest[len(prefix):]
	width := len(rest) - len(strings.TrimLeft(tail, "#"))
	if width < len(rest) && rest[width] == delim.Open[len(delim.Open)-1] {
		return width + 1
	}
	return 0
}

func (scan *literalScan) admits(delim *StringDelim, body string) bool {
	if delim.Opens != nil && !delim.Opens(scan.src, scan.pos) {
		return false
	}
	return (!delim.Whole || closesOnLine(body, delim)) && (!delim.Char || charLen(body, delim) > 0)
}

func closesOnLine(body string, delim *StringDelim) bool {
	for pos := 0; pos < len(body) && body[pos] != '\n'; pos++ {
		switch {
		case delim.Escapes && body[pos] == '\\':
			if pos+1 < len(body) && body[pos+1] == '\n' {
				return false
			}
			pos++
		case strings.HasPrefix(body[pos:], delim.Close):
			return true
		}
	}
	return false
}

func charLen(body string, delim *StringDelim) int {
	line, _, _ := strings.Cut(body, "\n")
	quote := delim.Close[0]
	if strings.HasPrefix(line, `\`) && len(line) > 1 {
		if closer := strings.IndexByte(line[2:], quote); closer >= 0 {
			return closer + len(`\x`) + 1
		}
		return 0
	}
	char, size := utf8.DecodeRuneInString(line)
	if char == rune(quote) || char == utf8.RuneError || len(line) <= size || line[size] != quote {
		return 0
	}
	return size + 1
}

func isPrefixByte(char byte) bool {
	// String prefixes (r, b, f, u) are ASCII, and "$" is a JS-only identifier byte.
	return IsIdentByte(char) && char != '$' && char < utf8.RuneSelf
}
