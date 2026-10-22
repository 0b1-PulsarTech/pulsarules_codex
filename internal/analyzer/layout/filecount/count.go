package filecount

import (
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

type dirCount struct {
	files       int
	isEntSchema bool
}

func countByDir(
	ctx *pass.Context,
	eligible func(pass.FileChange) bool,
	wanted map[string]bool,
) map[string]dirCount {
	byDir := make(map[string][]srcInfo)
	index := ctx.GoPackages()
	ctx.Sources.Walk(func(file string, ext string) bool {
		file = filepath.ToSlash(file)
		dir := path.Dir(file)
		if !wanted[dir] || !eligible(pass.FileChange{Path: file, Extension: ext}) ||
			ctx.TestClassifier.IsTest(file) || gomod.IsHiddenName(path.Base(file)) {
			return true
		}
		src, ok := ctx.Sources.Read(file)
		if !ok || lexer.IsGeneratedSource(src) {
			return true
		}
		parsed, indexed := index.File(file)
		// The index drops build-ignored .go files, which Go never compiles.
		if ext == ".go" && !indexed {
			return true
		}
		byDir[dir] = append(byDir[dir], inspectSource(parsed))
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
