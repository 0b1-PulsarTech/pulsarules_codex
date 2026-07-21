package discovery

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func WalkAllFiles(sources pass.SourceProvider) []pass.FileChange {
	var files []pass.FileChange
	sources.Walk(func(path, ext string) bool {
		files = append(files, pass.FileChange{
			Path:      path,
			Extension: ext,
			IsTest:    strings.HasSuffix(path, "_test.go"),
		})
		return true
	})
	return files
}

func ChangesToFileChanges(changes []vcs.Change) []pass.FileChange {
	files := make([]pass.FileChange, len(changes))
	for idx, change := range changes {
		files[idx] = pass.FileChange{
			Path:      change.Path,
			Extension: change.Extension,
			IsTest:    change.IsTest,
			Staged:    change.Staged,
		}
	}
	return files
}

func ToCoreRenames(renames []vcs.Rename) []pass.Rename {
	out := make([]pass.Rename, len(renames))
	for idx, rename := range renames {
		out[idx] = pass.Rename{
			OldPath: rename.OldPath, NewPath: rename.NewPath, Score: rename.Score,
		}
	}
	return out
}
