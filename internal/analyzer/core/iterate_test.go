package core

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
)

type fakeProvider struct {
	contents map[string][]byte
}

func (p fakeProvider) Read(path string) ([]byte, bool) {
	src, ok := p.contents[path]
	return src, ok
}

func (p fakeProvider) Walk(func(path string, ext string) bool) {}

var _ sources.Provider = fakeProvider{}

type eachChangedFileTestCase struct {
	name        string
	ctx         *AnalysisContext
	wantFiles   []string
	wantNilCall bool
}

func isGoFile(fc FileChange) bool { return fc.Extension == ".go" }

func eachChangedFileTestCases() []eachChangedFileTestCase {
	return []eachChangedFileTestCase{
		{
			name: "nil Sources yields nothing",
			ctx: &AnalysisContext{
				ChangedFiles: []FileChange{{Path: "a.go", Extension: ".go"}},
			},
			wantNilCall: true,
		},
		{
			name: "testdata fixture is skipped even when eligible",
			ctx: &AnalysisContext{
				Sources: fakeProvider{contents: map[string][]byte{
					"pkg/testdata/fixture.go": []byte("package fixture"),
					"pkg/real.go":             []byte("package pkg"),
				}},
				ChangedFiles: []FileChange{
					{Path: "pkg/testdata/fixture.go", Extension: ".go"},
					{Path: "pkg/real.go", Extension: ".go"},
				},
			},
			wantFiles: []string{"pkg/real.go"},
		},
		{
			name: "ineligible extension is skipped",
			ctx: &AnalysisContext{
				Sources: fakeProvider{contents: map[string][]byte{
					"a.go": []byte("package a"),
					"b.md": []byte("# b"),
				}},
				ChangedFiles: []FileChange{
					{Path: "a.go", Extension: ".go"},
					{Path: "b.md", Extension: ".md"},
				},
			},
			wantFiles: []string{"a.go"},
		},
		{
			name: "a changed file the provider cannot read is skipped",
			ctx: &AnalysisContext{
				Sources: fakeProvider{contents: map[string][]byte{}},
				ChangedFiles: []FileChange{
					{Path: "missing.go", Extension: ".go"},
				},
			},
			wantNilCall: true,
		},
	}
}

func TestEachChangedFile(t *testing.T) {
	t.Parallel()

	echoPath := func(fc FileChange, src []byte) []Finding {
		return []Finding{{File: fc.Path, Message: string(src)}}
	}

	for _, testCase := range eachChangedFileTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := EachChangedFile(testCase.ctx, isGoFile, echoPath)
			if testCase.wantNilCall {
				if got != nil {
					t.Fatalf("EachChangedFile() = %+v, want nil", got)
				}
				return
			}
			if len(got) != len(testCase.wantFiles) {
				t.Fatalf(
					"EachChangedFile() returned %d findings, want %d: %+v",
					len(got), len(testCase.wantFiles), got,
				)
			}
			for idx, path := range testCase.wantFiles {
				if got[idx].File != path {
					t.Fatalf("finding %d file = %q, want %q", idx, got[idx].File, path)
				}
			}
		})
	}
}
