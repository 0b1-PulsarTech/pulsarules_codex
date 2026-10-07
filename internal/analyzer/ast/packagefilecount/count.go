package packagefilecount

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
)

type dirCount struct {
	files       int
	isEntSchema bool
}

func countByDir(
	ctx *core.AnalysisContext,
	eligible func(core.FileChange) bool,
	wanted map[string]bool,
) map[string]dirCount {
	byDir := make(map[string][]srcInfo)
	ctx.Sources.Walk(func(file string, ext string) bool {
		file = filepath.ToSlash(file)
		dir := path.Dir(file)
		if !wanted[dir] || !eligible(core.FileChange{Path: file, Extension: ext}) ||
			ctx.TestPolicy.IsTest(file) || sources.IsHiddenName(path.Base(file)) {
			return true
		}
		src, ok := ctx.Sources.Read(file)
		if !ok || lexical.IsGeneratedSource(src) {
			return true
		}
		if entry := inspectSource(file, src); !entry.skip {
			byDir[dir] = append(byDir[dir], entry)
		}
		return true
	})
	counts := make(map[string]dirCount, len(byDir))
	for dir, infos := range byDir {
		counts[dir] = summarize(infos)
	}
	return counts
}

func summarize(infos []srcInfo) dirCount {
	major := majorityPackage(infos)
	var out dirCount
	embeds := 0
	for _, entry := range infos {
		if entry.pkg != "" && entry.pkg != major {
			continue
		}
		out.files++
		if entry.ent {
			embeds++
		}
	}
	// Judged from content: ent wants one schema file per table, and a directory
	// name would misjudge a rename.
	out.isEntSchema = embeds*2 > out.files
	return out
}

func majorityPackage(infos []srcInfo) string {
	votes := make(map[string]int)
	for _, entry := range infos {
		if entry.pkg != "" {
			votes[entry.pkg]++
		}
	}
	best := ""
	for name, count := range votes {
		if count > votes[best] || count == votes[best] && name < best {
			best = name
		}
	}
	return best
}
