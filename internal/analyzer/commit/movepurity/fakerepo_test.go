package movepurity

import (
	"maps"
	"path"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type fakeRepo struct {
	head        map[string]string
	staged      map[string]string
	stagedModes map[string]string
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

func (repo fakeRepo) Blob(side vcs.Side, path string) (string, error) {
	return blobOf(repo.tree(side), path)
}

func (repo fakeRepo) Entry(side vcs.Side, path string) (mode, id string, err error) {
	content, readErr := blobOf(repo.tree(side), path)
	mode = "100644"
	if override, ok := repo.stagedModes[path]; ok && side == vcs.Staged {
		mode = override
	}
	return mode, content, readErr
}

func (repo fakeRepo) Dir(side vcs.Side, dir string) ([]string, error) {
	return dirOf(repo.tree(side), dir), nil
}

func blobOf(tree map[string]string, path string) (string, error) {
	content, ok := tree[path]
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

type emptyRepo struct{ fakeRepo }

func (repo emptyRepo) Blob(side vcs.Side, path string) (string, error) {
	if side == vcs.Head {
		return "", errMissing
	}
	return repo.fakeRepo.Blob(side, path)
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
