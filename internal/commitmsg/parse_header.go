package commitmsg

import "strings"

func (p *parser) parseType(msg *Message) bool {
	start := p.pos
	for p.pos < len(p.runes) && !isTypeDelimiter(p.peek()) {
		p.pos++
	}
	msg.Type = string(p.runes[start:p.pos])
	if msg.Type == "" {
		return false
	}

	next := p.peek()
	if next != '(' && next != '!' && next != ':' {
		msg.Type = ""
		msg.Description = strings.TrimSpace(string(p.runes[start:]))
		return false
	}
	return true
}

// scope ::= "(" scopechars ")"
func (p *parser) parseScope(msg *Message) {
	if p.peek() == '(' {
		p.pos++
		scopeStart := p.pos
		for p.pos < len(p.runes) && p.peek() != ')' && p.peek() != '\n' {
			p.pos++
		}
		msg.Scope = string(p.runes[scopeStart:p.pos])
		if p.peek() == ')' {
			p.pos++
		}
	}
}

func (p *parser) parseBreakingAndColon(msg *Message) {
	if p.peek() == '!' {
		msg.Breaking = true
		p.pos++
	}
	if p.peek() == ':' {
		p.pos++
		if p.peek() == ' ' {
			p.pos++
		}
	}
}

func (p *parser) parseDescription(msg *Message) {
	descStart := p.pos
	for p.pos < len(p.runes) && p.peek() != '\n' {
		p.pos++
	}
	msg.Description = strings.TrimSpace(string(p.runes[descStart:p.pos]))

	if strings.HasPrefix(msg.Description, "[wip]") || strings.HasPrefix(msg.Description, "[WIP]") {
		msg.IsWIP = true
	}
}
