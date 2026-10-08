package sfc

import "strings"

type markup struct {
	text, lower string
	pos         int
	open        []string
	blocks      int
	scripts     []Script
	hasModule   bool
	hasInstance bool
	dropped     bool
	unclosed    bool
	scanned     int
	bounded     bool
}

func (scan *markup) run() error {
	for scan.pos < len(scan.lower) {
		next := strings.IndexAny(scan.lower[scan.pos:], "<{")
		if next < 0 {
			scan.visit(len(scan.lower) - scan.pos)
			return nil
		}
		scan.visit(next)
		scan.pos += next
		if scan.lower[scan.pos] == '{' {
			scan.expression()
			continue
		}
		if err := scan.tag(); err != nil {
			return err
		}
	}
	return nil
}

func (scan *markup) tag() error {
	rest := scan.lower[scan.pos:]
	switch {
	case strings.HasPrefix(rest, "<!--"):
		scan.skipPast("-->")
	case strings.HasPrefix(rest, "<!"):
		scan.skipPast(">")
	case strings.HasPrefix(rest, "</"):
		scan.closing()
	case len(rest) > 1 && isNameStart(rest[1]):
		return scan.opening()
	default:
		scan.visit(1)
		scan.pos++
	}
	return nil
}

func (scan *markup) visit(width int) { scan.scanned += width }

func (scan *markup) skipPast(terminator string) {
	idx := strings.Index(scan.lower[scan.pos:], terminator)
	if idx < 0 {
		scan.visit(len(scan.lower) - scan.pos)
		scan.pos = len(scan.lower)
		return
	}
	scan.visit(idx + len(terminator))
	scan.pos += idx + len(terminator)
}

func (scan *markup) closing() {
	name := tagName(scan.lower[scan.pos+2:])
	scan.visit(len(name))
	scan.skipPast(">")
	const window = 32
	for idx := len(scan.open) - 1; idx >= 0 && idx >= len(scan.open)-window; idx-- {
		if scan.open[idx] == name {
			scan.open = scan.open[:idx]
			return
		}
	}
}

func (scan *markup) expression() {
	view := scan.view(scan.pos)
	braces := skipMustache(view, scan.pos)
	scan.visit(scanWork(braces, view, scan.pos))
	if !braces.Closed {
		scan.bounded = scan.bounded || braces.Exhausted
		if braces.End >= len(scan.lower) {
			scan.stop()
			return
		}
	}
	body := strings.TrimLeft(scan.lower[scan.pos+1:braces.End], blanks)
	switch {
	case strings.HasPrefix(body, "#"):
		scan.blocks++
	case isBlockClose(body):
		scan.blocks = max(scan.blocks-1, 0)
	}
	scan.pos = max(braces.End, scan.pos+1)
}

func (scan *markup) stop() {
	scan.unclosed = true
	scan.pos = len(scan.lower)
}

func (scan *markup) topLevel() bool {
	return len(scan.open) == 0 && scan.blocks == 0
}

func isNameStart(char byte) bool {
	return 'a' <= char && char <= 'z'
}

func tagName(rest string) string {
	end := strings.IndexAny(rest, " \t\r\n/><")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
