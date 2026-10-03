package analysis

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func walkAllFiles(provider sources.SourceProvider) []core.FileChange {
	var files []core.FileChange
	provider.Walk(func(path, ext string) bool {
		files = append(files, core.FileChange{
			Path:      path,
			Extension: ext,
			IsTest:    strings.HasSuffix(path, "_test.go"),
		})
		return true
	})
	return files
}

func changesToFileChanges(changes []vcs.Change) []core.FileChange {
	files := make([]core.FileChange, len(changes))
	for idx, change := range changes {
		files[idx] = core.FileChange{
			Path:      change.Path,
			Extension: change.Extension,
			IsTest:    change.IsTest,
			Staged:    change.Staged,
		}
	}
	return files
}

func toCoreRenames(renames []vcs.Rename) []core.Rename {
	out := make([]core.Rename, len(renames))
	for idx, rename := range renames {
		out[idx] = core.Rename{
			OldPath: rename.OldPath, NewPath: rename.NewPath, Score: rename.Score,
		}
	}
	return out
}

func markGenerated(files []core.FileChange, provider sources.SourceProvider) {
	if provider == nil {
		return
	}
	for i := range files {
		src, ok := provider.Read(files[i].Path)
		if !ok {
			continue
		}
		files[i].Generated = lexical.IsGeneratedSource(src)
	}
}

func populateASTCache(
	changedFiles []core.FileChange,
	provider sources.SourceProvider,
) *astcache.Cache {
	if len(changedFiles) == 0 {
		return nil
	}
	cache := astcache.New()
	// A _test.go file is parsed too: time-discipline polices clocks in tests.
	for _, fc := range changedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := provider.Read(fc.Path)
		if !ok {
			continue
		}
		if _, err := cache.Parse(fc.Path, src); err != nil {
			continue
		}
	}
	return cache
}
