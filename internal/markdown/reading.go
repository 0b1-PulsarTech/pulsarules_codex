package markdown

import "strings"

const maxFenceShift = 3

type reading struct {
	quotes    bool
	tabs      bool
	tickInfo  bool
	bareClose bool
	strict    bool
	indented  bool
	ticks     tickEscaping
}

func readings() []reading {
	return []reading{
		// Rows 0 and 1 are the plain-ticks reading and the quote-aware, indented reading,
		// kept so the union never loses a region they covered.
		{ticks: literalTicks},
		{quotes: true, tickInfo: true, bareClose: true, indented: true, ticks: literalTicks},
		{
			quotes: true, tabs: true, tickInfo: true, bareClose: true,
			strict: true, indented: true, ticks: escapedTicks,
		},
	}
}

type lineShape struct {
	depth    int
	indent   int
	char     byte
	run      int
	bare     bool
	blank    bool
	indented bool
	tickInfo bool
}

func (how reading) parse(text string) lineShape {
	const indentedCode = 4

	var shape lineShape
	rest := text
	for how.quotes {
		var quoted bool
		if rest, quoted = stripQuote(rest); !quoted {
			break
		}
		shape.depth++
	}
	space := " "
	if how.tabs {
		space = " \t"
	}
	body := strings.TrimLeft(rest, space)
	shape.indent = columns(rest[:len(rest)-len(body)])
	shape.indented = shape.indent >= indentedCode || strings.HasPrefix(rest, "\t")
	shape.blank = strings.TrimSpace(body) == ""
	if body == "" || body[0] != '`' && body[0] != '~' {
		return shape
	}
	shape.fenceRun(body)
	return shape
}

func stripQuote(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > maxFenceShift || !strings.HasPrefix(trimmed, ">") {
		return line, false
	}
	return strings.TrimPrefix(trimmed[1:], " "), true
}

func columns(space string) int {
	const tabColumns = 4

	width := 0
	for _, char := range space {
		width++
		if char == '\t' {
			width += tabColumns - 1
		}
	}
	return width
}

func (shape *lineShape) fenceRun(body string) {
	for shape.run < len(body) && body[shape.run] == body[0] {
		shape.run++
	}
	shape.char = body[0]
	shape.bare = strings.TrimSpace(body[shape.run:]) == ""
	shape.tickInfo = shape.char == '`' && strings.Contains(body[shape.run:], "`")
}
