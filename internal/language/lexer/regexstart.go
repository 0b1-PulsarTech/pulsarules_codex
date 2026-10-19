package lexer

import (
	"slices"
	"strings"
	"unicode/utf8"
)

type RegexContext struct {
	After      string
	Words      []string
	ParenWords []string
}

func (scan *literalScan) regexAllowed() bool {
	// A regex that found no end on this line was a division; the rest of the line agrees.
	if scan.pos < scan.noRegexTo {
		return false
	}
	end := scan.lastEnd
	if end == 0 {
		return true
	}
	prev := scan.src[end-1]
	switch {
	case IsIdentByte(prev):
		return scan.keywordBefore(end, scan.syntax.RegexAfter.Words)
	// The ")" of an "if (...)" header ends a condition, not an operand.
	case prev == ')' && end == scan.headerEnd:
		return true
	// A "</" closes a markup tag; it never opens a regex.
	case prev == '<' && end == scan.pos:
		return false
	case strings.IndexByte("+-!", prev) >= 0 && scan.postfixBefore(end):
		return false
	}
	return strings.IndexByte(scan.syntax.RegexAfter.After, prev) >= 0
}

func (scan *literalScan) postfixBefore(end int) bool {
	// "i++", "a[0]--" and "x!" end an operand, so the next "/" divides.
	last := scan.src[end-1]
	opStart := end - 1
	if last != '!' {
		opStart = end - 2
		if opStart < 0 || scan.src[opStart] != last {
			return false
		}
	}
	if opStart == 0 {
		return false
	}
	before := scan.src[opStart-1]
	return IsIdentByte(before) || before == ')' || before == ']'
}

func (scan *literalScan) keywordBefore(end int, words []string) bool {
	start := end
	for start > 0 && IsIdentByte(scan.src[start-1]) {
		start--
	}
	scan.work += end - start
	// After a dot the word is a property name, whatever it spells.
	if start > 0 && scan.src[start-1] == '.' {
		return false
	}
	return slices.Contains(words, scan.src[start:end])
}

func (scan *literalScan) trackParen(char byte) {
	headers := scan.syntax.RegexAfter.ParenWords
	switch {
	case len(headers) == 0:
	case char == '(':
		isHeader := scan.lastEnd > 0 && IsIdentByte(scan.src[scan.lastEnd-1]) &&
			scan.keywordBefore(scan.lastEnd, headers)
		scan.parens = append(scan.parens, isHeader)
	case char == ')' && len(scan.parens) > 0:
		last := len(scan.parens) - 1
		if scan.parens[last] {
			scan.headerEnd = scan.pos + 1
		}
		scan.parens = scan.parens[:last]
	}
}

func (scan *literalScan) regex(delim *StringDelim, start int) {
	inClass := false
	for scan.pos < len(scan.src) {
		scan.work++
		char := scan.src[scan.pos]
		switch {
		case char == '\\':
			scan.pos += escapeLen(scan.src[scan.pos:])
		case char == '\n':
			scan.divide(start+len(delim.Open), scan.pos)
			return
		case char == '[':
			inClass = true
			scan.pos++
		case char == ']':
			inClass = false
			scan.pos++
		case char == '/' && !inClass:
			scan.pos++
			scan.emit(start, delim)
			return
		default:
			scan.pos++
		}
	}
	scan.divide(start+len(delim.Open), len(scan.src))
}

func (scan *literalScan) divide(resume, lineEnd int) {
	// Rescanning from just past the slash keeps the scan linear: no regex opens before lineEnd.
	scan.pos = resume
	scan.noRegexTo = lineEnd
}

func isSpace(char byte) bool { return char == ' ' || char == '\t' || char == '\r' || char == '\n' }

func StartsExpression(word string) bool {
	switch word {
	case "return", "typeof", "case", "in", "of", "void", "yield", "await", "delete",
		"throw", "new", "instanceof", "else", "do", "extends", "default":
		return true
	}
	return false
}

func IsHeaderWord(word string) bool {
	return word == "if" || word == "while" || word == "for" || word == "with"
}

func IsIdentByte(char byte) bool {
	letter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
	digit := char >= '0' && char <= '9'
	return letter || digit || char == '_' || char == '$' || char >= utf8.RuneSelf
}

func JSStrings() StringSyntax {
	interp := "${"
	return StringSyntax{
		Delims: []StringDelim{
			{Open: `"`, Close: `"`, Escapes: true, Continues: true},
			{Open: `'`, Close: `'`, Escapes: true, Continues: true},
			{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: interp},
			{
				Open: "`", Close: "`", Escapes: true, Multiline: true,
				Raw: true, Interp: interp, Tag: "String.raw",
			},
			{Open: "/", Close: "/", Escapes: true, Regex: true},
		},
		RegexAfter: RegexContext{
			After: "(,=:[!&|?{};+-*%<>~^",
			Words: strings.Fields(
				"return typeof case in of void yield await delete throw new " +
					"instanceof else do extends default",
			),
			ParenWords: strings.Fields("if while for with"),
		},
	}
}
