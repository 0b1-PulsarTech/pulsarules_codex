package commitmsg

import (
	"strings"
	"unicode"
)

func (p *parser) remainingText() string {
	return strings.TrimSpace(string(p.runes[p.pos:]))
}

func (p *parser) consumeLine() string {
	start := p.pos
	for p.pos < len(p.runes) && p.peek() != '\n' {
		p.pos++
	}
	line := string(p.runes[start:p.pos])
	if p.peek() == '\n' {
		p.pos++
	}
	return line
}

func (p *parser) skipLeadingWhitespace() {
	for p.pos < len(p.runes) && (p.peek() == ' ' || p.peek() == '\t') {
		p.pos++
	}
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.runes) && p.peek() == ' ' {
		p.pos++
	}
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.runes)
}

func (p *parser) consume(r rune) bool {
	if p.pos < len(p.runes) && p.runes[p.pos] == r {
		p.pos++
		return true
	}
	return false
}

func (p *parser) peek() rune {
	if p.pos < len(p.runes) {
		return p.runes[p.pos]
	}
	return 0
}

func isEmojiChar(r rune) bool {
	return unicode.IsLower(r) || unicode.IsDigit(r) || r == '_'
}

func isTypeDelimiter(r rune) bool {
	return r == '(' || r == ':' || r == '!' || r == '\n' || r == ' '
}
