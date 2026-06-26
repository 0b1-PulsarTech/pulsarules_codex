package commitmsg

import (
	"strings"
	"unicode"
)

func (cursor *parser) remainingText() string {
	return strings.TrimSpace(string(cursor.runes[cursor.pos:]))
}

func (cursor *parser) consumeLine() string {
	start := cursor.pos
	for cursor.pos < len(cursor.runes) && cursor.peek() != '\n' {
		cursor.pos++
	}
	line := string(cursor.runes[start:cursor.pos])
	if cursor.peek() == '\n' {
		cursor.pos++
	}
	return line
}

func (cursor *parser) skipLeadingWhitespace() {
	for cursor.pos < len(cursor.runes) && (cursor.peek() == ' ' || cursor.peek() == '\t') {
		cursor.pos++
	}
}

func (cursor *parser) skipSpaces() {
	for cursor.pos < len(cursor.runes) && cursor.peek() == ' ' {
		cursor.pos++
	}
}

func (cursor *parser) atEnd() bool {
	return cursor.pos >= len(cursor.runes)
}

func (cursor *parser) consume(char rune) bool {
	if cursor.pos < len(cursor.runes) && cursor.runes[cursor.pos] == char {
		cursor.pos++
		return true
	}
	return false
}

func (cursor *parser) peek() rune {
	if cursor.pos < len(cursor.runes) {
		return cursor.runes[cursor.pos]
	}
	return 0
}

func isEmojiChar(char rune) bool {
	return unicode.IsLower(char) || unicode.IsDigit(char) || char == '_'
}

func isTypeDelimiter(char rune) bool {
	return char == '(' || char == ':' || char == '!' || char == '\n' || char == ' '
}
