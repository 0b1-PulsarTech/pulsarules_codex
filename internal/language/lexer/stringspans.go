package lexer

import "sort"

type Span struct {
	Start int
	End   int
	Raw   bool
	Regex bool
}

type Result struct {
	Spans    []Span
	Comments []Span
	Regions  []Span
	Complete bool
	Cut      int
	ToEOF    bool
}

const cutAtEOF = -1

func (scan *literalScan) cut(at int) {
	// Only a literal cut at a newline vouches for the text before it.
	if !scan.cutOff {
		scan.cutAt = at
	}
	scan.cutOff = true
}

func (scan *literalScan) endFrames() {
	// An embedded-code frame still open at EOF means a "${" never found its "}".
	if len(scan.frames) == 0 {
		return
	}
	scan.cut(cutAtEOF)
	if !scan.syntax.EndAtNewline {
		return
	}
	// The caller keeps an EOF cut, so the open literal must hide the rest like any EOF run.
	from := scan.frames[0].from
	for len(scan.spans) > 0 && scan.spans[len(scan.spans)-1].Start >= from {
		scan.spans = scan.spans[:len(scan.spans)-1]
	}
	scan.spans = append(scan.spans, Span{Start: from, End: len(scan.src)})
}

func (scan *literalScan) cutAtLine(at int) {
	// A string that hit its newline loses the scan's place unless the syntax ends it there.
	if !scan.syntax.EndAtNewline {
		scan.cut(at)
	}
}

func InSpans(spans []Span, offset int) bool {
	_, found := SpanAt(spans, offset)
	return found
}

func SpanAt(spans []Span, offset int) (Span, bool) {
	idx := sort.Search(len(spans), func(at int) bool { return spans[at].End > offset })
	if idx < len(spans) && spans[idx].Start <= offset {
		return spans[idx], true
	}
	return Span{}, false
}

func Literals(src string, comments CommentSyntax, syntax StringSyntax) Result {
	if syntax.Within == nil {
		whole := scanLiterals(src, comments, syntax)
		whole.Regions = []Span{{Start: 0, End: len(src)}}
		return whole
	}
	found := Result{Complete: true}
	for _, within := range syntax.Within(src) {
		start := within.Start
		region := scanLiterals(src[start:within.End], comments, syntax)
		for _, span := range region.Spans {
			span.Start += start
			span.End += start
			found.Spans = append(found.Spans, span)
		}
		for _, span := range region.Comments {
			span.Start += start
			span.End += start
			found.Comments = append(found.Comments, span)
		}
		found.Regions = append(found.Regions, within)
		if !region.Complete {
			found.Complete, found.Cut, found.ToEOF = false, start+region.Cut, region.ToEOF
			break
		}
	}
	return found
}
