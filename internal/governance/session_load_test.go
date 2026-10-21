package governance

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type recordingProvider struct {
	calls   int
	sources []syntax.Source
	root    string
	module  string
	read    func(string) ([]byte, bool)
}

func (recorder *recordingProvider) Languages() []language.ID {
	return []language.ID{language.Go}
}

func (recorder *recordingProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	recorder.calls++
	recorder.sources = append(recorder.sources, corpus.Sources...)
	recorder.root, recorder.read = corpus.Root, corpus.Read
	recorder.module = corpus.Module()
	files := map[string]*syntax.File{}
	for _, source := range corpus.Sources {
		files[source.Path] = &syntax.File{Path: source.Path, Language: language.Go}
	}
	return files, nil
}

func loadSession(t *testing.T, name, body string, install func(*Session)) (
	string, *pass.Context,
) {
	t.Helper()
	dir := t.TempDir()
	gitInitRepo(t, dir)
	if name != "" {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	sess := NewSession(repo, "", nil, testCatalog(t), nil)
	install(sess)
	return repo.Root(), sess.Load(sess.Discover(ScopeChanged, nil, FileSetChanged))
}

func TestSession_Load_BuildsSyntaxCache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		write      bool
		withFake   bool
		wantCache  bool
		wantParsed []string
	}{
		{name: "changed file with no providers", write: true, wantCache: true},
		{
			name: "changed file reaches the provider", write: true, withFake: true,
			wantCache: true, wantParsed: []string{"a.go"},
		},
		{name: "no changed files leaves it nil", withFake: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingProvider{}
			fileName := "a.go"
			if !testCase.write {
				fileName = ""
			}
			_, ctx := loadSession(t, fileName, "package x\n", func(sess *Session) {
				sess.providers = func(*astcache.Cache) []syntax.Provider {
					if testCase.withFake {
						return []syntax.Provider{recorder}
					}
					return nil
				}
			})

			if (ctx.Syntax != nil) != testCase.wantCache {
				t.Fatalf("Syntax non-nil = %v, want %v", ctx.Syntax != nil, testCase.wantCache)
			}
			_, found := ctx.Syntax.Get("a.go")
			if found != (len(testCase.wantParsed) > 0) {
				t.Fatalf("Get(a.go) found = %v", found)
			}
			if len(recorder.sources) != len(testCase.wantParsed) {
				t.Fatalf("provider sources = %v, want %v", recorder.sources, testCase.wantParsed)
			}
		})
	}
}

func TestSession_Load_DropsSkippedFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		file string
		want bool
	}{
		{name: "testdata file dropped from AST and syntax only", file: "pkg/testdata/x.go"},
		{name: "vendor file dropped from AST and syntax only", file: "vendor/m/x.go"},
		{name: "ordinary file kept", file: "pkg/x.go", want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, ctx := loadSession(t, testCase.file, "package x\n", func(sess *Session) {
				sess.providers = func(*astcache.Cache) []syntax.Provider { return nil }
			})
			if len(ctx.ChangedFiles) != 1 || ctx.ChangedFiles[0].Path != testCase.file {
				t.Fatalf("ChangedFiles = %v, want unfiltered %s", ctx.ChangedFiles, testCase.file)
			}
			if got := ctx.ASTCache != nil; got != testCase.want {
				t.Fatalf("ASTCache non-nil = %v, want %v", got, testCase.want)
			}
			if got := ctx.Syntax != nil; got != testCase.want {
				t.Fatalf("Syntax non-nil = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSession_Load_ProviderInputs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		file     string
		body     string
		wantTest bool
	}{
		{name: "plain source", file: "a.go", body: "package x\n"},
		{name: "test source", file: "a_test.go", body: "package x\n", wantTest: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingProvider{}
			var gotCache *astcache.Cache
			root, ctx := loadSession(t, testCase.file, testCase.body, func(sess *Session) {
				sess.providers = func(cache *astcache.Cache) []syntax.Provider {
					gotCache = cache
					return []syntax.Provider{recorder}
				}
			})
			assertLazyOnce(t, ctx, recorder)
			if gotCache == nil || gotCache != ctx.ASTCache {
				t.Fatalf("factory got cache %v, want the context's", gotCache)
			}
			assertProviderInputs(t, recorder, providerInputs{
				root: root, body: testCase.body, want: syntax.Source{
					Path: testCase.file, IsTest: testCase.wantTest,
				},
			})
		})
	}
}

func assertLazyOnce(t *testing.T, ctx *pass.Context, recorder *recordingProvider) {
	t.Helper()

	if recorder.calls != 0 {
		t.Fatalf("provider called %d times before any accessor", recorder.calls)
	}
	ctx.Syntax.Unavailable()
	if recorder.calls != 1 {
		t.Fatalf("provider called %d times, want 1", recorder.calls)
	}
}

type providerInputs struct {
	root string
	body string
	want syntax.Source
}

func assertProviderInputs(t *testing.T, recorder *recordingProvider, inputs providerInputs) {
	t.Helper()

	want := inputs.want
	want.Content = []byte(inputs.body)
	if len(recorder.sources) != 1 || !reflect.DeepEqual(recorder.sources[0], want) {
		t.Fatalf("sources = %+v, want %+v", recorder.sources, want)
	}
	if recorder.root != inputs.root {
		t.Fatalf("root = %q, want %q", recorder.root, inputs.root)
	}
	if got, ok := recorder.read(inputs.want.Path); !ok || string(got) != inputs.body {
		t.Fatalf("read(%s) = %q, %v", inputs.want.Path, got, ok)
	}
}

func TestSession_Load_ProviderModule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		goMod      string
		wantModule string
	}{
		{
			name: "module path of the repo root", goMod: "module example.com/app\n\ngo 1.25\n",
			wantModule: "example.com/app",
		},
		{name: "no go.mod gives no module"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingProvider{}
			_, ctx := loadSession(t, "a.go", "package x\n", func(sess *Session) {
				if testCase.goMod != "" {
					goModPath := filepath.Join(sess.repo.Root(), "go.mod")
					if err := os.WriteFile(goModPath, []byte(testCase.goMod), 0o600); err != nil {
						t.Fatalf("write go.mod: %v", err)
					}
				}
				sess.providers = func(*astcache.Cache) []syntax.Provider {
					return []syntax.Provider{recorder}
				}
			})
			ctx.Syntax.Unavailable()

			if recorder.module != testCase.wantModule {
				t.Fatalf("Corpus.Module() = %q, want %q", recorder.module, testCase.wantModule)
			}
		})
	}
}
