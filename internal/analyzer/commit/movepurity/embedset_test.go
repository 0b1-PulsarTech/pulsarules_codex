package movepurity

import (
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
	}
	runRenameCases(t, testCases)
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

func TestEmbedSet_Findings(t *testing.T) {
	t.Parallel()

	const lead = "changes the files a //go:embed pattern in "
	const goAdvice = "move every file the pattern covers in the same commit, " +
		"and keep the new location free of other files"
	const assetAdvice = "keep the file where the pattern finds it, " +
		"or commit the move on its own with the embed pattern updated"
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
