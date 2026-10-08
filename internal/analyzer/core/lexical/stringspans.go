package lexical

import (
	"sort"
	"strings"
)

type Span struct {
	Start int
	End   int
	Raw   bool
	Regex bool
}

type Result struct {
	Spans    []Span
	Regions  []Span
	Complete bool
	Cut      int
	ToEOF    bool
}

const cutAtEOF = -1

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
	if syntax.Within.Open == "" {
		whole := scanLiterals(src, comments, syntax)
		whole.Regions = []Span{{Start: 0, End: len(src)}}
		return whole
	}
	found := Result{Complete: true}
	for pos := 0; pos < len(src); {
		start, end, ok := nextRegion(src, pos, syntax.Within)
		if !ok {
			break
		}
		region := scanLiterals(src[start:end], comments, syntax)
		for _, span := range region.Spans {
			span.Start += start
			span.End += start
			found.Spans = append(found.Spans, span)
		}
		found.Regions = append(found.Regions, Span{Start: start, End: end})
		if !region.Complete {
			found.Complete, found.Cut, found.ToEOF = false, start+region.Cut, region.ToEOF
			break
		}
		pos = end
	}
	return found
}

func nextRegion(src string, from int, region BlockDelim) (start, end int, ok bool) {
	_, after, found := strings.Cut(src[from:], region.Open)
	if !found {
		return 0, 0, false
	}
	// The region starts right after the opener, so tag attributes are lexed like code.
	start = len(src) - len(after)
	body, _, _ := strings.Cut(after, region.Close)
	return start, start + len(body), true
}
