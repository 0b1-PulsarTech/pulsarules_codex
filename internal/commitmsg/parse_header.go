package commitmsg

import "strings"

func (cursor *parser) parseType(msg *Message) bool {
	start := cursor.pos
	for cursor.pos < len(cursor.runes) && !isTypeDelimiter(cursor.peek()) {
		cursor.pos++
	}
	msg.Type = string(cursor.runes[start:cursor.pos])
	if msg.Type == "" {
		return false
	}

	next := cursor.peek()
	if next != '(' && next != '!' && next != ':' {
		msg.Type = ""
		msg.Description = strings.TrimSpace(string(cursor.runes[start:]))
		return false
	}
	return true
}

func (cursor *parser) parseScope(msg *Message) {
	if cursor.peek() == '(' {
		cursor.pos++
		scopeStart := cursor.pos
		for cursor.pos < len(cursor.runes) && cursor.peek() != ')' && cursor.peek() != '\n' {
			cursor.pos++
		}
		msg.Scope = string(cursor.runes[scopeStart:cursor.pos])
		if cursor.peek() == ')' {
			cursor.pos++
		}
	}
}

func (cursor *parser) parseBreakingAndColon(msg *Message) {
	if cursor.peek() == '!' {
		msg.Breaking = true
		cursor.pos++
	}
	if cursor.peek() == ':' {
		cursor.pos++
		if cursor.peek() == ' ' {
			cursor.pos++
		}
	}
}

func (cursor *parser) parseDescription(msg *Message) {
	descStart := cursor.pos
	for cursor.pos < len(cursor.runes) && cursor.peek() != '\n' {
		cursor.pos++
	}
	msg.Description = strings.TrimSpace(string(cursor.runes[descStart:cursor.pos]))

	if strings.HasPrefix(msg.Description, "[wip]") || strings.HasPrefix(msg.Description, "[WIP]") {
		msg.IsWIP = true
	}
}
