package core

import (
	"path/filepath"
	"slices"
	"strings"
)

// EachChangedFile calls check with the source text of every changed file
// eligible accepts, skipping a testdata fixture the same way a full Walk
// does (see walkSkipDirs): the FileSetChanged path never goes through
// Walk, so this is the one place that skip has to be repeated for the
// changed-file path every scanning analyzer shares.
//
// why: five analyzers hand-rolled this loop shape (nil-Sources guard,
// range ChangedFiles, an eligibility filter, Sources.Read, skip on a
// failed read); four of the five never skipped testdata, so a fixture
// deliberately written wrong tripped a real finding.
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
