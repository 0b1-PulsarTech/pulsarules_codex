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
