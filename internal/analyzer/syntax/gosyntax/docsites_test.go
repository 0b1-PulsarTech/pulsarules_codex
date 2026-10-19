package gosyntax

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type docSiteCase struct {
	name   string
	source syntax.Source
	body   string
	want   []syntax.DocSite
}

func docSiteCases() []docSiteCase {
	return []docSiteCase{
		{
			name: "documented function", source: plainSource(),
			body: "package x\n\n// run does it.\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "undocumented function yields nothing", source: plainSource(),
			body: "package x\n\nfunc run() {}\n",
		},
		{
			name:   "directive only is exempt",
			source: plainSource(),
			body:   "package x\n\n//nolint:foo\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "mixed directive and prose", source: plainSource(),
			body: "package x\n\n// run does it.\n//nolint:foo\nfunc run() {}\n",
			want: []syntax.DocSite{{
				Symbol: "run", Kind: syntax.DocFunction, Line: 3, MixedDirective: true,
			}},
		},
		{
			name:   "deprecated only is exempt",
			source: plainSource(),
			body:   "package x\n\n// Deprecated: use other.\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "deprecated at three lines is exempt", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n// c\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "deprecated at four lines is prose", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n// c\n// d\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "deprecated after prose is prose", source: plainSource(),
			body: "package x\n\n// run does.\n// Deprecated: a\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "deprecated with a second paragraph is prose", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n//\n// b\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "simplification is exempt", source: plainSource(),
			body: "package x\n\n// simplification: no cache.\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
	}
}

func testDocSiteCases() []docSiteCase {
	return []docSiteCase{
		{
			name: "test function in a test file is exempt",
			source: syntax.Source{
				Path: "pkg_test.go", IsTest: true,
			},
			body: "package x\n\nimport \"testing\"\n\n" +
				"// TestRun checks it.\nfunc TestRun(t *testing.T) {}\n",
			want: []syntax.DocSite{
				{Symbol: "TestRun", Kind: syntax.DocFunction, Line: 5, Exempt: true},
			},
		},
		{
			name:   "test-named function outside a test file is not exempt",
			source: plainSource(),
			body: "package x\n\nimport \"testing\"\n\n" +
				"// TestRun checks it.\nfunc TestRun(t *testing.T) {}\n",
			want: []syntax.DocSite{{Symbol: "TestRun", Kind: syntax.DocFunction, Line: 5}},
		},
		{
			name: "non type declaration is ignored", source: plainSource(),
			body: "package x\n\n// limit caps it.\nconst limit = 1\n",
		},
	}
}

func TestDocSites(t *testing.T) {
	t.Parallel()

	for _, testCase := range append(docSiteCases(), testDocSiteCases()...) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, testCase.source, "").DocSites
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("DocSites = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
