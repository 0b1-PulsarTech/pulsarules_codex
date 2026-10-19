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

func goDocSiteCases() []docSiteCase {
	return []docSiteCase{
		{
			name: "multi-line block deprecated is exempt", source: plainSource(),
			body: "package x\n\n/*\nDeprecated: a\n*/\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "multi-line block over the cap is prose", source: plainSource(),
			body: "package x\n\n/*\nDeprecated: a\nb\nc\nd\n*/\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "at-deprecated is prose in Go", source: plainSource(),
			body: "package x\n\n// @deprecated use other.\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "deprecated plus a directive stays within the cap", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n// c\n//go:noinline\nfunc run() {}\n",
			want: []syntax.DocSite{{
				Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true,
				MixedDirective: true,
			}},
		},
		{
			name: "export directive is exempt", source: plainSource(),
			body: "package x\n\n//export run\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "ignore and eslint lines are prose in Go", source: plainSource(),
			body: "package x\n\n// ignore: unused\n// eslint-foo\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "empty star block is prose in Go", source: plainSource(),
			body: "package x\n\n/** */\nfunc run() {}\n",
			want: []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "closer-only comments are blank", source: plainSource(),
			body: "package x\n\n//*/\n// */\nfunc run() {}\n",
			want: []syntax.DocSite{
				{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true},
			},
		},
		{
			name: "simplification after a directive is exempt", source: plainSource(),
			body: "package x\n\n//go:noinline\n// simplification: no cache.\nfunc run() {}\n",
			want: []syntax.DocSite{{
				Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true,
				MixedDirective: true,
			}},
		},
	}
}

func goToolchainTextCases() []docSiteCase {
	exempt := []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true}}
	prose := []syntax.DocSite{{Symbol: "run", Kind: syntax.DocFunction, Line: 3}}
	const tail = "func run() {}\n"
	return []docSiteCase{
		{
			name: "generic tool directive before deprecated", source: plainSource(),
			body: "package x\n\n//revive:disable:exported\n// Deprecated: use B.\n" + tail,
			want: []syntax.DocSite{{
				Symbol: "run", Kind: syntax.DocFunction, Line: 3, Exempt: true,
			}},
		},
		{
			name: "kubebuilder directive before deprecated", source: plainSource(),
			body: "package x\n\n//kubebuilder:object:root=true\n// Deprecated: use C.\n" + tail,
			want: exempt,
		},
		{
			name: "lowercase generic directive before deprecated", source: plainSource(),
			body: "package x\n\n//deprecated:x\n// Deprecated: y\n" + tail,
			want: exempt,
		},
		{
			name: "generic directive stays within the cap", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n// c\n//todo:remove\n" + tail,
			want: exempt,
		},
		{
			name: "sqlc directive stays within the cap", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n// c\n//sqlc:skip\n" + tail,
			want: exempt,
		},
		{
			name: "export with a trailing space stays within the cap", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// b\n//export \n// d\n" + tail,
			want: exempt,
		},
		{
			name: "closer-only comment inside deprecated is text", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n//*/\n// more.\n" + tail,
			want: exempt,
		},
		{
			name: "spaced closer-only comment inside deprecated is text", source: plainSource(),
			body: "package x\n\n// Deprecated: a\n// */\n// more.\n" + tail,
			want: exempt,
		},
		{
			name: "slash line inside a deprecated block is text", source: plainSource(),
			body: "package x\n\n/*\nDeprecated: a\n//\nmore.\n*/\n" + tail,
			want: exempt,
		},
		{
			name: "opener inside a deprecated block is text", source: plainSource(),
			body: "package x\n\n/*\nDeprecated: a\n/*\nmore.\n*/\n" + tail,
			want: exempt,
		},
		{
			name: "directive inside a block is text", source: plainSource(),
			body: "package x\n\n/*\nreal prose.\n//go:noinline\n*/\n" + tail,
			want: prose,
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

	for _, testCase := range slices.Concat(
		docSiteCases(), goDocSiteCases(), goToolchainTextCases(), testDocSiteCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, testCase.source, "").DocSites
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("DocSites = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
