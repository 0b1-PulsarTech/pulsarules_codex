package analysis

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

// walkAllFiles enumerates every source file in sources and converts it to
// core.FileChange, so a whole-tree run feeds analyzers the same shape a
// changed-files run does. Staged is always false: nothing here came from git
// status.
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

// changesToFileChanges maps vcs.Change values onto the analyzer contract's
// FileChange, keeping the vcs package free of any dependency on core.
func changesToFileChanges(changes []vcs.Change) []core.FileChange {
	files := make([]core.FileChange, len(changes))
	for i, c := range changes {
		files[i] = core.FileChange{
			Path: c.Path, Extension: c.Extension, IsTest: c.IsTest, Staged: c.Staged,
		}
	}
	return files
}

// toCoreRenames maps vcs.Rename values onto the analyzer contract's Rename,
// keeping the vcs package free of any dependency on core.
func toCoreRenames(renames []vcs.Rename) []core.Rename {
	out := make([]core.Rename, len(renames))
	for i, r := range renames {
		out[i] = core.Rename{OldPath: r.OldPath, NewPath: r.NewPath, Score: r.Score}
	}
	return out
}

// markGenerated flags every file carrying the Go generated-code marker, in
// place, so the suppression pass can tell machine-written source from source a
// human can actually fix. Reads go through the SourceProvider's cache, so the
// bytes it pulls here are the same ones the analyzers read afterwards.
func markGenerated(files []core.FileChange, sources core.SourceProvider) {
	if sources == nil {
		return
	}
	for i := range files {
		src, ok := sources.Read(files[i].Path)
		if !ok {
			continue
		}
		files[i].Generated = core.IsGeneratedSource(src)
	}
}

// populateASTCache pre-parses all changed Go files into an AST cache so
// analyzers can call Get(path) without file I/O or parsing.
func populateASTCache(changedFiles []core.FileChange, sources core.SourceProvider) *astcache.Cache {
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
