package movepurity

import (
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
)

type gitReader interface {
	HeadBlob(path string) (string, error)
	StagedBlob(path string) (string, error)
	HeadEntry(path string) (mode, id string, err error)
	StagedEntry(path string) (mode, id string, err error)
	HeadDir(dir string) ([]string, error)
	StagedDir(dir string) ([]string, error)
}

type (
	entry struct {
		mode string
		id   string
	}
	blobResult struct {
		content string
		err     error
	}
	entryResult struct {
		entry entry
		err   error
	}
	dirResult struct {
		files []string
		err   error
	}
	parseResult struct {
		file gosrc.File
		err  error
	}
	clauseResult struct {
		name      string
		isIgnored bool
	}
	snapshot struct {
		readBlob  func(path string) (string, error)
		readEntry func(path string) (mode, id string, err error)
		readDir   func(dir string) ([]string, error)
		clash     string
		blobs     map[string]blobResult
		entries   map[string]entryResult
		dirs      map[string]dirResult
		parsed    map[string]parseResult
		clauses   map[string]clauseResult
		modules   map[string]gosrc.Module
		names     map[string]string
	}
)

func newHeadSnapshot(reader gitReader) *snapshot {
	return newSnapshot(reader.HeadBlob, reader.HeadEntry, reader.HeadDir, "\x00old")
}

func newStagedSnapshot(reader gitReader) *snapshot {
	return newSnapshot(reader.StagedBlob, reader.StagedEntry, reader.StagedDir, "\x00new")
}

func newSnapshot(
	readBlob func(string) (string, error),
	readEntry func(string) (string, string, error),
	readDir func(string) ([]string, error),
	clash string,
) *snapshot {
	return &snapshot{
		readBlob:  readBlob,
		readEntry: readEntry,
		readDir:   readDir,
		clash:     clash,
		blobs:     map[string]blobResult{},
		entries:   map[string]entryResult{},
		dirs:      map[string]dirResult{},
		parsed:    map[string]parseResult{},
		clauses:   map[string]clauseResult{},
		modules:   map[string]gosrc.Module{},
		names:     map[string]string{},
	}
}

func cached[Key comparable, Value any](cache map[Key]Value, key Key, load func() Value) Value {
	if hit, ok := cache[key]; ok {
		return hit
	}
	value := load()
	cache[key] = value
	return value
}

func (snap *snapshot) blob(file string) (string, error) {
	got := cached(snap.blobs, file, func() blobResult {
		content, err := snap.readBlob(file)
		return blobResult{content: content, err: err}
	})
	return got.content, got.err
}

func (snap *snapshot) entryOf(file string) (entry, error) {
	got := cached(snap.entries, file, func() entryResult {
		mode, id, err := snap.readEntry(file)
		return entryResult{entry: entry{mode: mode, id: id}, err: err}
	})
	return got.entry, got.err
}

func (snap *snapshot) dir(dir string) ([]string, error) {
	got := cached(snap.dirs, dir, func() dirResult {
		files, err := snap.readDir(dir)
		return dirResult{files: files, err: err}
	})
	return got.files, got.err
}

func (snap *snapshot) parse(file string) (gosrc.File, error) {
	got := cached(snap.parsed, file, func() parseResult {
		src, err := snap.blob(file)
		if err != nil {
			return parseResult{err: err}
		}
		parsed, parseErr := gosrc.Parse(src)
		return parseResult{file: parsed, err: parseErr}
	})
	return got.file, got.err
}

func (snap *snapshot) clause(file string) (name string, isIgnored bool) {
	got := cached(snap.clauses, file, func() clauseResult {
		src, err := snap.blob(file)
		if err != nil {
			return clauseResult{}
		}
		clause, ignored := gosrc.PackageClause(src)
		return clauseResult{name: clause, isIgnored: ignored}
	})
	return got.name, got.isIgnored
}

func (snap *snapshot) module(dir string) gosrc.Module {
	return cached(snap.modules, dir, func() gosrc.Module {
		return gosrc.WalkModule(dir, snap.blob)
	})
}

func (snap *snapshot) packageName(dir string) string {
	return cached(snap.names, dir, func() string {
		files, err := snap.dir(dir)
		if err != nil {
			return ""
		}
		for _, file := range slices.Sorted(slices.Values(files)) {
			if !isProductionGo(file) {
				continue
			}
			if name, isIgnored := snap.clause(file); !isIgnored && name != "" {
				return name
			}
		}
		return ""
	})
}

func (snap *snapshot) addDecls(file, pkg string, set map[string]bool) {
	if parsed, err := snap.parse(file); err == nil && parsed.Pkg == pkg {
		maps.Copy(set, parsed.TopLevel)
	}
}

func (snap *snapshot) addAnyDecls(file string, set map[string]bool) {
	if parsed, err := snap.parse(file); err == nil {
		maps.Copy(set, parsed.TopLevel)
	}
}
