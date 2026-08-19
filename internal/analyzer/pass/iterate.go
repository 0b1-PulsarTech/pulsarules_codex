package pass

import (
	"path/filepath"
	"slices"
	"strings"
)

func EachChangedFile(
	ctx *Context,
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

func isTestdataFixture(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "testdata")
}
