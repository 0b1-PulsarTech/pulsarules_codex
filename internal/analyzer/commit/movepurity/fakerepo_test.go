package movepurity

import (
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type fakeRepo struct {
	head        map[string]string
	staged      map[string]string
	stagedModes map[string]string
	failHeadDir string
	failBlob    string
}

var _ gitReader = fakeRepo{}

func newFakeRepo(head, staged map[string]string) fakeRepo {
	const goMod = "module example.com/m\n\ngo 1.25\n"
	fake := fakeRepo{head: maps.Clone(head), staged: maps.Clone(staged)}
	for _, tree := range []map[string]string{fake.head, fake.staged} {
		if _, ok := tree["go.mod"]; !ok {
			tree["go.mod"] = goMod
		}
	}
	return fake
}

func (repo fakeRepo) tree(side vcs.Side) map[string]string {
	if side == vcs.Staged {
		return repo.staged
	}
	return repo.head
}

func (repo fakeRepo) Blob(side vcs.Side, file string) (string, error) {
	if side == vcs.Head && file == repo.failBlob {
		return "", errMissing
	}
	return blobOf(repo.tree(side), file)
}

func (repo fakeRepo) Entry(side vcs.Side, file string) (mode, id string, err error) {
	content, readErr := blobOf(repo.tree(side), file)
	mode = "100644"
	if override, ok := repo.stagedModes[file]; ok && side == vcs.Staged {
		mode = override
	}
	return mode, content, readErr
}

func (repo fakeRepo) Dir(side vcs.Side, dir string) ([]string, error) {
	tree := repo.tree(side)
	if side == vcs.Head && dir == repo.failHeadDir {
		return nil, errMissing
	}
	if side == vcs.Head {
		return childrenOf(tree, dir), nil
	}
	return dirOf(tree, dir), nil
}

func blobOf(tree map[string]string, file string) (string, error) {
	content, ok := tree[file]
	if !ok {
		return "", errMissing
	}
	return content, nil
}

func dirOf(tree map[string]string, dir string) []string {
	return slices.DeleteFunc(slices.Sorted(maps.Keys(tree)), func(name string) bool {
		return path.Dir(name) != dir
	})
}

func childrenOf(tree map[string]string, dir string) []string {
	prefix := dir + "/"
	if dir == "." {
		prefix = ""
	}
	names := map[string]bool{}
	for name := range tree {
		if rest, isUnder := strings.CutPrefix(name, prefix); isUnder {
			first, _, _ := strings.Cut(rest, "/")
			names[prefix+first] = true
		}
	}
	return slices.Sorted(maps.Keys(names))
}

type emptyRepo struct{ fakeRepo }

func (repo emptyRepo) Blob(side vcs.Side, file string) (string, error) {
	if side == vcs.Head {
		return "", errMissing
	}
	return repo.fakeRepo.Blob(side, file)
}

type noHeadDirRepo struct{ fakeRepo }

func (repo noHeadDirRepo) Dir(side vcs.Side, dir string) ([]string, error) {
	if side == vcs.Head {
		return nil, errMissing
	}
	return repo.fakeRepo.Dir(side, dir)
}

type noDirRepo struct{ fakeRepo }

func (repo noDirRepo) Dir(side vcs.Side, dir string) ([]string, error) {
	if side == vcs.Staged {
		return nil, errMissing
	}
	return repo.fakeRepo.Dir(side, dir)
}

func embedRow(name, directive string, comoved []pass.Rename) renameCase {
	const oldGo, newGo = "internal/util/a.go", "internal/routing/a.go"
	staged := map[string]string{
		newGo:                         embedSrc("routing", directive),
		routingMain:                   routingOther,
		"internal/routing/public.txt": "secret\n",
	}
	if len(comoved) > 0 {
		staged["internal/routing/public.txt"] = "public\n"
	}
	return renameCase{
		name: name,
		head: map[string]string{
			oldGo:                       embedSrc("util", directive),
			"internal/util/public.txt":  "public\n",
			"internal/util/static/x.js": "x\n",
			"internal/util/static/y.js": "y\n",
		},
		staged:  staged,
		renames: append([]pass.Rename{pair(oldGo, newGo)}, comoved...),
	}
}

func accepted(row renameCase) renameCase {
	row.want = true
	return row
}

func failingHeadDir(row renameCase, dir string) renameCase {
	row.failHeadDir = dir
	return row
}

func withFile(row renameCase, oldPath, newPath string) renameCase {
	if oldPath != "" {
		row.head[oldPath] = "z\n"
	}
	if newPath != "" {
		row.staged[newPath] = "z\n"
	}
	if oldPath != "" && newPath != "" && oldPath != newPath {
		row.renames = append(row.renames, pair(oldPath, newPath))
	}
	return row
}

func embedLiteralNamed(pattern string) renameCase {
	oldFile, newFile := "internal/util/"+pattern, "internal/routing/"+pattern
	renamed := embedRow("glob lookalike "+pattern, "//go:embed "+pattern, nil)
	renamed.head[oldFile] = "x\n"
	renamed.staged[newFile] = "x\n"
	renamed.renames = append(renamed.renames, pair(oldFile, newFile))
	return renamed
}

func embedSrc(pkg, directive string) string {
	return pkgFile(pkg, "import _ \"embed\"\n\n"+directive+"\nvar Data string\n")
}

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

func movesToRootRow() renameCase {
	row := landsAtRootRow()
	row.name = "package and its embedded file move to the module root"
	row.staged["x.txt"] = "x\n"
	delete(row.staged, "a/x.txt")
	row.renames = append(row.renames, pair("a/x.txt", "x.txt"))
	return row
}
