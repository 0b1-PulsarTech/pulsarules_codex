package analysis

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func walkAllFiles(sources core.SourceProvider) []core.FileChange {
	var files []core.FileChange
	sources.Walk(func(path, ext string) bool {
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

func populateASTCache(
	changedFiles []core.FileChange,
	sources core.SourceProvider,
) *astcache.Cache {
	if len(changedFiles) == 0 {
		return nil
	}
	cache := astcache.New()
	for _, fc := range changedFiles {
		if fc.Extension == ".go" && !fc.IsTest {
			src, ok := sources.Read(fc.Path)
			if !ok {
				continue
			}
			if _, err := cache.Parse(fc.Path, src); err != nil {
				continue
			}
		}
	}
	return cache
}
