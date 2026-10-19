package budget

import (
	"strings"
	"unicode"
)

type scanner struct {
	text        string
	pos         int
	stack       []byte
	speculative int
	deepest     int
	widest      int
	templates   []int
	commentEnd  int
	commentFrom int
	regexOK     bool
	prevByte    byte
	prevWord    string
	fault       string
	firstRegex  int
}

func newScanner(text string) *scanner {
	return &scanner{text: text, regexOK: true, firstRegex: -1}
}

func (scan *scanner) run() int {
	for scan.pos < len(scan.text) && scan.deepest <= MaxNesting &&
		scan.widest <= maxSpeculative && scan.fault == "" {
		scan.step()
	}
	if scan.fault == "" && scan.pos >= len(scan.text) && len(scan.templates) > 0 {
		scan.fault = "an unterminated template"
	}
	return scan.deepest
}

func (scan *scanner) step() {
	char := scan.text[scan.pos]
	switch {
	case char == '/':
		scan.slash()
	case char == '\'' || char == '"':
		scan.skipString(char)
		scan.operand(char)
	case char == '`':
		scan.pos++
		scan.skipTemplate()
	case isIdentByte(char):
		scan.word()
	default:
		scan.bracket(char)
	}
}

func (scan *scanner) word() {
	start := scan.pos
	for scan.pos < len(scan.text) && isIdentByte(scan.text[scan.pos]) {
		scan.pos++
	}
	scan.prevWord = scan.text[start:scan.pos]
	scan.regexOK = startsExpression(scan.prevWord) && !isProperty(scan.text, start)
}

func (scan *scanner) bracket(char byte) {
	switch char {
	case '(':
		scan.openParen()
	case '[':
		scan.open(kindBracket)
	case '{':
		scan.openBrace()
	case '<':
		scan.openAngle()
	case ')', ']':
		scan.closeBracket(char)
	case '}':
		scan.closeBrace()
	case '>':
		scan.closeAngle()
	default:
		if strings.IndexByte(blanks, char) < 0 {
			scan.operator(char)
		}
		scan.pos++
	}
}

func (scan *scanner) peek(offset int) byte {
	if scan.pos+offset < 0 || scan.pos+offset >= len(scan.text) {
		return 0
	}
	return scan.text[scan.pos+offset]
}

func (scan *scanner) openParen() {
	wordEnd := scan.pos
	if scan.afterComment() {
		wordEnd = scan.commentFrom
	}
	switch word, _ := wordBefore(scan.text[:wordEnd]); {
	case speculative(scan.text, wordEnd, scan.pos):
		scan.open(kindArrow)
	case isHead(word):
		scan.open(kindHead)
	default:
		scan.open(kindParen)
	}
}

func (scan *scanner) afterComment() bool {
	return scan.commentEnd > 0 && scan.commentEnd <= scan.pos &&
		strings.TrimFunc(scan.text[scan.commentEnd:scan.pos], unicode.IsSpace) == ""
}

func (scan *scanner) noteComment() {
	if !scan.afterComment() {
		scan.commentFrom = scan.pos
	}
}

func (scan *scanner) openAngle() {
	if next := scan.peek(1); next != '=' && next != '<' && scan.peek(-1) != '<' {
		scan.push(kindAngle)
	}
	scan.operator('<')
	scan.pos++
}

func (scan *scanner) closeAngle() {
	if scan.peek(-1) != '=' && scan.top() == kindAngle {
		scan.drop()
	}
	scan.operator('>')
	scan.pos++
}

func (scan *scanner) closeBrace() {
	scan.closeBracket('}')
	last := len(scan.templates) - 1
	if scan.fault == "" && last >= 0 && len(scan.stack) == scan.templates[last] {
		scan.templates = scan.templates[:last]
		scan.skipTemplate()
	}
}

func isProperty(text string, start int) bool {
	return start > 0 && text[start-1] == '.' && !strings.HasSuffix(text[:start], "..")
}
