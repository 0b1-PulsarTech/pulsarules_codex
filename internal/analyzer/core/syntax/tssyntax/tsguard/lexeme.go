package tsguard

import "strings"

func (scan *scanner) skipLineComment() {
	scan.noteComment()
	scan.pos = lineEnd(scan.text, scan.pos+2)
	scan.commentEnd = scan.pos
}

func (scan *scanner) skipBlockComment() {
	scan.noteComment()
	idx := strings.Index(scan.text[scan.pos+2:], "*/")
	if idx < 0 {
		scan.pos = len(scan.text)
		scan.fault = "an unterminated comment"
		return
	}
	scan.pos += 2 + idx + 2
	scan.commentEnd = scan.pos
}

func (scan *scanner) skipString(quote byte) {
	scan.pos++
	for scan.pos < len(scan.text) {
		switch scan.text[scan.pos] {
		case '\\':
			scan.pos += escapeWidth(scan.text, scan.pos)
			continue
		case quote:
			scan.pos++
			return
		}
		if terminatorAt(scan.text, scan.pos) {
			return
		}
		scan.pos++
	}
	scan.fault = "an unterminated string"
}

func (scan *scanner) skipRegex() {
	inClass := false
	if scan.firstRegex < 0 {
		scan.firstRegex = scan.pos
	}
	scan.pos++
	for scan.pos < len(scan.text) {
		char := scan.text[scan.pos]
		switch {
		case char == '\\':
			scan.pos += escapeWidth(scan.text, scan.pos)
			continue
		case terminatorAt(scan.text, scan.pos):
			scan.operand('/')
			return
		case char == '[':
			inClass = true
		case char == ']':
			inClass = false
		case char == '/' && !inClass:
			scan.pos++
			scan.operand('/')
			return
		}
		scan.pos++
	}
	scan.fault = "an unterminated regular expression"
}

func escapeWidth(text string, pos int) int {
	const plain, withCRLF = 2, 3
	if strings.HasPrefix(text[pos+1:], "\r\n") {
		return withCRLF
	}
	return plain
}

func lineEnd(text string, from int) int {
	for idx := from; idx < len(text); idx++ {
		if terminatorAt(text, idx) {
			return idx
		}
	}
	return len(text)
}

func terminatorAt(text string, idx int) bool {
	const lead, middle, lineSep, paraSep = 0xE2, 0x80, 0xA8, 0xA9
	switch text[idx] {
	case '\n', '\r':
		return true
	case lead:
		return idx+2 < len(text) && text[idx+1] == middle &&
			(text[idx+2] == lineSep || text[idx+2] == paraSep)
	}
	return false
}

func (scan *scanner) skipTemplate() {
	for scan.pos < len(scan.text) {
		switch scan.text[scan.pos] {
		case '\\':
			scan.pos += escapeWidth(scan.text, scan.pos)
			continue
		case '`':
			scan.pos++
			scan.operand('`')
			return
		case '$':
			if scan.peek(1) == '{' {
				scan.templates = append(scan.templates, len(scan.stack))
				scan.open(kindBrace)
				scan.pos++
				return
			}
		}
		scan.pos++
	}
	scan.fault = "an unterminated template"
}
