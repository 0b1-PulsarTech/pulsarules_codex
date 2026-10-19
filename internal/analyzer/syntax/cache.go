package syntax

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const notSummarised = "not summarised"

type Unavailable struct {
	Language language.ID
	Files    int
	Reason   string
}

type Cache struct {
	once        sync.Once
	build       func()
	files       map[string]*File
	unavailable []Unavailable
}

type Corpus struct {
	Root    string
	Sources []Source
	Read    func(string) ([]byte, bool)
}

func NewCache(ctx context.Context, providers []Provider, corpus Corpus) *Cache {
	cache := &Cache{files: map[string]*File{}}
	// A provider applies its own timeout inside Summarize, because the
	// context outlives Load.
	cache.build = func() { cache.run(ctx, providers, corpus) }
	return cache
}

func (cache *Cache) run(ctx context.Context, providers []Provider, corpus Corpus) {
	grouped := make([][]Source, len(providers))
	for _, source := range corpus.Sources {
		if _, ok := corpus.Read(source.Path); !ok {
			continue
		}
		ext := strings.ToLower(filepath.Ext(source.Path))
		idx := slices.IndexFunc(providers, func(provider Provider) bool {
			return provider.Handles(ext)
		})
		if idx >= 0 {
			grouped[idx] = append(grouped[idx], source)
		}
	}
	for idx, provider := range providers {
		if len(grouped[idx]) > 0 {
			narrowed := corpus
			narrowed.Sources = grouped[idx]
			cache.summarize(ctx, provider, narrowed)
		}
	}
}

func (cache *Cache) summarize(ctx context.Context, provider Provider, corpus Corpus) {
	summaries, err := provider.Summarize(ctx, corpus)
	reason := notSummarised
	if err != nil {
		reason = err.Error()
	}
	var missing []string
	for _, source := range corpus.Sources {
		file := summaries[source.Path]
		if file == nil {
			missing = append(missing, source.Path)
			continue
		}
		if file.Language == "" {
			file.Language = language.ForExtension(filepath.Ext(source.Path)).ID
		}
		if file.Language == "" {
			file.Language = provider.Language()
		}
		file.Path = source.Path
		cache.files[source.Path] = file
	}
	cache.recordMissing(provider.Language(), missing, reason)
}

func (cache *Cache) recordMissing(fallback language.ID, missing []string, reason string) {
	counts := map[language.ID]int{}
	for _, path := range missing {
		id := language.ForExtension(filepath.Ext(path)).ID
		if id == "" {
			id = fallback
		}
		counts[id]++
	}
	for _, id := range slices.Sorted(maps.Keys(counts)) {
		cache.unavailable = append(cache.unavailable, Unavailable{
			Language: id, Files: counts[id], Reason: reason,
		})
	}
}

func (cache *Cache) ensure() {
	cache.once.Do(cache.build)
}

func (cache *Cache) Get(path string) (*File, bool) {
	if cache == nil {
		return nil, false
	}
	cache.ensure()
	file, ok := cache.files[path]
	return file, ok
}

func (cache *Cache) Unavailable() []Unavailable {
	if cache == nil {
		return nil
	}
	cache.ensure()
	return slices.Clone(cache.unavailable)
}
