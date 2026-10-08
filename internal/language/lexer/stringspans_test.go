package lexer

import (
	"slices"
	"strings"
	"testing"
)

func TestLiterals_WithinConfinesToRegions(t *testing.T) {
	t.Parallel()

	comments, strs := testSyntax()
	strs.Within = BlockDelim{Open: "<script", Close: "</script>"}
	testCases := []struct {
		name string
		src  string
		want []string
	}{
		{"markup quote outside the script", `<p class="a">it's</p>`, nil},
		{"script body", `<p>'x'</p><script>a = 'y'</script><p>'z'</p>`, []string{`'y'`}},
		{"two scripts", `<script>'a'</script> '!' <script>'b'</script>`, []string{`'a'`, `'b'`}},
		{"unclosed script runs to EOF", `<script>'a'`, []string{`'a'`}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := spanTexts(testCase.src, Literals(testCase.src, comments, strs).Spans)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Literals(%q) = %q, want %q", testCase.src, got, testCase.want)
			}
		})
	}
}

func TestLiterals_WithinCompleteNeedsEveryRegion(t *testing.T) {
	t.Parallel()

	comments, strs := testSyntax()
	strs.Within = BlockDelim{Open: "<script", Close: "</script>"}
	testCases := []struct {
		name string
		src  string
		want bool
	}{
		{"no script", `<p>'x</p>`, true},
		{"two clean scripts", `<script>'a'</script> <script>'b'</script>`, true},
		{"second script is cut off", `<script>'a'</script> <script>/* b`, false},
		{"first script is cut off", `<script>/* a</script> <script>'b'</script>`, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := Literals(testCase.src, comments, strs).Complete; got != testCase.want {
				t.Errorf("Complete = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}

func TestLiterals_RegionsAreTheScannedText(t *testing.T) {
	t.Parallel()

	comments, strs := testSyntax()
	whole := Literals("a 'x'", comments, strs).Regions
	if len(whole) != 1 || whole[0] != (Span{Start: 0, End: 5}) {
		t.Errorf("whole-file regions = %v, want one span over the file", whole)
	}
	strs.Within = BlockDelim{Open: "<script", Close: "</script>"}
	src := `<p>a</p><script>'y'</script><i/><script>'z'`
	got := Literals(src, comments, strs).Regions
	want := []Span{{Start: 15, End: 19}, {Start: 39, End: 43}}
	if !slices.Equal(got, want) {
		t.Errorf("Regions = %v, want %v", got, want)
	}
	if none := Literals("<p>'x'</p>", comments, strs).Regions; len(none) != 0 {
		t.Errorf("Regions without a script = %v, want none", none)
	}
}

func TestInSpans(t *testing.T) {
	t.Parallel()

	spans := []Span{{Start: 2, End: 5}, {Start: 8, End: 9}}
	testCases := []struct {
		offset int
		want   bool
	}{
		{0, false},
		{1, false},
		{2, true},
		{4, true},
		{5, false},
		{7, false},
		{8, true},
		{9, false},
		{100, false},
	}
	for _, testCase := range testCases {
		if got := InSpans(spans, testCase.offset); got != testCase.want {
			t.Errorf("InSpans(%d) = %v, want %v", testCase.offset, got, testCase.want)
		}
	}
	if InSpans(nil, 0) {
		t.Error("InSpans(nil, 0) = true, want false")
	}
}

func TestSpanAt(t *testing.T) {
	t.Parallel()

	spans := []Span{{Start: 2, End: 5}, {Start: 8, End: 9, Raw: true}}
	testCases := []struct {
		name   string
		offset int
		want   Span
		found  bool
	}{
		{"before", 1, Span{}, false},
		{"first start", 2, spans[0], true},
		{"first last byte", 4, spans[0], true},
		{"between", 6, Span{}, false},
		{"raw span", 8, spans[1], true},
		{"past the end", 9, Span{}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, found := SpanAt(spans, testCase.offset)
			if got != testCase.want || found != testCase.found {
				t.Errorf("SpanAt(%d) = %v, %v", testCase.offset, got, found)
			}
		})
	}
}

func TestLiterals_CutInsideARegionIsAbsolute(t *testing.T) {
	t.Parallel()

	comments, _ := testSyntax()
	strs := StringSyntax{
		Delims: []StringDelim{{Open: "'", Close: "'", Escapes: true}},
		Within: BlockDelim{Open: "<script", Close: "</script>"},
	}
	const src = "<i/><script>x = 'y\nz</script><script>'w'</script>"
	got := Literals(src, comments, strs)
	want := strings.Index(src, "'y")
	if got.Complete || got.Cut != want || got.ToEOF {
		t.Errorf("Complete, Cut = %v, %d, want a cut at %d", got.Complete, got.Cut, want)
	}
	if len(got.Regions) != 1 || len(got.Spans) != 1 {
		t.Errorf("regions %v, spans %v, want only the first script", got.Regions, got.Spans)
	}
}

func TestLiterals_EOFInsideARegionIsFlagged(t *testing.T) {
	t.Parallel()

	comments, _ := testSyntax()
	strs := StringSyntax{
		Delims: []StringDelim{{Open: "'", Close: "'", Escapes: true}},
		Within: BlockDelim{Open: "<script", Close: "</script>"},
	}
	const src = "<i/><script>x = 'y"
	got := Literals(src, comments, strs)
	if got.Complete || !got.ToEOF || got.Cut != strings.Index(src, ">x") {
		t.Errorf("Complete, Cut, ToEOF = %v, %d, %v", got.Complete, got.Cut, got.ToEOF)
	}
}
