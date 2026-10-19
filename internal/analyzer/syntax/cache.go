package syntax

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
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
	Module  func() string
}

func NewCache(providers []Provider, corpus Corpus) *Cache {
	cache := &Cache{files: map[string]*File{}}
	// No caller holds a context, so the lazy build starts from Background.
	cache.build = func() { cache.run(context.Background(), providers, corpus) }
	return cache
}

func (cache *Cache) run(ctx context.Context, providers []Provider, corpus Corpus) {
	grouped := make([][]Source, len(providers))
	for _, source := range corpus.Sources {
		content, ok := corpus.Read(source.Path)
		if !ok {
			continue
		}
		source.Content = content
		id := language.ForExtension(filepath.Ext(source.Path)).ID
		idx := slices.IndexFunc(providers, func(provider Provider) bool {
			return slices.Contains(provider.Languages(), id)
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
		file.Path = source.Path
		cache.files[source.Path] = file
	}
	cache.recordMissing(missing, reason)
}

func (cache *Cache) recordMissing(missing []string, reason string) {
	counts := map[language.ID]int{}
	for _, path := range missing {
		counts[language.ForExtension(filepath.Ext(path)).ID]++
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
