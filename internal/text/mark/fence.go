package mark

type fence struct {
	char   byte
	run    int
	depth  int
	indent int
	start  int
}

type readingScan struct {
	how          reading
	regions      []Region
	open         fence
	codeMayStart bool
}

func (scan *readingScan) line(text string, offset int) {
	shape := scan.how.parse(text)
	end := offset + len(text)
	switch {
	case scan.open.char != 0:
		scan.inFence(shape, end)
	case scan.opens(shape):
		scan.open = fence{
			char: shape.char, run: shape.run, depth: shape.depth,
			indent: shape.indent, start: offset,
		}
	case scan.how.indented && scan.codeMayStart && shape.indented:
		scan.regions = append(scan.regions, Region{Start: offset, End: end})
	default:
		scan.regions = append(scan.regions, inlineSpans(text, offset, scan.how.ticks)...)
		scan.codeMayStart = shape.blank
		return
	}
	scan.codeMayStart = scan.open.char == 0
}

func (scan *readingScan) opens(shape lineShape) bool {
	const minFenceRun = 3
	// A backtick fence's info string holds no backtick; otherwise the run is inline code.
	return shape.run >= minFenceRun && (!scan.how.tickInfo || !shape.tickInfo)
}

func (scan *readingScan) inFence(shape lineShape, end int) {
	open := &scan.open
	if shape.char != open.char || shape.run < open.run || scan.how.bareClose && !shape.bare {
		return
	}
	switch {
	case !scan.how.strict, closes(shape, *open):
		scan.regions = append(scan.regions, Region{Start: open.start, End: end})
		scan.open = fence{}
	case shape.depth < open.depth || shape.depth == open.depth && shape.indent < open.indent:
		// A quote or list may have ended, so this line closes the fence or opens a new one.
		open.depth, open.indent, open.run = shape.depth, shape.indent, shape.run
	}
}

func closes(shape lineShape, open fence) bool {
	return shape.depth == open.depth &&
		shape.indent >= open.indent && shape.indent <= max(maxFenceShift, open.indent)
}

func (scan *readingScan) finish(total int) []Region {
	// An unclosed fence runs to the end of the file, the way a renderer treats it.
	if scan.open.char != 0 {
		scan.regions = append(scan.regions, Region{Start: scan.open.start, End: total})
	}
	return scan.regions
}
