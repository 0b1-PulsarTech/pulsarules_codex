//go:build unix

package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.ID() != AnalyzerID || analyzer.Stage() != core.StageStatic {
		t.Fatalf("ID %q Stage %v", analyzer.ID(), analyzer.Stage())
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := core.DelegatedCommand{
		ID: "first", Format: "dart-analyze-json", Run: []string{"no-such-tool-xyz"},
	}
	broken := core.DelegatedCommand{ID: "second", Format: "nope", Run: []string{"x"}}

	testCases := []struct {
		name   string
		config *core.AnalysisConfig
		want   []string
	}{
		{"nil config", nil, nil},
		{"nothing configured", &core.AnalysisConfig{}, nil},
		{
			"each command is reported on its own",
			&core.AnalysisConfig{Delegated: []core.DelegatedCommand{missing, broken}},
			[]string{"first/unavailable", "second/config"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{ProjectDir: root, Config: testCase.config}
			findings := NewAnalyzer(nil).Analyze(ctx)
			if len(findings) != len(testCase.want) {
				t.Fatalf("findings = %+v, want ids %v", findings, testCase.want)
			}
			for idx, finding := range findings {
				if finding.AnalyzerID != testCase.want[idx] {
					t.Errorf(
						"finding %d = %q, want %q",
						idx,
						finding.AnalyzerID,
						testCase.want[idx],
					)
				}
			}
		})
	}
}

func TestAnalyzeKeepsOnlyListedFiles(t *testing.T) {
	t.Parallel()

	root := aliasedTree(t)
	config := &core.AnalysisConfig{Delegated: []core.DelegatedCommand{{
		ID: entryID, Format: "dart-analyze-json",
		Run: reportOn(t, root, "build/out.dart", "web/src/a.dart"),
	}}}
	skips := sources.NewSkipSet(nil)
	testCases := []struct {
		name    string
		sources sources.Provider
		want    int
	}{
		{"no provider keeps every real file", nil, 2},
		{
			"provider lists one of them",
			sources.NewListedProvider(
				sources.NewProvider(root, skips), []string{"web/src/a.dart"}, skips,
			),
			1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{
				ProjectDir: root, Config: config, Sources: testCase.sources,
			}
			if got := len(NewAnalyzer(nil).Analyze(ctx)); got != testCase.want {
				t.Errorf("findings = %d, want %d", got, testCase.want)
			}
		})
	}
}

type walkCounter struct {
	walks *atomic.Int32
}

func (counter walkCounter) Read(string) ([]byte, bool) { return nil, false }

func (counter walkCounter) Walk(func(path, ext string) bool) { counter.walks.Add(1) }

func TestAnalyzeListsOnlyWhenACommandIsConfigured(t *testing.T) {
	t.Parallel()

	command := core.DelegatedCommand{
		ID: entryID, Format: "dart-analyze-json", Run: []string{"no-such-tool-xyz"},
	}
	testCases := []struct {
		name      string
		config    *core.AnalysisConfig
		wantWalks int32
	}{
		{"nil config", nil, 0},
		{"no delegated entry", &core.AnalysisConfig{}, 0},
		{"one delegated entry", &core.AnalysisConfig{
			Delegated: []core.DelegatedCommand{command},
		}, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var walks atomic.Int32
			ctx := &core.AnalysisContext{
				ProjectDir: t.TempDir(), Config: testCase.config,
				Sources: walkCounter{walks: &walks},
			}

			NewAnalyzer(nil).Analyze(ctx)

			if got := walks.Load(); got != testCase.wantWalks {
				t.Errorf("walks = %d, want %d", got, testCase.wantWalks)
			}
		})
	}
}

func scrubbedGitEnv() []string {
	var env []string
	for _, entry := range os.Environ() {
		switch {
		case strings.HasPrefix(entry, "GIT_DIR="),
			strings.HasPrefix(entry, "GIT_INDEX_FILE="),
			strings.HasPrefix(entry, "GIT_WORK_TREE="):
		default:
			env = append(env, entry)
		}
	}
	return env
}

func git(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...)
	cmd.Env = scrubbedGitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		if mkErr := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o700); mkErr != nil {
			t.Fatalf("mkdir: %v", mkErr)
		}
		if wrErr := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); wrErr != nil {
			t.Fatalf("write: %v", wrErr)
		}
	}
}

