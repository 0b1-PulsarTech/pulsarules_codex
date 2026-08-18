package textmarkers

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// scanned is the file set both analyzers read.
var scanned = map[string]bool{".go": true, ".md": true}

// eachMarkedFile calls check for every eligible changed file's source text.
//
// why: testdata is skipped here as well as in core.walkSkipDirs, because the
// changed-file path has no such filter and a fixture is deliberately dirty -
// at error severity it would block the very commit that fixes it.
func eachMarkedFile(
	ctx *core.AnalysisContext,
	check func(core.FileChange, string) []core.Finding,
) []core.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		if !scanned[strings.ToLower(fc.Extension)] || isFixture(fc.Path) {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, check(fc, string(src))...)
	}
	return findings
}

func isFixture(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "testdata")
}
