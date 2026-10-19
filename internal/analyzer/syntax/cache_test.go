package syntax

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type routingCase struct {
	name          string
	paths         []string
	unreadable    string
	noProviders   bool
	wantGoCalls   [][]string
	wantDartCalls [][]string
	wantGet       []string
	wantMissing   []string
}

func routingCases() []routingCase {
	return []routingCase{
		{
			name:          "routes by extension with one call per provider",
			paths:         []string{"a.go", "lib/b.dart", "c.GO", "d.go"},
			wantGoCalls:   [][]string{{"a.go", "c.GO", "d.go"}},
			wantDartCalls: [][]string{{"lib/b.dart"}},
			wantGet:       []string{"a.go", "lib/b.dart", "c.GO", "d.go"},
		},
		{
			name:        "unhandled extensions ignored",
			paths:       []string{"a.go", "notes.md", "Makefile"},
			wantGoCalls: [][]string{{"a.go"}},
			wantGet:     []string{"a.go"},
			wantMissing: []string{"notes.md", "Makefile"},
		},
		{
			name:        "unreadable path never reaches a provider",
			paths:       []string{"a.go", "gone.go"},
			unreadable:  "gone.go",
			wantGoCalls: [][]string{{"a.go"}},
			wantGet:     []string{"a.go"},
			wantMissing: []string{"gone.go"},
		},
		{
			name:        "first handling provider wins",
			paths:       []string{"a.go"},
			wantGoCalls: [][]string{{"a.go"}},
			wantGet:     []string{"a.go"},
		},
		{name: "no paths calls nothing", wantMissing: []string{"a.go"}},
		{
			name:        "empty provider list produces nothing",
			paths:       []string{"a.go"},
			noProviders: true,
			wantMissing: []string{"a.go"},
		},
	}
}

func TestNewCache_Routing(t *testing.T) {
	t.Parallel()

	for _, testCase := range routingCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			goProvider := &fakeProvider{language: language.Go, languages: goLanguages()}
			dartProvider := &fakeProvider{
				language:  language.Dart,
				languages: []language.ID{language.Dart},
			}
			overlap := &fakeProvider{language: language.JavaScript, languages: goLanguages()}
			providers := []Provider{goProvider, dartProvider, overlap}
			if testCase.noProviders {
				providers = nil
			}
			read := func(path string) ([]byte, bool) { return nil, path != testCase.unreadable }
			cache := NewCache(
				providers,
				Corpus{Root: "root", Sources: toSources(testCase.paths...), Read: read},
			)

			for _, path := range testCase.wantGet {
				if file, ok := cache.Get(path); !ok || file.Path != path {
					t.Fatalf("Get(%q) = %v, %v", path, file, ok)
				}
			}
			for _, path := range testCase.wantMissing {
				if _, ok := cache.Get(path); ok {
					t.Fatalf("Get(%q) found a file", path)
				}
			}
			if got := cache.Unavailable(); len(got) != 0 {
				t.Fatalf("Unavailable() = %v, want none", got)
			}
			assertCalls(t, "go", goProvider.calls, testCase.wantGoCalls)
			assertCalls(t, "dart", dartProvider.calls, testCase.wantDartCalls)
			assertCalls(t, "overlap", overlap.calls, nil)
		})
	}
}

func assertCalls(t *testing.T, name string, got [][]Source, want [][]string) {
	t.Helper()

	if !slices.EqualFunc(callPaths(got), want, slices.Equal) {
		t.Fatalf("%s calls = %v, want %v", name, callPaths(got), want)
	}
}

type resultCase struct {
	name            string
	provider        *fakeProvider
	paths           []string
	wantGet         []string
	wantMissing     []string
	wantUnavailable []Unavailable
	wantLanguage    map[string]language.ID
}

func resultCases() []resultCase {
	return slices.Concat(failureCases(), shapeCases(), splitCases())
}

func failureCases() []resultCase {
	failure := errors.New("parser down")
	return []resultCase{
		{
			name: "total failure keeps no files",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				err:       failure,
				noResults: true,
			},
			paths:       []string{"a.go", "b.go"},
			wantMissing: []string{"a.go", "b.go"},
			wantUnavailable: []Unavailable{
				{Language: language.Go, Files: 2, Reason: "parser down"},
			},
		},
		{
			name: "partial results survive an error",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				err:       failure,
				omit:      []string{"b.go"},
			},
			paths:       []string{"a.go", "b.go"},
			wantGet:     []string{"a.go"},
			wantMissing: []string{"b.go"},
			wantUnavailable: []Unavailable{
				{Language: language.Go, Files: 1, Reason: "parser down"},
			},
		},
		{
			name: "omitted path is not summarised",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				omit:      []string{"b.go"},
			},
			paths:       []string{"a.go", "b.go"},
			wantGet:     []string{"a.go"},
			wantMissing: []string{"b.go"},
			wantUnavailable: []Unavailable{
				{Language: language.Go, Files: 1, Reason: notSummarised},
			},
		},
		{
			name: "nil file counts as omitted",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				nilFiles:  []string{"a.go"},
			},
			paths:       []string{"a.go"},
			wantMissing: []string{"a.go"},
			wantUnavailable: []Unavailable{
				{Language: language.Go, Files: 1, Reason: notSummarised},
			},
		},
	}
}

