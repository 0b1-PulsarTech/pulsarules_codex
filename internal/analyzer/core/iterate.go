package core

import (
	"path/filepath"
	"slices"
	"strings"
)

// EachChangedFile calls check with the source of every changed file
// eligible accepts, skipping testdata fixtures the way Walk does.
//
// why: five analyzers hand-rolled this loop, and four of the five never
// skipped testdata, so a fixture deliberately written wrong tripped a finding.
func EachChangedFile(
	ctx *AnalysisContext,
	eligible func(fc FileChange) bool,
	check func(fc FileChange, src []byte) []Finding,
) []Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []Finding
	for _, fc := range ctx.ChangedFiles {
		if !eligible(fc) || isTestdataFixture(fc.Path) {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, check(fc, src)...)
	}
	return findings
}

// isTestdataFixture reports whether path runs through a "testdata"
// directory, the go tool's own convention for fixtures a build ignores -
// they are deliberately non-conforming and must never surface as findings.
func isTestdataFixture(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "testdata")
}
