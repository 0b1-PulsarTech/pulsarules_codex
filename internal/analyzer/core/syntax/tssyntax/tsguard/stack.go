package tsguard

const (
	kindParen   = '('
	kindArrow   = 'a'
	kindHead    = 'h'
	kindBracket = '['
	kindBrace   = '{'
	kindObject  = 'o'
	kindAngle   = '<'
)

func (scan *scanner) open(kind byte) {
	scan.push(kind)
	scan.operator(scan.text[scan.pos])
	scan.pos++
}

func (scan *scanner) operator(char byte) {
	scan.prevByte = char
	scan.prevWord = ""
	scan.regexOK = !scan.postfix(char)
}

func (scan *scanner) postfix(char byte) bool {
	return (char == '+' || char == '-') && scan.pos > 0 && scan.text[scan.pos-1] == char
}

func (scan *scanner) operand(char byte) {
	scan.prevByte = char
	scan.prevWord = ""
	scan.regexOK = false
}

func (scan *scanner) push(kind byte) {
	scan.stack = append(scan.stack, kind)
	scan.deepest = max(scan.deepest, len(scan.stack))
	if kind == kindArrow {
		scan.speculative++
		scan.widest = max(scan.widest, scan.speculative)
	}
}

func (scan *scanner) top() byte {
	if len(scan.stack) == 0 {
		return 0
	}
	return scan.stack[len(scan.stack)-1]
}

func (scan *scanner) drop() {
	if scan.top() == kindArrow {
		scan.speculative--
	}
	scan.stack = scan.stack[:len(scan.stack)-1]
}

func (scan *scanner) closeBracket(closer byte) {
	for scan.top() == kindAngle {
		scan.drop()
	}
	top := scan.top()
	if !closes(closer, top) {
		scan.fault = "a closing bracket without its opener"
		scan.pos++
		return
	}
	scan.drop()
	scan.pos++
	if top == kindHead || (top == kindBrace && closer == '}') {
		scan.operator(closer)
		return
	}
	scan.operand(closer)
}

func closes(closer, top byte) bool {
	switch closer {
	case ')':
		return top == kindParen || top == kindArrow || top == kindHead
	case ']':
		return top == kindBracket
	}
	return top == kindBrace || top == kindObject
}
