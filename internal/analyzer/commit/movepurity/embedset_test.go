package movepurity

import (
	"path"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func assetRow(name, directive, oldFile, newFile string) renameCase {
	const embedderGo = "internal/assets/embed.go"
	return renameCase{
		name: name,
		head: map[string]string{
			embedderGo: embedSrc("assets", directive),
			oldFile:    "asset\n",
		},
		staged: map[string]string{
			embedderGo: embedSrc("assets", directive),
			newFile:    "asset\n",
		},
		renames: []pass.Rename{pair(oldFile, newFile)},
	}
}

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

func testEmbedderRow(name, directive, oldFile, newFile string) renameCase {
	row := assetRow(name, directive, oldFile, newFile)
	return embedderAt(row, "internal/assets/embed_test.go", embedSrc("assets", directive))
}

func embedderAt(row renameCase, file, src string) renameCase {
	for _, tree := range []map[string]string{row.head, row.staged} {
		delete(tree, "internal/assets/embed.go")
		tree[file] = src
	}
	return row
}

func grandparentRow() renameCase {
	const embedder, asset = "internal/web/embed.go", "internal/web/dist/css/a.css"
	row := assetRow("pattern in a grandparent directory", "//go:embed dist", asset, "x/a.css")
	row.head = map[string]string{embedder: embedSrc("web", "//go:embed dist"), asset: "asset\n"}
	row.staged = map[string]string{
		embedder:  embedSrc("web", "//go:embed dist"),
		"x/a.css": "asset\n",
	}
	return row
}

func renamedEmbedderRow() renameCase {
	const oldGo, newGo = "internal/web/a.go", "internal/web/b.go"
	const asset, moved = "internal/web/static/x.txt", "x/x.txt"
	src := embedSrc("web", "//go:embed static")
	return renameCase{
		name:    "embedder renamed inside its directory still embeds there",
		head:    map[string]string{oldGo: src, asset: "x\n", "internal/web/static/y.txt": "y\n"},
		staged:  map[string]string{newGo: src, moved: "x\n", "internal/web/static/y.txt": "y\n"},
		renames: []pass.Rename{pair(asset, moved), {OldPath: oldGo, NewPath: newGo, Score: 100}},
	}
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
	testCases := []renameCase{accepted(moved), accepted(nested)}
	runRenameCases(t, testCases)
}

func rootAssetRow() renameCase {
	row := assetRow(
		"asset leaves a directory embedded at the module root",
		"//go:embed static", "static/logo.png", "other/logo.png",
	)
	row.head = map[string]string{
		"embed.go": embedSrc("assets", "//go:embed static"), "static/logo.png": "asset\n",
	}
	row.staged = map[string]string{
		"embed.go": embedSrc("assets", "//go:embed static"), "other/logo.png": "asset\n",
	}
	return row
}

func landsAtRootRow() renameCase {
	return renameCase{
		name: "byte-identical Go file lands at the module root",
		head: map[string]string{
			"a/embed.go": embedSrc("assets", "//go:embed x.txt"),
			"a/x.txt":    "x\n",
		},
		staged: map[string]string{
			"embed.go": embedSrc("assets", "//go:embed x.txt"),
			"a/x.txt":  "x\n",
		},
		renames: []pass.Rename{{OldPath: "a/embed.go", NewPath: "embed.go", Score: 100}},
	}
}

func TestEmbedSet_Findings(t *testing.T) {
	t.Parallel()

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
	testCases := []struct {
		renameCase
		wantMessage string
		wantAdvice  string
	}{
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analyzer := NewAnalyzer(newFakeRepo(testCase.head, testCase.staged))
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