func shapeCases() []resultCase {
	return []resultCase{
		{
			name: "unrequested entry is dropped",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				extra:     []string{"x.go"},
			},
			paths:       []string{"a.go"},
			wantGet:     []string{"a.go"},
			wantMissing: []string{"x.go"},
		},
		{
			name: "blank language filled from the extension",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				blankLang: true,
			},
			paths:        []string{"a.go"},
			wantGet:      []string{"a.go"},
			wantLanguage: map[string]language.ID{"a.go": language.Go},
		},
		{
			name: "absolute path from the provider is replaced",
			provider: &fakeProvider{
				language: language.Go, languages: goLanguages(),
				pathFor: func(path string) string { return "/abs/" + path },
			},
			paths: []string{"a.go"}, wantGet: []string{"a.go"},
		},
		{
			name: "empty path from the provider is replaced",
			provider: &fakeProvider{
				language:  language.Go,
				languages: goLanguages(),
				pathFor:   func(string) string { return "" },
			},
			paths: []string{"a.go"}, wantGet: []string{"a.go"},
		},
	}
}

func splitCases() []resultCase {
	failure := errors.New("parser down")
	return []resultCase{
		{
			name: "failure splits by the actual language",
			provider: &fakeProvider{
				language:  language.TypeScript,
				languages: []language.ID{language.TypeScript, language.JavaScript, language.Svelte},
				err:       failure, noResults: true,
			},
			paths: []string{"a.ts", "b.svelte", "c.js", "d.js"},
			wantUnavailable: []Unavailable{
				{Language: language.JavaScript, Files: 2, Reason: "parser down"},
				{Language: language.Svelte, Files: 1, Reason: "parser down"},
				{Language: language.TypeScript, Files: 1, Reason: "parser down"},
			},
			wantMissing: []string{"a.ts"},
		},
	}
}

func TestNewCache_Results(t *testing.T) {
	t.Parallel()

	for _, testCase := range resultCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := NewCache(
				[]Provider{testCase.provider},
				Corpus{Root: "root", Sources: toSources(testCase.paths...), Read: allRead},
			)

			for _, path := range testCase.wantGet {
				if file, ok := cache.Get(path); !ok || file.Path != path {
					t.Fatalf("Get(%q) = %v, %v", path, file, ok)
				}
			}
			for _, path := range testCase.wantMissing {
				if _, ok := cache.Get(path); ok {
					t.Fatalf("Get(%q) found a file", path)
				}
			}
			for path, want := range testCase.wantLanguage {
				if file, _ := cache.Get(path); file.Language != want {
					t.Fatalf("Get(%q).Language = %q, want %q", path, file.Language, want)
				}
			}
			if !slices.Equal(cache.Unavailable(), testCase.wantUnavailable) {
				got := cache.Unavailable()
				t.Fatalf("Unavailable() = %v, want %v", got, testCase.wantUnavailable)
			}
		})
	}
}

func TestNewCache_ProvidersReceiveContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		sources     []Source
		unreadable  string
		wantSources []Source
		wantReads   map[string]int
	}{
		{
			name: "content and flags reach the provider in order",
			sources: []Source{
				{Path: "a_test.go", IsTest: true},
				{Path: "b.go"},
			},
			wantSources: []Source{
				{Path: "a_test.go", IsTest: true, Content: []byte("src:a_test.go")},
				{Path: "b.go", Content: []byte("src:b.go")},
			},
			wantReads: map[string]int{"a_test.go": 1, "b.go": 1},
		},
		{
			name:        "unreadable source reported once by Cache",
			sources:     []Source{{Path: "a.go"}, {Path: "gone.go"}},
			unreadable:  "gone.go",
			wantSources: []Source{{Path: "a.go", Content: []byte("src:a.go")}},
			wantReads:   map[string]int{"a.go": 1, "gone.go": 1},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			provider := &fakeProvider{language: language.Go, languages: goLanguages()}
			reads := map[string]int{}
			read := func(path string) ([]byte, bool) {
				reads[path]++
				return []byte("src:" + path), path != testCase.unreadable
			}
			cache := NewCache(
				[]Provider{provider},
				Corpus{Root: "/the/root", Sources: testCase.sources, Read: read},
			)
			cache.Unavailable()

			if !reflect.DeepEqual(provider.gotSource, testCase.wantSources) {
				t.Fatalf("sources = %+v, want %+v", provider.gotSource, testCase.wantSources)
			}
			if !reflect.DeepEqual(reads, testCase.wantReads) {
				t.Fatalf("reads = %v, want %v", reads, testCase.wantReads)
			}
			if provider.gotRoot != "/the/root" {
				t.Fatalf("root = %q", provider.gotRoot)
			}
			if got := cache.Unavailable(); len(got) != 0 {
				t.Fatalf("Unavailable() = %v, want none", got)
			}
		})
	}
}

func TestNewCache_Lazy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		access func(*Cache)
	}{
		{name: "Get", access: func(cache *Cache) { cache.Get("a.go") }},
		{name: "Unavailable", access: func(cache *Cache) { cache.Unavailable() }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			provider := &fakeProvider{language: language.Go, languages: goLanguages()}
			cache := NewCache(
				[]Provider{provider},
				Corpus{Root: "root", Sources: toSources("a.go"), Read: allRead},
			)
			if len(provider.calls) != 0 {
				t.Fatalf("Summarize ran before any accessor: %v", provider.calls)
			}
			for range 3 {
				testCase.access(cache)
				cache.Get("a.go")
				cache.Unavailable()
			}
			if len(provider.calls) != 1 {
				t.Fatalf("Summarize calls = %d, want 1", len(provider.calls))
			}
		})
	}
}

func TestCache_NilReceiver(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		empty func(cache *Cache) bool
	}{
		{name: "Get", empty: func(cache *Cache) bool {
			file, ok := cache.Get("a.go")
			return file == nil && !ok
		}},
		{name: "Unavailable", empty: func(cache *Cache) bool {
			return cache.Unavailable() == nil
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if !testCase.empty(nil) {
				t.Fatalf("%s on a nil cache returned content", testCase.name)
			}
		})
	}
}
