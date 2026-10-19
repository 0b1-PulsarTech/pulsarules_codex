package syntax

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const notSummarised = "not summarised"

type Unavailable struct {
	Language Language
	Files    int
	Reason   string
}

type Cache struct {
	once        sync.Once
	ctx         context.Context //nolint:containedctx // the lazy build outlives NewCache
	providers   []Provider
	corpus      Corpus
	files       map[string]*File
	unavailable []Unavailable
}

type Corpus struct {
	Root    string
	Sources []Source
	Read    func(string) ([]byte, bool)
}

func NewCache(ctx context.Context, providers []Provider, corpus Corpus) *Cache {
	// Providers time-box themselves inside Summarize: ctx outlives Load.
	return &Cache{
		ctx: ctx, providers: providers, corpus: corpus, files: map[string]*File{},
	}
}

func (cache *Cache) run() {
	ctx, providers, corpus := cache.ctx, cache.providers, cache.corpus
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
			file.Language = LanguageForExtension(filepath.Ext(source.Path))
		}
		if file.Language == "" {
			file.Language = provider.Language()
		}
		file.Path = source.Path
		cache.files[source.Path] = file
	}
	cache.recordMissing(provider.Language(), missing, reason)
}

func (cache *Cache) recordMissing(fallback Language, missing []string, reason string) {
	counts := map[Language]int{}
	for _, path := range missing {
		language := LanguageForExtension(filepath.Ext(path))
		if language == "" {
			language = fallback
		}
		counts[language]++
	}
	for _, language := range slices.Sorted(maps.Keys(counts)) {
		cache.unavailable = append(cache.unavailable, Unavailable{
			Language: language, Files: counts[language], Reason: reason,
		})
	}
}

func (cache *Cache) ensure() {
	cache.once.Do(cache.run)
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
