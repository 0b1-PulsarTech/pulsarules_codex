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
	result[Value any] struct {
		value Value
		err   error
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
		blobs     map[string]result[string]
		entries   map[string]result[entry]
		dirs      map[string]result[[]string]
		parsed    map[string]result[gosrc.File]
		clauses   map[string]clauseResult
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
		blobs:     map[string]result[string]{},
		entries:   map[string]result[entry]{},
		dirs:      map[string]result[[]string]{},
		parsed:    map[string]result[gosrc.File]{},
		clauses:   map[string]clauseResult{},
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
	got := cached(snap.blobs, file, func() result[string] {
		content, err := snap.readBlob(file)
		return result[string]{value: content, err: err}
	})
	return got.value, got.err
}

func (snap *snapshot) entryOf(file string) (entry, error) {
	got := cached(snap.entries, file, func() result[entry] {
		mode, id, err := snap.readEntry(file)
		return result[entry]{value: entry{mode: mode, id: id}, err: err}
	})
	return got.value, got.err
}

func (snap *snapshot) dir(dir string) ([]string, error) {
	got := cached(snap.dirs, dir, func() result[[]string] {
		files, err := snap.readDir(dir)
		return result[[]string]{value: files, err: err}
	})
	return got.value, got.err
}

func (snap *snapshot) parse(file string) (gosrc.File, error) {
	got := cached(snap.parsed, file, func() result[gosrc.File] {
		src, err := snap.blob(file)
		if err != nil {
			return result[gosrc.File]{err: err}
		}
		parsed, parseErr := gosrc.Parse(src)
		return result[gosrc.File]{value: parsed, err: parseErr}
	})
	return got.value, got.err
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
	return gosrc.WalkModule(dir, snap.blob)
}

func (snap *snapshot) packageName(dir string) string {
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
