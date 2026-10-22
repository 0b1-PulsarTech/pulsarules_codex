package movepurity

import (
	"path"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestEmbedSet_StaysPure(t *testing.T) {
	t.Parallel()

	const static = "//go:embed static"
	testCases := []renameCase{
		accepted(assetRow(
			"unrelated directory",
			static, "internal/other/a.png", "internal/elsewhere/a.png",
		)),
		accepted(assetRow(
			"file beside the embedder that no pattern names",
			"//go:embed public.txt",
			"internal/assets/other.png",
			"internal/assets/img/other.png",
		)),
		accepted(assetRow(
			"hidden file under a plain directory pattern",
			static, "internal/assets/static/.keep", "internal/other/keep",
		)),
		accepted(assetRow(
			"file under a hidden directory below a plain directory pattern",
			static, "internal/assets/static/_gen/a.txt", "internal/other/a.txt",
		)),
		accepted(assetRow(
			"testdata file no pattern embeds",
			static, "internal/assets/testdata/a.json", "internal/assets/testdata/sub/a.json",
		)),
		accepted(assetRow(
			"documentation beside the root embedder",
			"//go:embed templates", "docs/a.md", "docs/guide/a.md",
		)),
		accepted(assetRow(
			"sibling package assets",
			static, "internal/sibling/static/x.png", "internal/sibling/assets/x.png",
		)),
		accepted(assetRow(
			"same-directory rename of a file no pattern covers",
			"//go:embed *.txt", "internal/assets/a.png", "internal/assets/b.png",
		)),
		accepted(assetRow(
			"same-directory rename of a hidden file under a plain directory pattern",
			static, "internal/assets/static/.keep", "internal/assets/static/.keep2",
		)),
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_Detected(t *testing.T) {
	t.Parallel()

	const static, nested = "//go:embed static", "internal/assets/static/css/a.css"
	testCases := []renameCase{
		assetRow(
			"asset leaves the embedded directory",
			static, "internal/assets/static/logo.png", "internal/other/logo.png",
		),
		assetRow(
			"asset enters the embedded directory",
			static, "internal/other/logo.png", "internal/assets/static/logo.png",
		),
		assetRow(
			"asset moves within the embedded directory",
			static, "internal/assets/static/a.png", "internal/assets/static/img/a.png",
		),
		assetRow(
			"glob pattern, file leaves",
			"//go:embed *.txt", "internal/assets/a.txt", "internal/other/a.txt",
		),
		assetRow(
			"glob pattern, file enters",
			"//go:embed *.txt", "internal/other/a.txt", "internal/assets/a.txt",
		),
		assetRow(
			"literal file pattern",
			"//go:embed public.txt", "internal/assets/public.txt", "internal/other/public.txt",
		),
		assetRow(
			"nested file under a directory pattern",
			static, nested, "internal/other/a.css",
		),
		assetRow(
			"all: pattern covers a hidden file",
			"//go:embed all:static", "internal/assets/static/.keep", "internal/other/keep",
		),
		assetRow(
			"all: pattern covers a hidden directory",
			"//go:embed all:static", "internal/assets/static/_gen/a.txt", "internal/other/a.txt",
		),
		assetRow(
			"glob names a hidden file",
			"//go:embed static/*", "internal/assets/static/.keep", "internal/other/keep",
		),
		assetRow(
			"explicit hidden file name",
			"//go:embed static/.keep", "internal/assets/static/.keep", "internal/other/keep",
		),
		assetRow(
			"hidden file entering an all: pattern",
			"//go:embed all:static", "internal/other/keep", "internal/assets/static/.keep",
		),
		assetRow(
			"second directive line covers the file",
			"//go:embed public.txt\n//go:embed static",
			"internal/assets/static/a.png", "internal/other/a.png",
		),
		grandparentRow(),
		renamedEmbedderRow(),
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_DetectedBeyondPackage(t *testing.T) {
	t.Parallel()

	const static = "//go:embed static"
	testCases := []renameCase{
		assetRow(
			"testdata glob, file moves into a subdirectory",
			"//go:embed testdata/*.json",
			"internal/assets/testdata/a.json", "internal/assets/testdata/sub/a.json",
		),
		testEmbedderRow(
			"testdata directory embedded by a test file, file leaves",
			"//go:embed testdata",
			"internal/assets/testdata/a.json", "internal/other/testdata/a.json",
		),
		assetRow(
			"all: pattern on an underscore directory, file moves deeper",
			"//go:embed all:_assets",
			"internal/assets/_assets/x", "internal/assets/_assets/y/x",
		),
		assetRow(
			"hidden file leaves into a hidden name under all:",
			"//go:embed all:static", "internal/assets/static/.keep", "internal/other/.keep",
		),
		assetRow(
			"hidden file leaves into a hidden name under a glob",
			"//go:embed static/*", "internal/assets/static/.env", "internal/other/.env",
		),
		assetRow(
			"hidden file leaves into a hidden name under an explicit name",
			"//go:embed static/.keep", "internal/assets/static/.keep", "internal/other/.keep",
		),
		assetRow(
			"underscore file leaves into an underscore name under all:",
			"//go:embed all:static", "internal/assets/static/_x.txt", "internal/other/_x.txt",
		),
		assetRow(
			"same-directory rename changes what a glob includes",
			"//go:embed *.txt", "internal/assets/a.txt", "internal/assets/a.md",
		),
		assetRow(
			"same-directory rename inside an embedded directory",
			static, "internal/assets/static/a.css", "internal/assets/static/b.css",
		),
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_GuardsFailClosed(t *testing.T) {
	t.Parallel()

	const static, leaving = "//go:embed static", "internal/assets/static/logo.png"
	row := func(name string) renameCase {
		return assetRow(name, static, leaving, "internal/other/logo.png")
	}
	unreadable := row("embedder blob cannot be read")
	unreadable.failBlob = "internal/assets/embed.go"
	broken := row("embedder does not parse")
	broken = embedderAt(broken, "internal/assets/embed.go", embedSrc("assets", static)+"func (\n")
	testCases := []renameCase{
		failingHeadDir(row("embedder directory cannot be listed"), "internal/assets"),
		unreadable,
		broken,
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_SkipsEmbeddersNeverCompiled(t *testing.T) {
	t.Parallel()

	const static = "//go:embed static"
	row := func(name, file, src string) renameCase {
		leaving := path.Dir(file) + "/static/logo.png"
		base := assetRow(name, static, leaving, "internal/other/logo.png")
		return accepted(embedderAt(base, file, src))
	}
	testCases := []renameCase{
		row("build-ignored embedder", "internal/assets/embed.go",
			"//go:build ignore\n\n"+embedSrc("assets", static)),
		row("embedder under testdata", "internal/assets/testdata/gen.go",
			embedSrc("gen", static)),
		row("embedder under an underscore directory", "internal/assets/_gen/gen.go",
			embedSrc("gen", static)),
		row("embedder with an underscore file name", "internal/assets/_gen.go",
			embedSrc("assets", static)),
		row("embedder with a dot file name", "internal/assets/.gen.go",
			embedSrc("assets", static)),
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_NotCovered(t *testing.T) {
	t.Parallel()

	const embedderGo = "internal/assets/embed.go"
	moved := assetRow(
		"asset moves with its embedding package",
		"//go:embed static",
		"internal/assets/static/a.png", "internal/assets2/static/a.png",
	)
	moved.head["internal/assets/embed.go"] = embedSrc("assets", "//go:embed static")
	moved.staged["internal/assets2/embed.go"] = embedSrc("assets", "//go:embed static")
	delete(moved.staged, embedderGo)
	moved.renames = append(moved.renames, pair(embedderGo, "internal/assets2/embed.go"))
	nested := assetRow(
		"embedder above a nested module boundary",
		"//go:embed sub", "internal/assets/sub/x/a.png", "internal/assets/sub/y/a.png",
	)
	nested.head["internal/assets/sub/go.mod"] = "module example.com/sub\n"
	nested.staged["internal/assets/sub/go.mod"] = "module example.com/sub\n"
	left := embedRow("relocation leaves a nested module behind", "//go:embed static", []pass.Rename{
		pair("internal/util/static/x.js", "internal/routing/static/x.js"),
		pair("internal/util/static/y.js", "internal/routing/static/y.js"),
	})
	left = withFile(left, "internal/util/static/sub/go.mod", "")
	left = withFile(left, "internal/util/static/sub/z.js", "")
	whole := embedRow(
		"relocation takes a nested module along",
		"//go:embed static",
		left.renames[1:],
	)
	whole = withFile(whole, "internal/util/static/sub/go.mod", "internal/routing/static/sub/go.mod")
	whole = withFile(whole, "internal/util/static/sub/z.js", "internal/routing/static/sub/z.js")
	testCases := []renameCase{
		accepted(
			moved,
		),
		accepted(nested),
		accepted(movesToRootRow()),
		accepted(left),
		accepted(whole),
	}
	runRenameCases(t, testCases)
}

func TestEmbedRelocation_SameFile(t *testing.T) {
	t.Parallel()

	const embedPublic, embedStatic = "//go:embed public.txt", "//go:embed static"
	const ownText, ownJS = "internal/routing/public.txt", "internal/routing/static/x.js"
	public := []pass.Rename{pair("internal/util/public.txt", ownText)}
	static := []pass.Rename{
		pair("internal/util/static/x.js", ownJS),
		pair("internal/util/static/y.js", "internal/routing/static/y.js"),
	}
	dir := func(name string, comoved []pass.Rename) renameCase {
		return embedRow(name, embedStatic, comoved)
	}
	testCases := []renameCase{
		withFile(
			embedRow("own file in new dir, glob pattern", "//go:embed *.txt", nil),
			ownText, ownText,
		),
		withFile(dir("own file in new dir, directory pattern", static[1:]), ownJS, ownJS),
		withFile(
			dir("different file renamed in under a kept name", static[1:]),
			"internal/other/x.js", ownJS,
		),
		withFile(
			embedRow("same count, different names", "//go:embed *.txt", nil),
			"internal/routing/other.txt", "internal/routing/other.txt",
		),
		accepted(withFile(
			dir("covered file of the new dir moves away", static),
			"internal/routing/static/z.js", "internal/other/z.js",
		)),
		accepted(withFile(
			embedRow("own file swapped out by the arriving one", embedPublic, public),
			"internal/routing/public.txt", "internal/other/public.txt",
		)),
	}
	runRenameCases(t, testCases)
}

type findingCase struct {
	renameCase
	wantMessage string
	wantAdvice  string
}

func embedFindingCases() []findingCase {
	const lead = "changes the files a //go:embed pattern in "
	const goAdvice = "move every file the pattern covers in the same commit, " +
		"and keep the new location free of other files"
	const assetAdvice = "commit this rename on its own and say in its message " +
		"which embedded files it adds or removes"
	left := embedRow("embedded file stays behind", "//go:embed public.txt", nil)
	identical := renameCase{
		name: "byte-identical Go file leaves its embedded file behind",
		head: map[string]string{
			"a/embed.go": embedSrc("assets", "//go:embed x.txt"),
			"a/x.txt":    "x\n",
		},
		staged: map[string]string{
			"b/embed.go": embedSrc("assets", "//go:embed x.txt"),
			"a/x.txt":    "x\n",
		},
		renames: []pass.Rename{{OldPath: "a/embed.go", NewPath: "b/embed.go", Score: 100}},
	}
	asset := assetRow(
		"asset leaves the embedded directory",
		"//go:embed static", "internal/assets/static/logo.png", "internal/other/logo.png",
	)
	goFile := goAssetRow(
		"Go file leaves the embedded directory",
		"//go:embed static", "internal/assets/static/sub/b.go", "internal/other/b.go",
	)
	compiled := assetRow(
		"compiled file leaves the embedded directory",
		"//go:embed static", "internal/assets/static/x.h", "internal/other/x.h",
	)
	return []findingCase{
		{
			left,
			"move of internal/util/a.go -> internal/routing/a.go " + lead + "internal/routing includes",
			goAdvice,
		},
		{identical, "move of a/embed.go -> b/embed.go " + lead + "b includes", goAdvice},
		{
			asset,
			"move of internal/assets/static/logo.png -> internal/other/logo.png " + lead +
				"internal/assets includes",
			assetAdvice,
		},
		{
			goFile,
			"move of internal/assets/static/sub/b.go -> internal/other/b.go " + lead +
				"internal/assets includes",
			assetAdvice,
		},
		{
			compiled,
			"move of internal/assets/static/x.h -> internal/other/x.h " + lead +
				"internal/assets includes",
			assetAdvice,
		},
		{
			rootAssetRow(),
			"move of static/logo.png -> other/logo.png " + lead + "the module root includes",
			assetAdvice,
		},
		{
			landsAtRootRow(),
			"move of a/embed.go -> embed.go " + lead + "the module root includes",
			goAdvice,
		},
	}
}

func unreadFindingCases() []findingCase {
	const static, leaving = "//go:embed static", "internal/assets/static/logo.png"
	const message = "move of " + leaving + " -> internal/other/logo.png may change the files " +
		"a //go:embed pattern in internal/assets includes: "
	const advice = "fix that file, or commit this rename on its own"
	row := func(name string) renameCase {
		return assetRow(name, static, leaving, "internal/other/logo.png")
	}
	unreadable := row("embedder blob cannot be read")
	unreadable.failBlob = "internal/assets/embed.go"
	broken := embedderAt(
		row("embedder does not parse"),
		"internal/assets/embed.go", embedSrc("assets", static)+"func (\n",
	)
	entering := embedderAt(
		assetRow("asset enters beside an embedder that does not parse", static,
			"internal/other/logo.png", leaving),
		"internal/assets/embed.go", embedSrc("assets", static)+"func (\n",
	)
	return []findingCase{
		{unreadable, message + "internal/assets/embed.go could not be read or parsed", advice},
		{broken, message + "internal/assets/embed.go could not be read or parsed", advice},
		{
			entering,
			"move of internal/other/logo.png -> " + leaving + " may change the files " +
				"a //go:embed pattern in internal/assets includes: " +
				"internal/assets/embed.go could not be read or parsed", advice,
		},
		{
			failingHeadDir(row("directory cannot be listed"), "internal/assets"),
			message + "internal/assets/ could not be read or parsed", advice,
		},
	}
}

func TestEmbedSet_Findings(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(embedFindingCases(), unreadFindingCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analyzer := NewAnalyzer(testCase.repo())
			findings := analyzer.Analyze(&pass.Context{StagedRenames: testCase.renames})
			if len(findings) != 1 {
				t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
			}
			if got := findings[0]; got.Message != testCase.wantMessage ||
				got.Suggestion != testCase.wantAdvice {
				t.Fatalf(
					"got message %q suggestion %q, want %q and %q",
					got.Message, got.Suggestion, testCase.wantMessage, testCase.wantAdvice,
				)
			}
		})
	}
}
