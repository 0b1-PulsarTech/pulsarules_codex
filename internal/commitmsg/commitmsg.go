package commitmsg

import (
	"strings"
)

type parser struct {
	runes []rune
	pos   int
}

func Parse(raw string) Message {
	msg := Message{Raw: raw}
	cursor := &parser{runes: []rune(raw)}
	cursor.parse(&msg)
	// Lenient: a malformed message returns whatever parsed; the commit rules validate it.
	return msg
}

func (cursor *parser) parse(msg *Message) {
	cursor.skipLeadingWhitespace()
	cursor.parseEmojis(msg)
	cursor.skipSpaces()

	if cursor.isSpecialInitial(msg) {
		msg.IsInitial = true
		msg.Description = cursor.remainingText()
		return
	}
	if cursor.isSpecialMerge(msg) {
		msg.IsMerge = true
		msg.Description = cursor.remainingText()
		return
	}

	cursor.parseHeader(msg)
	if cursor.atEnd() {
		return
	}
	cursor.parseBodyAndFooters(msg)
}

func (cursor *parser) parseEmojis(msg *Message) {
	for {
		name, ok := cursor.tryEmoji()
		if !ok {
			break
		}
		// Uncapped: truncating here would misread extras as the type.
		msg.Emojis = append(msg.Emojis, name)
		cursor.skipSpaces()
	}
}

func (cursor *parser) tryEmoji() (string, bool) {
	const maxEmojiNameLength = 32

	start := cursor.pos
	if !cursor.consume(':') {
		cursor.pos = start
		return "", false
	}
	nameStart := cursor.pos
	for cursor.pos < len(cursor.runes) {
		char := cursor.runes[cursor.pos]
		if char == ':' {
			break
		}
		// A name this long is unterminated punctuation, not an emoji.
		if char == ' ' || char == '\n' || cursor.pos-nameStart > maxEmojiNameLength {
			cursor.pos = start
			return "", false
		}
		if !isEmojiChar(char) {
			cursor.pos = start
			return "", false
		}
		cursor.pos++
	}
	if cursor.pos == nameStart || cursor.pos >= len(cursor.runes) {
		cursor.pos = start
		return "", false
	}
	name := string(cursor.runes[nameStart:cursor.pos])
	cursor.pos++
	return name, true
}

func (cursor *parser) parseHeader(msg *Message) {
	if !cursor.parseType(msg) {
		return
	}
	cursor.parseScope(msg)
	cursor.parseBreakingAndColon(msg)
	cursor.parseDescription(msg)
}

func (cursor *parser) parseBodyAndFooters(msg *Message) {
	if cursor.peek() == '\n' {
		cursor.pos++
	}
	if cursor.peek() != '\n' {
		return
	}
	cursor.pos++

	var bodyLines []string
	for cursor.pos < len(cursor.runes) {
		line := cursor.consumeLine()
		if isFooterLine(line) {
			footer := parseFooter(line)
			if footer.Key != "" {
				msg.Footers = append(msg.Footers, footer)
			}
			break
		}
		if line == "" {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	msg.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	for cursor.pos < len(cursor.runes) {
		line := cursor.consumeLine()
		if line == "" {
			continue
		}
		footer := parseFooter(line)
		if footer.Key != "" {
			msg.Footers = append(msg.Footers, footer)
		}
	}
}
