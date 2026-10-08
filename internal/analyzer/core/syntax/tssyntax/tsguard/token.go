package tsguard

import "strings"

func isHead(word string) bool {
	switch word {
	case "if", "while", "for", "with":
		return true
	}
	return false
}

func (scan *scanner) valueNext() bool {
	if scan.prevWord != "" {
		return startsExpression(scan.prevWord)
	}
	return scan.prevByte != 0 && strings.IndexByte("=(,[:?&|!+-*%<~^", scan.prevByte) >= 0
}

func (scan *scanner) openBrace() {
	if scan.valueNext() {
		scan.open(kindObject)
		return
	}
	scan.open(kindBrace)
}

func (scan *scanner) slash() {
	rest := scan.text[scan.pos:]
	switch {
	case strings.HasPrefix(rest, "//"):
		scan.skipLineComment()
	case strings.HasPrefix(rest, "/*"):
		scan.skipBlockComment()
	case scan.regexOK && scan.peek(-1) != '<':
		scan.skipRegex()
	default:
		scan.operator('/')
		scan.pos++
	}
}
