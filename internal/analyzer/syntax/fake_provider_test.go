package syntax

import (
	"context"
	"slices"
)

type fakeProvider struct {
	language  Language
	exts      []string
	err       error
	omit      []string
	nilFiles  []string
	extra     []string
	noResults bool
	blankLang bool
	pathFor   func(string) string
	calls     [][]Source
	gotRoot   string
	gotCtx    context.Context
	gotRead   func(string) ([]byte, bool)
}

func (fake *fakeProvider) Language() Language { return fake.language }

func (fake *fakeProvider) Handles(ext string) bool { return slices.Contains(fake.exts, ext) }

func (fake *fakeProvider) Summarize(
	ctx context.Context, corpus Corpus,
) (map[string]*File, error) {
	fake.calls = append(fake.calls, slices.Clone(corpus.Sources))
	fake.gotRoot, fake.gotRead, fake.gotCtx = corpus.Root, corpus.Read, ctx
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