func stageUnmerged(t *testing.T, root string) {
	t.Helper()
	blob := strings.TrimSpace(string(git(t, root, "hash-object", "-w", "conflict.dart")))
	var lines strings.Builder
	for stage := 1; stage <= 3; stage++ {
		fmt.Fprintf(&lines, "100644 %s %d\tconflict.dart\n", blob, stage)
	}
	cmd := exec.CommandContext(t.Context(), "git", "-C", root, "update-index", "--index-info")
	cmd.Env, cmd.Stdin = scrubbedGitEnv(), strings.NewReader(lines.String())
	if err := cmd.Run(); err != nil {
		t.Fatalf("stage unmerged: %v", err)
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	outside := t.TempDir()
	writeTree(t, outside, map[string]string{"x.dart": "x"})
	writeTree(t, root, map[string]string{
		".gitignore": "build/\nignored_link\next_link\ninner_link\next_ign\n",
		"web/a.dart": "a", "new.dart": "n",
		"build/out.dart": "o", ".dart_tool/x.dart": "x", "tool/.dart_tool/z.dart": "z",
		"tracked_gone.dart": "g", "web/deep/b.dart": "b", "conflict.dart": "c",
		"sub/x.dart": "s",
	})
	links := map[string]string{
		"shared": "web", "untracked_link": "web", "toignored.dart": "build/out.dart",
		"linked.dart": "web/a.dart",
		"tobuild":     "build", "toouter": outside, "ignored_link": "web",
		"web/ext_link": "../new.dart", "web/inner_link": "deep/b.dart",
		"web/ext_ign": "../build/out.dart",
	}
	for link, target := range links {
		if lnErr := os.Symlink(target, filepath.Join(root, link)); lnErr != nil {
			t.Fatalf("symlink: %v", lnErr)
		}
	}
	tracked := []string{
		"web/a.dart", "web/deep/b.dart", "shared", "toignored.dart", "linked.dart",
		"tracked_gone.dart", "tobuild", "toouter",
	}
	git(t, root, "init", "-q")
	git(t, root, append([]string{"add"}, tracked...)...)
	stageUnmerged(t, root)
	const gitlink = "160000,0123456789012345678901234567890123456789,sub"
	git(t, root, "update-index", "--add", "--cacheinfo", gitlink)
	if rmErr := os.Remove(filepath.Join(root, "tracked_gone.dart")); rmErr != nil {
		t.Fatalf("remove tracked file: %v", rmErr)
	}
	return root
}

func listedPaths(t *testing.T, root string) []string {
	t.Helper()
	flags := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	output := git(t, root, flags...)
	var paths []string
	for entry := range strings.SplitSeq(string(output), "\x00") {
		stat, err := os.Lstat(filepath.Join(root, filepath.FromSlash(entry)))
		if entry != "" && err == nil && !stat.IsDir() {
			paths = append(paths, entry)
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

//nolint:ireturn // test helper hands back the interface the analyzer consumes
func engineSources(t *testing.T, root string) sources.Provider {
	t.Helper()
	paths := listedPaths(t, root)
	skips := sources.NewSkipSet(sources.GitSkipDirs())
	return sources.NewListedProvider(sources.NewProvider(root, skips), paths, skips)
}

func describe(findings []core.Finding) []string {
	lines := make([]string, 0, len(findings))
	for _, finding := range findings {
		where := finding.File
		if strings.HasSuffix(finding.AnalyzerID, "/config") {
			where = finding.Message
		}
		lines = append(lines, finding.AnalyzerID+" "+where)
	}
	return lines
}

type gitCase struct {
	name  string
	dir   string
	files []string
	want  []string
}

func gitCases() []gitCase {
	const everything = "web/a.dart new.dart build/out.dart gone.dart"

	return []gitCase{
		{
			"tracked and untracked kept, ignored excluded and missing dropped",
			"", strings.Fields(everything),
			[]string{entryID + "/c web/a.dart", entryID + "/c new.dart", entryID + "/dropped "},
		},
		{"only an ignored file is a quiet pass", "", []string{"build/out.dart"}, nil},
		{"only a skip dir file is a quiet pass", "", []string{".dart_tool/x.dart"}, nil},
		{
			"a tracked link to an ignored file is loud",
			"",
			[]string{"toignored.dart"},
			[]string{entryID + "/c toignored.dart"},
		},
		{
			"a tracked link to a tracked target reported alone is loud",
			"",
			[]string{"linked.dart"},
			[]string{entryID + "/c linked.dart"},
		},
		{
			"a tracked link beside its reported target is reported too",
			"",
			[]string{"web/a.dart", "linked.dart"},
			[]string{entryID + "/c web/a.dart", entryID + "/c linked.dart"},
		},
		{
			"only a missing file is a failure",
			"",
			[]string{"gone.dart"},
			[]string{entryID + "/failed "},
		},
		{"an ignored dir is refused", "build", []string{"build/out.dart"}, []string{
			entryID + "/config " + noFilesMessage("build"),
		}},
		{
			"a dir with only skip dir files is refused",
			"tool",
			[]string{"tool/.dart_tool/z.dart"},
			[]string{entryID + "/config " + noFilesMessage("tool")},
		},
	}
}

func TestAnalyzeOverARealGitRepo(t *testing.T) {
	t.Parallel()

	root := gitRepo(t)
	provider := engineSources(t, root)
	for _, testCase := range slices.Concat(gitCases(), dirLinkCases(), escapeCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := &core.AnalysisConfig{Delegated: []core.DelegatedCommand{{
				ID: entryID, Format: "dart-analyze-json", Dir: testCase.dir,
				Run: reportOn(t, root, testCase.files...),
			}}}
			ctx := &core.AnalysisContext{ProjectDir: root, Config: config, Sources: provider}

			got := describe(NewAnalyzer(nil).Analyze(ctx))

			if !slices.Equal(got, testCase.want) {
				t.Errorf("findings = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestScrubbedListingMatchesVCS(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"GIT_DIR", "GIT_INDEX_FILE", "GIT_WORK_TREE"} {
		if _, set := os.LookupEnv(name); set {
			t.Skipf("%s is set, as inside a git hook", name)
		}
	}
	root := gitRepo(t)
	repo, err := vcs.Open(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	lister, isLister := repo.(interface{ ListFiles() ([]string, error) })
	if !isLister {
		t.Fatal("repository does not list files")
	}

	want, err := lister.ListFiles()
	if err != nil {
		t.Fatalf("list files: %v", err)
	}

	got := listedPaths(t, root)
	if !slices.Equal(got, want) {
		t.Errorf("listing = %q, want %q", got, want)
	}
}

func dirLinkCases() []gitCase {
	return []gitCase{
		{
			"a file under a tracked dir link is reported at the link path",
			"",
			[]string{"shared/a.dart"},
			[]string{entryID + "/c shared/a.dart"},
		},
		{
			"a file deep under a tracked dir link is reported at the link path",
			"",
			[]string{"shared/deep/b.dart"},
			[]string{entryID + "/c shared/deep/b.dart"},
		},
		{
			"a file under a dir link to an ignored dir is loud",
			"",
			[]string{"tobuild/out.dart"},
			[]string{entryID + "/c tobuild/out.dart"},
		},
		{
			"a file under a dir link to an outside dir is loud",
			"",
			[]string{"toouter/x.dart"},
			[]string{entryID + "/c toouter/x.dart"},
		},
		{"a file under a missing path of a dir link is a failure", "", []string{
			"shared/none.dart",
		}, []string{entryID + "/failed "}},
		{"ignored dir link", "", []string{
			"ignored_link/a.dart",
		}, nil},
		{
			"an untracked dir link that is not ignored is reported",
			"",
			[]string{"untracked_link/a.dart"},
			[]string{entryID + "/c untracked_link/a.dart"},
		},
		{"a file under a gitlink is a quiet pass", "", []string{"sub/x.dart"}, nil},
		{"a gitlink dir is refused", "sub", []string{"sub/x.dart"}, []string{
			entryID + "/config " + noFilesMessage("sub"),
		}},
	}
}

func escapeCases() []gitCase {
	return []gitCase{
		{
			"an ignored link out of the entry dir to a listed file is reported",
			"web",
			[]string{"web/ext_link"},
			[]string{entryID + "/c web/ext_link"},
		},
		{
			"an ignored link out of the entry dir to an unlisted file stays a quiet pass",
			"web",
			[]string{"web/ext_ign"},
			nil,
		},
		{
			"an ignored link inside the entry dir stays a quiet pass",
			"web",
			[]string{"web/inner_link"},
			nil,
		},
	}
}
