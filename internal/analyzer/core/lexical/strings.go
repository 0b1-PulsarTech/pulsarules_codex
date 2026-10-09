package lexical

import "strings"

type StringDelim struct {
	Open      string
	Close     string
	Escapes   bool
	Multiline bool
	Raw       bool
	Regex     bool
	Continues bool
	Interp    string
	Tag       string
}

type StringSyntax struct {
	Delims       []StringDelim
	RegexAfter   RegexContext
	Within       BlockDelim
	EndAtNewline bool
}

type codeFrame struct {
	delim *StringDelim
	depth int
	from  int
}

type literalScan struct {
	src       string
	comments  CommentSyntax
	syntax    StringSyntax
	spans     []Span
	commented []Span
	pos       int
	frames    []codeFrame
	work      int
	cutOff    bool
	cutAt     int
	lastEnd   int
	noRegexTo int
	parens    []bool
	headerEnd int
}

func scanLiterals(src string, comments CommentSyntax, syntax StringSyntax) Result {
	scan := &literalScan{src: src, comments: comments, syntax: syntax}
	spans := scan.run()
	return Result{
		Spans: spans, Comments: scan.commented,
		Complete: !scan.cutOff, Cut: max(scan.cutAt, 0), ToEOF: scan.cutAt < 0,
	}
}

func (scan *literalScan) run() []Span {
	for scan.pos < len(scan.src) {
		scan.step()
	}
	scan.endFrames()
	return scan.spans
}

func (scan *literalScan) step() {
	scan.work++
	rest := scan.src[scan.pos:]
	if skip := scan.commentLen(rest); skip > 0 {
		scan.commented = append(scan.commented, Span{Start: scan.pos, End: scan.pos + skip})
		scan.pos += skip
		return
	}
	before := scan.pos
	scan.consume(rest)
	if !isSpace(scan.src[before]) {
		scan.lastEnd = scan.pos
	}
}

func (scan *literalScan) consume(rest string) {
	if len(scan.frames) > 0 && scan.braceStep(rest[0]) {
		return
	}
	if delim := scan.opening(rest); delim != nil {
		scan.pos += len(delim.Open)
		scan.literal(delim, scan.pos-len(delim.Open))
		return
	}
	scan.trackParen(rest[0])
	scan.pos++
}

func (scan *literalScan) braceStep(char byte) bool {
	// A delim's Interp opens embedded code that its matching bare "}" closes.
	top := &scan.frames[len(scan.frames)-1]
	switch {
	case char == '{':
		top.depth++
	case char == '}' && top.depth > 0:
		top.depth--
	case char == '}':
		delim := top.delim
		scan.frames = scan.frames[:len(scan.frames)-1]
		scan.pos++
		scan.literal(delim, scan.pos)
		return true
	default:
		return false
	}
	scan.pos++
	return true
}

func (scan *literalScan) literal(delim *StringDelim, start int) {
	if delim.Regex {
		scan.regex(delim, start)
		return
	}
	for scan.pos < len(scan.src) {
		scan.work++
		rest := scan.src[scan.pos:]
		switch {
		case delim.Escapes && rest[0] == '\\':
			scan.pos += scan.escapeStep(rest, delim)
		case strings.HasPrefix(rest, delim.Close):
			scan.pos += len(delim.Close)
			scan.emit(start, delim)
			return
		case delim.Interp != "" && strings.HasPrefix(rest, delim.Interp):
			scan.emit(start, delim)
			scan.frames = append(scan.frames, codeFrame{delim: delim, from: scan.pos})
			scan.pos += len(delim.Interp)
			return
		case !delim.Multiline && rest[0] == '\n':
			scan.cutAtLine(start)
			scan.emit(start, delim)
			return
		default:
			scan.pos++
		}
	}
	// Running to EOF says nothing about where the scan lost its place.
	scan.cut(cutAtEOF)
	scan.emit(start, delim)
}

func (scan *literalScan) emit(start int, delim *StringDelim) {
	scan.pos = min(scan.pos, len(scan.src))
	span := Span{Start: start, End: scan.pos, Raw: delim.Raw, Regex: delim.Regex}
	scan.spans = append(scan.spans, span)
}

func escapeLen(rest string) int {
	if strings.HasPrefix(rest[1:], "\r\n") {
		return 1 + len("\r\n")
	}
	return 2
}

func (scan *literalScan) escapeStep(rest string, delim *StringDelim) int {
	// Only JS lets a backslash continue a quoted literal onto the next line.
	if !delim.Multiline && !delim.Continues && lineBreakAt(rest[1:]) {
		return 1
	}
	return escapeLen(rest)
}

func lineBreakAt(rest string) bool {
	return strings.HasPrefix(rest, "\n") || strings.HasPrefix(rest, "\r\n")
}
