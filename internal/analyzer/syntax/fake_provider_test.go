package syntax

import (
	"context"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type fakeProvider struct {
	language  language.ID
	languages []language.ID
	err       error
	omit      []string
	nilFiles  []string
	extra     []string
	noResults bool
	blankLang bool
	pathFor   func(string) string
	calls     [][]Source
	gotRoot   string
	gotSource []Source
	gotRead   func(string) ([]byte, bool)
}

func (fake *fakeProvider) Languages() []language.ID { return fake.languages }

func (fake *fakeProvider) Summarize(
	_ context.Context, corpus Corpus,
) (map[string]*File, error) {
	fake.calls = append(fake.calls, slices.Clone(corpus.Sources))
	fake.gotRoot, fake.gotRead = corpus.Root, corpus.Read
	fake.gotSource = slices.Clone(corpus.Sources)
	if fake.noResults {
		return nil, fake.err
	}
	out := map[string]*File{}
	for _, source := range corpus.Sources {
		switch {
		case slices.Contains(fake.omit, source.Path):
		case slices.Contains(fake.nilFiles, source.Path):
			out[source.Path] = nil
		default:
			out[source.Path] = &File{Path: source.Path, Language: fake.language}
			if fake.blankLang {
				out[source.Path].Language = ""
			}
			if fake.pathFor != nil {
				out[source.Path].Path = fake.pathFor(source.Path)
			}
		}
	}
	for _, path := range fake.extra {
		out[path] = &File{Path: path, Language: fake.language}
	}
	return out, fake.err
}

func allRead(string) ([]byte, bool) { return []byte("x"), true }

func toSources(paths ...string) []Source {
	sources := make([]Source, len(paths))
	for idx, path := range paths {
		sources[idx] = Source{Path: path}
	}
	return sources
}

func callPaths(calls [][]Source) [][]string {
	out := make([][]string, 0, len(calls))
	for _, call := range calls {
		paths := make([]string, 0, len(call))
		for _, source := range call {
			paths = append(paths, source.Path)
		}
		out = append(out, paths)
	}
	return out
}

func goLanguages() []language.ID { return []language.ID{language.Go} }
