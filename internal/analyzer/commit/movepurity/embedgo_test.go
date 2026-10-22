package movepurity

import (
	"path"
	"testing"
)

func goAssetRow(name, directive, oldFile, newFile string) renameCase {
	row := assetRow(name, directive, oldFile, newFile)
	src := pkgFile("sub", "const K = 1\n")
	row.head[oldFile], row.staged[newFile] = src, src
	return row
}

func TestEmbedSet_GoAndCompiledFiles(t *testing.T) {
	t.Parallel()

	const static, nested = "//go:embed static", "internal/assets/static/sub/b.go"
	testRow := embedderAt(
		goAssetRow(
			"Go file under an embedded testdata directory moves to a sibling",
			"//go:embed testdata",
			"internal/assets/testdata/src/a/a.go", "internal/assets/testdata/src/b/a.go",
		),
		"internal/assets/embed_test.go", embedSrc("assets", "//go:embed testdata"),
	)
	testCases := []renameCase{
		goAssetRow("Go package leaves the embedded directory", static, nested, "internal/o/b.go"),
		goAssetRow("Go file enters the embedded directory", static, "internal/o/b.go", nested),
		goAssetRow(
			"Go file renamed inside the embedded directory",
			static, nested, "internal/assets/static/sub/c.go",
		),
		testRow,
		assetRow(
			"compiled file leaves the embedded directory",
			static, "internal/assets/static/x.h", "internal/other/x.h",
		),
		accepted(goAssetRow(
			"Go file under testdata that no pattern covers",
			static, "internal/assets/testdata/src/a/a.go", "internal/assets/testdata/src/b/a.go",
		)),
	}
	runRenameCases(t, testCases)
}

func clauseRow(name, directive, oldFile, newFile string, pkgs [2]string) renameCase {
	row := assetRow(name, directive, oldFile, newFile)
	sibling := path.Dir(newFile) + "/s.go"
	row.head[oldFile] = pkgFile(pkgs[0], "const K = 1\n")
	row.staged[newFile] = pkgFile(pkgs[1], "const K = 1\n")
	row.head[sibling], row.staged[sibling] = pkgFile(pkgs[1], ""), pkgFile(pkgs[1], "")
	return row
}

func TestEmbedSet_ClauseFixedGoFiles(t *testing.T) {
	t.Parallel()

	const static, nested = "//go:embed static", "internal/assets/static/sub/b.go"
	testRow := embedderAt(
		clauseRow(
			"analysistest layout, package a becomes package b under embedded testdata",
			"//go:embed testdata",
			"internal/assets/testdata/src/a/a.go", "internal/assets/testdata/src/b/a.go",
			[2]string{"a", "b"},
		),
		"internal/assets/embed_test.go", embedSrc("assets", "//go:embed testdata"),
	)
	testCases := []renameCase{
		clauseRow(
			"clause-fixed Go package leaves the embedded directory",
			static, nested, "internal/other/b.go", [2]string{"sub", "other"},
		),
		clauseRow(
			"clause-fixed Go file enters the embedded directory",
			static, "internal/other/b.go", nested, [2]string{"other", "sub"},
		),
		testRow,
		accepted(clauseRow(
			"clause-fixed Go file that no pattern covers",
			static, "internal/o/b.go", "internal/p/b.go", [2]string{"o", "p"},
		)),
	}
	runRenameCases(t, testCases)
}

func TestEmbedSet_EditedFilesKeepTheEditVerdict(t *testing.T) {
	t.Parallel()

	const leaving, entering = "internal/assets/static/x.h", "internal/other/x.h"
	asset := assetRow("compiled file edited as it leaves", "//go:embed static", leaving, entering)
	asset.staged[entering] = "edited\n"
	moves := newMoveContext(asset.repo(), asset.renames)
	if got := moves.renameVerdict(asset.renames[0]); got != verdictEdited {
		t.Fatalf("renameVerdict() = %v, want %v", got, verdictEdited)
	}
}
