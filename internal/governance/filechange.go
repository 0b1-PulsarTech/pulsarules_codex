package governance

import (
	"log/slog"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func walkAllFiles(provider source.Provider) []pass.FileChange {
	var files []pass.FileChange
	provider.Walk(func(path, ext string) bool {
		files = append(files, pass.FileChange{
			Path:      path,
			Extension: ext,
			IsTest:    strings.HasSuffix(path, "_test.go"),
		})
		return true
	})
	return files
}

func changesToFileChanges(changes []vcs.Change) []pass.FileChange {
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

func toCoreRenames(renames []vcs.Rename) []pass.Rename {
	out := make([]pass.Rename, len(renames))
	for idx, rename := range renames {
		out[idx] = pass.Rename{
			OldPath: rename.OldPath, NewPath: rename.NewPath, Score: rename.Score,
		}
	}
	return out
}

func markGenerated(files []pass.FileChange, provider source.Provider) {
	if provider == nil {
		return
	}
	atlasDirs := map[string]bool{}
	for i := range files {
		src, ok := provider.Read(files[i].Path)
		if !ok {
			continue
		}
		files[i].Generated = memoAtlas(files[i].Path, atlasDirs, provider) ||
			lexer.IsGeneratedSource(src)
	}
}

func populateASTCache(changedFiles []pass.FileChange, provider source.Provider) *astcache.Cache {
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

func (s *Session) stagedRenames() []pass.Rename {
	const renameProbeScore = 1
	// Low so a partial rename still reaches move-purity, which judges it against its own
	// minimum instead of git's detector dropping it first.
	renames, err := s.repo.StagedRenames(renameProbeScore)
	if err != nil {
		s.logger.Warn("staged renames unavailable", slog.Any("error", err))
		return nil
	}
	return toCoreRenames(renames)
}
