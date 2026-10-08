package governance

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/testfile"
)

func walkAllFiles(provider source.Provider, policy testfile.Classifier) []pass.FileChange {
	var files []pass.FileChange
	provider.Walk(func(path, ext string) bool {
		files = append(files, pass.FileChange{
			Path:      path,
			Extension: ext,
			IsTest:    policy.IsTest(path),
		})
		return true
	})
	return files
}

func changesToFileChanges(
	changes []vcs.Change, policy testfile.Classifier,
) []pass.FileChange {
	files := make([]pass.FileChange, len(changes))
	for idx, change := range changes {
		files[idx] = pass.FileChange{
			Path:      change.Path,
			Extension: change.Extension,
			IsTest:    policy.IsTest(change.Path),
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
	for idx := range files {
		src, ok := provider.Read(files[idx].Path)
		if !ok {
			continue
		}
		files[idx].Generated = memoAtlas(files[idx].Path, atlasDirs, provider) ||
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

func changedSources(files []pass.FileChange) []syntax.Source {
	sources := make([]syntax.Source, len(files))
	for idx, file := range files {
		sources[idx] = syntax.Source{Path: file.Path, IsTest: file.IsTest}
	}
	return sources
}

func configFinding(err error) finding.Finding {
	return finding.NewReporter("config", finding.SeverityError, finding.CatProject).At(
		".pulsarules.yaml", 0, err.Error(), "fix the test globs",
	)
}

func (sess *Session) stagedRenames() []pass.Rename {
	const renameProbeScore = 1
	// Low so a partial rename still reaches move-purity, which judges it against its own
	// minimum instead of git's detector dropping it first.
	renames, err := sess.repo.StagedRenames(renameProbeScore)
	if err != nil {
		sess.logger.Warn("staged renames unavailable", slog.Any("error", err))
		return nil
	}
	return toCoreRenames(renames)
}
