package contextdiscipline

import (
	"go/parser"
	"go/token"
	"testing"
)

func processLifetimeMarkerCases() []checkFileTestCase {
	cases := processLifetimeMarkerSuppressCases()
	return append(cases, processLifetimeMarkerBoundaryCases()...)
}

func processLifetimeMarkerSuppressCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "marker with a reason on the same line suppresses the finding",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor { //pulsarules:process-lifetime lives with the process\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "marker with a reason on the line above suppresses the finding",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\t//pulsarules:process-lifetime lives with the process\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "marker with no reason on the line above reports the reason finding instead",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\t//pulsarules:process-lifetime\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantSubstr: "has no stated reason",
			wantLine:   5,
		},
		{
			name: "marker with no reason on the same line reports the reason finding instead",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor { //pulsarules:process-lifetime\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantSubstr: "has no stated reason",
			wantLine:   4,
		},
	}
}

func processLifetimeMarkerBoundaryCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a marker with no word boundary is not a marker at all",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor { //pulsarules:process-lifetimeX\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
			wantLine:   4,
		},
		{
			name: "a hyphen-suffixed near-marker is not a marker at all",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor { //pulsarules:process-lifetime-forever\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
			wantLine:   4,
		},
		{
			name: "a marker trailing an unrelated preceding statement does not suppress the loop",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tmark() //pulsarules:process-lifetime unrelated note\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n" +
				"func mark() {}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
			wantLine:   5,
		},
	}
}

func TestMatchProcessLifetimeMarker(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		text       string
		wantReason string
		wantOK     bool
	}{
		{
			name:       "bare marker has no reason",
			text:       "pulsarules:process-lifetime",
			wantReason: "",
			wantOK:     true,
		},
		{
			name:       "marker with a reason after a space",
			text:       "pulsarules:process-lifetime lives with the process",
			wantReason: "lives with the process",
			wantOK:     true,
		},
		{
			name:   "a suffix with no word boundary is not the marker",
			text:   "pulsarules:process-lifetimeX",
			wantOK: false,
		},
		{
			name:   "a hyphenated suffix is not the marker",
			text:   "pulsarules:process-lifetime-forever",
			wantOK: false,
		},
		{
			name:   "unrelated text is not the marker",
			text:   "TODO: fix this later",
			wantOK: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reason, ok := matchProcessLifetimeMarker(testCase.text)
			if ok != testCase.wantOK {
				t.Fatalf("ok = %v, want %v", ok, testCase.wantOK)
			}
			if ok && reason != testCase.wantReason {
				t.Fatalf("reason = %q, want %q", reason, testCase.wantReason)
			}
		})
	}
}

func TestLinesWithCode(t *testing.T) {
	t.Parallel()

	source := "package foo\n" +
		"func run() {\n" +
		"\tdoWork()\n" +
		"}\n" +
		"func doWork() {}\n"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	lines := linesWithCode(fset, file)
	if !lines[3] {
		t.Fatalf("lines = %v, want line 3 (doWork() call) marked", lines)
	}
}
