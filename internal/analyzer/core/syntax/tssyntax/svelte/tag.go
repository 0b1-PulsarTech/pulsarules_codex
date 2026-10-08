package svelte

import (
	"slices"
	"strings"
)

func (scan *markup) opening() error {
	name := tagName(scan.lower[scan.pos+1:])
	exact := tagName(scan.text[scan.pos+1:])
	attrStart := scan.pos + 1 + len(name)
	end, stop := scan.tagEnd(attrStart)
	scan.visit(stop - attrStart)
	if end == badBrace {
		scan.stop()
		return nil
	}
	if end == textTag {
		scan.pos = stop
		return nil
	}
	if end < 0 {
		scan.pos = len(scan.lower)
		return nil
	}
	attrs := scan.lower[attrStart:end]
	selfClosed := strings.HasSuffix(attrs, "/")
	scan.pos = end + 1
	switch {
	case selfClosed:
	case isRawElement(exact):
		return scan.raw(name, attrs)
	case !isVoid(name) && !endsImplicitly(name):
		scan.open = append(scan.open, name)
	}
	return nil
}

func (scan *markup) raw(name, attrs string) error {
	start := scan.pos
	stop := strings.Index(scan.lower[start:], "</"+name)
	scan.visit(scanWidth(stop, len(scan.lower)-start))
	record := name == "script" && scan.topLevel() && scan.wanted(attrs)
	scan.noteDropped(name)
	if stop < 0 {
		scan.pos = len(scan.lower)
		if record {
			return errUnterminated
		}
		return nil
	}
	stop += start
	scan.pos = stop
	scan.skipPast(">")
	if record {
		scan.scripts = append(scan.scripts, Script{
			TypeScript: isTypeScript(attrs),
			Text:       blank(scan.text, start, stop),
		})
	}
	return nil
}

func scanWidth(found, rest int) int {
	if found < 0 {
		return rest
	}
	return found
}

func (scan *markup) wanted(attrs string) bool {
	if !isJavaScript(attrs) {
		return false
	}
	if isModule(attrs) {
		taken := scan.hasModule
		scan.hasModule = true
		return !taken
	}
	taken := scan.hasInstance
	scan.hasInstance = true
	return !taken
}

const (
	textTag  = -2
	badBrace = -3
)

func isRawElement(name string) bool { return name == "script" || name == "style" }

func (scan *markup) noteDropped(name string) {
	if name == "script" && !scan.topLevel() && !slices.Contains(scan.open, "svelte:head") {
		scan.dropped = true
	}
}

func (scan *markup) attrBraces(pos int) int {
	braces := skipBraces(scan.view(pos), pos)
	if braces.Closed {
		return max(braces.End, pos+1)
	}
	scan.bounded = scan.bounded || braces.Exhausted
	if braces.End >= len(scan.lower) {
		return -1
	}
	return max(braces.End, pos+1)
}

func (scan *markup) tagEnd(from int) (end, stop int) {
	lower := scan.lower
	var quote byte
	for pos := from; pos < len(lower); pos++ {
		char := lower[pos]
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case opensQuote(lower, pos):
			quote = char
		case char == '{':
			next := scan.attrBraces(pos)
			if next < 0 {
				return badBrace, len(lower)
			}
			pos = next - 1
		case char == '<':
			return textTag, pos
		case char == '>':
			return pos, pos + 1
		}
	}
	return -1, len(lower)
}

func isVoid(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta",
		"param", "source", "track", "wbr":
		return true
	}
	return false
}

func endsImplicitly(name string) bool {
	switch name {
	case "p", "li", "dt", "dd", "option", "optgroup", "tr", "td", "th", "thead", "tbody",
		"tfoot", "colgroup", "rb", "rt", "rtc", "rp":
		return true
	}
	return false
}

func opensQuote(lower string, pos int) bool {
	char := lower[pos]
	return (char == '"' || char == '\'') && afterEquals(lower, pos)
}

func afterEquals(lower string, pos int) bool {
	idx := pos - 1
	for idx >= 0 && strings.IndexByte(blanks, lower[idx]) >= 0 {
		idx--
	}
	return idx >= 0 && lower[idx] == '='
}
