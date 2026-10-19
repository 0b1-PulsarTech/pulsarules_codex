package budget

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func (scan *scanner) valueNext() bool {
	if scan.prevWord != "" {
		// A regex may follow else/do, but the statement after them is a block, not an object.
		return lexer.StartsExpression(scan.prevWord) &&
			scan.prevWord != "else" && scan.prevWord != "do"
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
